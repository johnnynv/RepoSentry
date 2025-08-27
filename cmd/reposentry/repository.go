package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository management commands",
	Long:  "List, inspect, and manage monitored repositories",
}

var listReposCmd = &cobra.Command{
	Use:   "list",
	Short: "List monitored repositories",
	Long:  "Show all repositories currently being monitored by RepoSentry",
	RunE:  runListRepos,
}

var showRepoCmd = &cobra.Command{
	Use:   "show <repository-name>",
	Short: "Show repository details",
	Long:  "Display detailed information about a specific repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowRepo,
}

var (
	repoPort   int
	repoHost   string
	repoFormat string
)

func init() {
	listReposCmd.Flags().IntVar(&repoPort, "port", 8080, "RepoSentry API port")
	listReposCmd.Flags().StringVar(&repoHost, "host", "localhost", "RepoSentry host")
	listReposCmd.Flags().StringVar(&repoFormat, "format", "table", "Output format (table, json)")

	showRepoCmd.Flags().IntVar(&repoPort, "port", 8080, "RepoSentry API port")
	showRepoCmd.Flags().StringVar(&repoHost, "host", "localhost", "RepoSentry host")
	showRepoCmd.Flags().StringVar(&repoFormat, "format", "text", "Output format (text, json)")

	repoCmd.AddCommand(listReposCmd)
	repoCmd.AddCommand(showRepoCmd)

	rootCmd.AddCommand(repoCmd)
}

func runListRepos(cmd *cobra.Command, args []string) error {
	baseURL := fmt.Sprintf("http://%s:%d", repoHost, repoPort)

	repos, err := getRepositories(baseURL)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	if repoFormat == "json" {
		return printRepositoriesJSON(repos)
	}

	return printRepositoriesTable(repos)
}

func runShowRepo(cmd *cobra.Command, args []string) error {
	baseURL := fmt.Sprintf("http://%s:%d", repoHost, repoPort)
	repoName := args[0]

	repo, err := getRepository(baseURL, repoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	if repoFormat == "json" {
		return printRepositoryJSON(repo)
	}

	return printRepositoryText(repo)
}

func getRepositories(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/api/repositories")
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

func getRepository(baseURL, name string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/repositories/%s", baseURL, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("repository not found: %s", name)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func printRepositoriesJSON(repos map[string]interface{}) error {
	jsonBytes, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func printRepositoriesTable(repos map[string]interface{}) error {
	data, ok := repos["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	repositories, ok := data["repositories"].([]interface{})
	if !ok {
		fmt.Printf("No repositories configured\n")
		return nil
	}

	if len(repositories) == 0 {
		fmt.Printf("No repositories configured\n")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROVIDER\tURL\tBRANCH REGEX\tPOLL INTERVAL\tSTATUS")
	fmt.Fprintln(w, "----\t--------\t---\t------------\t-------------\t------")

	for _, repo := range repositories {
		if repoMap, ok := repo.(map[string]interface{}); ok {
			name := getStringValue(repoMap, "name")
			provider := getStringValue(repoMap, "provider")
			url := getStringValue(repoMap, "url")
			branchRegex := getStringValue(repoMap, "branch_regex")
			pollInterval := getStringValue(repoMap, "polling_interval")
			status := "Active" // TODO: Get actual status

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				name, provider, url, branchRegex, pollInterval, status)
		}
	}

	w.Flush()

	// Print summary
	total, _ := data["total"].(float64)
	fmt.Printf("\nTotal: %.0f repositories\n", total)

	return nil
}

func printRepositoryJSON(repo map[string]interface{}) error {
	jsonBytes, err := json.MarshalIndent(repo, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func printRepositoryText(repo map[string]interface{}) error {
	data, ok := repo["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	fmt.Printf("📁 Repository Details\n")
	fmt.Printf("====================\n\n")

	fmt.Printf("Name: %s\n", getStringValue(data, "name"))
	fmt.Printf("Provider: %s\n", getStringValue(data, "provider"))
	fmt.Printf("URL: %s\n", getStringValue(data, "url"))
	fmt.Printf("Branch Regex: %s\n", getStringValue(data, "branch_regex"))
	fmt.Printf("Poll Interval: %s\n", getStringValue(data, "polling_interval"))

	// Additional details if available
	if token := getStringValue(data, "token"); token != "" {
		fmt.Printf("Token: ***configured***\n")
	} else {
		fmt.Printf("Token: not configured\n")
	}

	// TODO: Add status information, last poll time, etc.

	return nil
}

func getStringValue(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
