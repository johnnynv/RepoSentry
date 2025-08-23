package main

import (
	"fmt"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Log management commands",
	Long:  "Commands for managing RepoSentry logs including rotation and statistics",
}

var logStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show log statistics",
	Long:  "Display current log file statistics including size, rotation info, etc.",
	RunE:  runLogStats,
}

var logRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Manually rotate log file",
	Long:  "Manually trigger log rotation",
	RunE:  runLogRotate,
}

func init() {
	logCmd.AddCommand(logStatsCmd)
	logCmd.AddCommand(logRotateCmd)

	// Add to root command
	rootCmd.AddCommand(logCmd)

	// Add flags for log commands
	logStatsCmd.Flags().StringVar(&configFile, "config", "config.yaml", "config file")
	logRotateCmd.Flags().StringVar(&configFile, "config", "config.yaml", "config file")
}

func runLogStats(cmd *cobra.Command, args []string) error {
	// Create temporary logger for config loading
	tempLoggerConfig := logger.DefaultConfig()
	tempLoggerManager, err := logger.NewManager(tempLoggerConfig)
	if err != nil {
		return fmt.Errorf("failed to create temp logger manager: %w", err)
	}
	defer tempLoggerManager.Close()

	// Load configuration
	configManager := config.NewManager(tempLoggerManager.GetRootLogger())
	if err := configManager.Load(configFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := configManager.Get()
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}

	// Create logger manager with actual config
	loggerManager, err := logger.NewManager(logger.Config{
		Level:  cfg.App.LogLevel,
		Format: "json",
		Output: cfg.App.LogFile,
		File: logger.FileConfig{
			MaxSize:    cfg.App.LogFileRotation.MaxSize,
			MaxAge:     cfg.App.LogFileRotation.MaxAge,
			MaxBackups: cfg.App.LogFileRotation.MaxBackups,
			Compress:   cfg.App.LogFileRotation.Compress,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create logger manager: %w", err)
	}
	defer loggerManager.Close()

	// Get log statistics
	stats, err := loggerManager.GetLogStats()
	if err != nil {
		return fmt.Errorf("failed to get log stats: %w", err)
	}

	if stats == nil {
		fmt.Println("Log rotation not enabled (output is not a file)")
		return nil
	}

	// Display statistics
	fmt.Println("=== 📊 RepoSentry Log Statistics ===")
	fmt.Printf("📁 Current File: %s\n", stats.CurrentFile)
	fmt.Printf("📦 Current Size: %s\n", stats.FormatSize(stats.CurrentSize))
	fmt.Printf("🕐 Last Modified: %s\n", stats.LastModified.Format("2006-01-02 15:04:05"))
	fmt.Printf("⚙️  Max Size: %d MB\n", stats.MaxSize)
	fmt.Printf("📅 Max Age: %d days\n", stats.MaxAge)
	fmt.Printf("📚 Max Backups: %d\n", stats.MaxBackups)
	fmt.Printf("🗜️  Compression: %t\n", stats.Compress)

	// Check if rotation is needed
	currentSizeMB := float64(stats.CurrentSize) / (1024 * 1024)
	if currentSizeMB > float64(stats.MaxSize)*0.8 {
		fmt.Printf("⚠️  Warning: Log file is %.1f%% of max size\n",
			(currentSizeMB/float64(stats.MaxSize))*100)
	}

	return nil
}

func runLogRotate(cmd *cobra.Command, args []string) error {
	// Create temporary logger for config loading
	tempLoggerConfig := logger.DefaultConfig()
	tempLoggerManager, err := logger.NewManager(tempLoggerConfig)
	if err != nil {
		return fmt.Errorf("failed to create temp logger manager: %w", err)
	}
	defer tempLoggerManager.Close()

	// Load configuration
	configManager := config.NewManager(tempLoggerManager.GetRootLogger())
	if err := configManager.Load(configFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := configManager.Get()
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}

	// Create logger manager with actual config
	loggerManager, err := logger.NewManager(logger.Config{
		Level:  cfg.App.LogLevel,
		Format: "json",
		Output: cfg.App.LogFile,
		File: logger.FileConfig{
			MaxSize:    cfg.App.LogFileRotation.MaxSize,
			MaxAge:     cfg.App.LogFileRotation.MaxAge,
			MaxBackups: cfg.App.LogFileRotation.MaxBackups,
			Compress:   cfg.App.LogFileRotation.Compress,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create logger manager: %w", err)
	}
	defer loggerManager.Close()

	// Perform rotation
	fmt.Println("🔄 Rotating log file...")
	if err := loggerManager.RotateLog(); err != nil {
		return fmt.Errorf("failed to rotate log: %w", err)
	}

	fmt.Println("✅ Log rotation completed successfully")
	return nil
}
