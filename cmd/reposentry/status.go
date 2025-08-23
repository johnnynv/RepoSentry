package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show RepoSentry status and health",
	Long:  "Display current status, health, and metrics of a running RepoSentry instance",
	RunE:  runStatus,
}

var (
	statusPort   int
	statusFormat string
	statusWatch  bool
	statusHost   string
)

func init() {
	statusCmd.Flags().IntVar(&statusPort, "port", 8080, "RepoSentry health check port")
	statusCmd.Flags().StringVar(&statusHost, "host", "localhost", "RepoSentry host")
	statusCmd.Flags().StringVar(&statusFormat, "format", "text", "Output format (text, json)")
	statusCmd.Flags().BoolVar(&statusWatch, "watch", false, "Watch mode - continuously show status")

	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusWatch {
		return runStatusWatch()
	}

	return runStatusOnce()
}

func runStatusOnce() error {
	baseURL := fmt.Sprintf("http://%s:%d", statusHost, statusPort)

	// Get health status
	health, err := getHealthStatus(baseURL)
	if err != nil {
		if statusFormat == "json" {
			result := map[string]interface{}{
				"status":  "unreachable",
				"error":   err.Error(),
				"healthy": false,
			}
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Printf("❌ RepoSentry is not reachable\n")
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("\n💡 Ensure RepoSentry is running on %s\n", baseURL)
		}
		return fmt.Errorf("service unreachable: %w", err)
	}

	// Get system status
	systemStatus, err := getSystemStatus(baseURL)
	if err != nil {
		systemStatus = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	// Get metrics
	metrics, err := getMetrics(baseURL)
	if err != nil {
		metrics = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	// Print results
	if statusFormat == "json" {
		return printStatusJSON(health, systemStatus, metrics)
	}

	return printStatusText(health, systemStatus, metrics)
}

func runStatusWatch() error {
	fmt.Printf("👀 Watching RepoSentry status (Ctrl+C to stop)\n\n")

	for {
		// Clear screen
		fmt.Print("\033[2J\033[H")

		fmt.Printf("🕐 %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

		if err := runStatusOnce(); err != nil {
			fmt.Printf("\nRetrying in 5 seconds...\n")
		}

		time.Sleep(5 * time.Second)
	}
}

func getHealthStatus(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func getSystemStatus(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func getMetrics(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func printStatusJSON(health, systemStatus, metrics map[string]interface{}) error {
	combined := map[string]interface{}{
		"health":  health,
		"system":  systemStatus,
		"metrics": metrics,
	}

	jsonBytes, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func printStatusText(health, systemStatus, metrics map[string]interface{}) error {
	// Print health status
	fmt.Printf("🏥 Health Status\n")
	fmt.Printf("================\n")

	if data, ok := health["data"].(map[string]interface{}); ok {
		if healthy, ok := data["healthy"].(bool); ok {
			if healthy {
				fmt.Printf("Status: ✅ Healthy\n")
			} else {
				fmt.Printf("Status: ❌ Unhealthy\n")
			}
		}

		if components, ok := data["components"].(map[string]interface{}); ok {
			fmt.Printf("Components:\n")
			for name, comp := range components {
				if compData, ok := comp.(map[string]interface{}); ok {
					status := compData["status"]
					fmt.Printf("  - %-12s %s\n", name+":", formatHealthStatus(status))
				}
			}
		}
	}

	fmt.Printf("\n")

	// Print system status
	fmt.Printf("⚙️  System Status\n")
	fmt.Printf("================\n")

	if data, ok := systemStatus["data"].(map[string]interface{}); ok {
		if state, ok := data["state"].(string); ok {
			fmt.Printf("State: %s\n", formatSystemState(state))
		}

		if startedAt, ok := data["started_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
				fmt.Printf("Started: %s\n", t.Format("2006-01-02 15:04:05"))
				fmt.Printf("Uptime: %s\n", time.Since(t).Truncate(time.Second))
			}
		}

		if version, ok := data["version"].(string); ok {
			fmt.Printf("Version: %s\n", version)
		}
	}

	fmt.Printf("\n")

	// Print metrics
	fmt.Printf("📊 Metrics\n")
	fmt.Printf("==========\n")

	if data, ok := metrics["data"].(map[string]interface{}); ok {
		printMetricsData(data)
	} else {
		fmt.Printf("Metrics not available\n")
	}

	return nil
}

func formatHealthStatus(status interface{}) string {
	if str, ok := status.(string); ok {
		switch str {
		case "healthy":
			return "✅ Healthy"
		case "unhealthy":
			return "❌ Unhealthy"
		default:
			return "❓ " + str
		}
	}
	return "❓ Unknown"
}

func formatSystemState(state string) string {
	switch state {
	case "running":
		return "🟢 Running"
	case "starting":
		return "🟡 Starting"
	case "stopping":
		return "🟡 Stopping"
	case "stopped":
		return "🔴 Stopped"
	case "error":
		return "🔴 Error"
	default:
		return "❓ " + state
	}
}

func printMetricsData(data map[string]interface{}) {
	// Print basic metrics
	for key, value := range data {
		switch key {
		case "requests_total":
			fmt.Printf("Total Requests: %s\n", formatNumber(value))
		case "errors_total":
			fmt.Printf("Total Errors: %s\n", formatNumber(value))
		case "repositories_monitored":
			fmt.Printf("Repositories: %s\n", formatNumber(value))
		case "events_processed":
			fmt.Printf("Events Processed: %s\n", formatNumber(value))
		case "last_poll_time":
			if str, ok := value.(string); ok {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					fmt.Printf("Last Poll: %s (%s ago)\n",
						t.Format("15:04:05"),
						time.Since(t).Truncate(time.Second))
				}
			}
		}
	}
}

func formatNumber(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
