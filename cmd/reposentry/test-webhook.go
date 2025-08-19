package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/johnnynv/RepoSentry/internal/trigger"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

var testWebhookCmd = &cobra.Command{
	Use:   "test-webhook",
	Short: "Test webhook sending to Tekton EventListener",
	Long:  "Send a test event to Tekton EventListener to verify integration",
	RunE:  runTestWebhook,
}

var (
	tektonURL    string
	tektonNS     string
	testRepo     string
	testBranch   string
	testCommit   string
	dryRun       bool
)

func init() {
	testWebhookCmd.Flags().StringVar(&tektonURL, "tekton-url", "", "Tekton EventListener URL")
	testWebhookCmd.Flags().StringVar(&tektonNS, "tekton-namespace", "tekton-pipelines", "Tekton namespace")
	testWebhookCmd.Flags().StringVar(&testRepo, "repo", "reposentry/test-repo", "Test repository name")
	testWebhookCmd.Flags().StringVar(&testBranch, "branch", "main", "Test branch name")
	testWebhookCmd.Flags().StringVar(&testCommit, "commit", "abcd1234567890abcdef1234567890abcdef1234", "Test commit SHA")
	testWebhookCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show payload, don't send")
	
	testWebhookCmd.MarkFlagRequired("tekton-url")
	rootCmd.AddCommand(testWebhookCmd)
}

func runTestWebhook(cmd *cobra.Command, args []string) error {
	// Initialize logger
	appLogger := logger.GetDefaultLogger()
	
	appLogger.WithFields(logger.Fields{
		"tekton_url": tektonURL,
		"namespace":  tektonNS,
		"repository": testRepo,
		"branch":     testBranch,
		"commit":     testCommit,
		"dry_run":    dryRun,
	}).Info("Starting Tekton webhook test")

	// Create test event
	event := types.Event{
		ID:         fmt.Sprintf("test_%d", time.Now().Unix()),
		Type:       types.EventTypeBranchUpdated,
		Repository: testRepo,
		Branch:     testBranch,
		CommitSHA:  testCommit,
		PrevCommit: "prev1234567890abcdef1234567890abcdef123456",
		Provider:   "github",
		Timestamp:  time.Now(),
		Status:     types.EventStatusPending,
		Metadata: map[string]string{
			"repository_url": fmt.Sprintf("https://github.com/%s", testRepo),
			"branch_ref":     fmt.Sprintf("refs/heads/%s", testBranch),
			"pusher_name":    "reposentry-test",
			"pusher_email":   "test@reposentry.local",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create Tekton trigger configuration
	config := trigger.TriggerConfig{
		Type:    "tekton",
		Enabled: true,
		Timeout: 30 * time.Second,
		Tekton: trigger.TektonConfig{
			EventListenerURL: tektonURL,
			Namespace:        tektonNS,
			Headers: map[string]string{
				"X-GitHub-Event": "push",
				"User-Agent":     "RepoSentry-Test/1.0",
			},
		},
	}

	// Create transformer and generate payload
	transformer := trigger.NewEventTransformer()
	payload, err := transformer.TransformToTekton(event)
	if err != nil {
		return fmt.Errorf("failed to transform event to Tekton payload: %w", err)
	}

	// Pretty print the payload
	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Println("🎯 Generated Tekton Payload:")
	fmt.Println(string(payloadJSON))
	fmt.Println()

	if dryRun {
		appLogger.Info("Dry run mode - not sending actual request")
		return nil
	}

	// Create Tekton trigger
	tektonTrigger, err := trigger.NewTektonTrigger(config)
	if err != nil {
		return fmt.Errorf("failed to create Tekton trigger: %w", err)
	}
	defer tektonTrigger.Close()

	// Send the event
	appLogger.Info("🚀 Sending webhook to Tekton EventListener...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := tektonTrigger.SendEvent(ctx, event)
	if err != nil {
		appLogger.WithFields(logger.Fields{
			"error": err.Error(),
		}).Error("❌ Failed to send webhook")
		return fmt.Errorf("webhook send failed: %w", err)
	}

	// Display results
	fmt.Printf("✅ Webhook sent successfully!\n\n")
	fmt.Printf("📊 Results:\n")
	fmt.Printf("  Event ID: %s\n", result.EventID)
	fmt.Printf("  Status Code: %d\n", result.StatusCode)
	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Duration: %v\n", result.Duration)
	fmt.Printf("  Response: %s\n", result.ResponseBody)
	
	if len(result.Metadata) > 0 {
		fmt.Printf("  Metadata:\n")
		for key, value := range result.Metadata {
			fmt.Printf("    %s: %s\n", key, value)
		}
	}

	appLogger.WithFields(logger.Fields{
		"event_id":    result.EventID,
		"status_code": result.StatusCode,
		"success":     result.Success,
		"duration":    result.Duration,
	}).Info("🎉 Tekton webhook test completed")

	fmt.Println("\n🔍 Check Tekton Dashboard for new PipelineRun:")
	fmt.Println("   kubectl get pipelinerun -n tekton-pipelines --sort-by=.metadata.creationTimestamp")
	fmt.Printf("   Expected name pattern: pytest-run-%s\n", payload.ShortSHA)

	return nil
}
