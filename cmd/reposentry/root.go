package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfigFile string
	globalLogLevel   string
	globalLogFormat  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "reposentry",
	Short: "RepoSentry - Git Repository Monitor",
	Long: `RepoSentry is a lightweight, cloud-native sentinel that keeps an independent watch 
over your GitLab and GitHub repositories.

It monitors repository branches for changes and triggers Tekton pipelines via webhooks.`,
	// Remove the global pre-run, let each command handle its own initialization
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&globalConfigFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&globalLogLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&globalLogFormat, "log-format", "json", "log format (json, text)")

	// Bind flags to viper
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log_format", rootCmd.PersistentFlags().Lookup("log-format"))
}

// initConfig reads in ENV variables if set.
func initConfig() {
	// Read in environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("RS") // RS_LOG_LEVEL, RS_CONFIG_PATH, etc.
}
