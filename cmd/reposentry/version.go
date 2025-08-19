package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// BuildInfo represents build information
type BuildInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print version information including build time and git commit",
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	
	// Add format flag
	versionCmd.Flags().StringP("output", "o", "text", "output format (text, json)")
}

func runVersion(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	
	buildInfo := BuildInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}

	switch output {
	case "json":
		jsonData, err := json.MarshalIndent(buildInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(jsonData))
	default:
		fmt.Printf("RepoSentry %s\n", buildInfo.Version)
		fmt.Printf("Build Time: %s\n", buildInfo.BuildTime)
		fmt.Printf("Git Commit: %s\n", buildInfo.GitCommit)
	}

	return nil
}
