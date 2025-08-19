package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/runtime"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the RepoSentry monitoring service",
	Long: `Start the RepoSentry monitoring service with the specified configuration.
This command starts all components (storage, git clients, poller, triggers) and
monitors repositories for changes according to the configuration.`,
	RunE: runRepoSentry,
}

var (
	configFile     string
	logLevel       string
	logFormat      string
	logFile        string
	healthPort     int
	daemonMode     bool
	pidFile        string
)

func init() {
	// Add run command to root
	rootCmd.AddCommand(runCmd)
	
	// Configuration flags
	runCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path (required)")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	runCmd.Flags().StringVar(&logFormat, "log-format", "json", "Log format (json, text)")
	runCmd.Flags().StringVar(&logFile, "log-file", "", "Log file path (optional, logs to stdout if not specified)")
	runCmd.Flags().IntVar(&healthPort, "health-port", 0, "Health check server port (0 to disable)")
	runCmd.Flags().BoolVarP(&daemonMode, "daemon", "d", false, "Run in daemon mode (background)")
	runCmd.Flags().StringVar(&pidFile, "pid-file", "", "PID file path (daemon mode only)")
	
	// Mark config as required
	runCmd.MarkFlagRequired("config")
}

func runRepoSentry(cmd *cobra.Command, args []string) error {
	// Initialize logger first
	appLogger, err := initializeLogger()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	appLogger.WithFields(logger.Fields{
		"operation": "startup",
		"config":    configFile,
		"log_level": logLevel,
		"daemon":    daemonMode,
	}).Info("Starting RepoSentry")

	// Handle daemon mode
	if daemonMode {
		if err := handleDaemonMode(appLogger); err != nil {
			return fmt.Errorf("failed to start in daemon mode: %w", err)
		}
	}

	// Load configuration
	configManager := config.NewManager(appLogger)
	if err := configManager.Load(configFile); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get configuration
	cfg := configManager.Get()
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}

	// Override configuration with CLI flags
	overrideConfigFromFlags(cfg, appLogger)

	// Create runtime factory and runtime
	factory := runtime.NewDefaultRuntimeFactory()
	rt, err := factory.CreateRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start the runtime in a goroutine
	runtimeErrChan := make(chan error, 1)
	go func() {
		if err := rt.Start(ctx); err != nil {
			runtimeErrChan <- fmt.Errorf("runtime start failed: %w", err)
			return
		}

		appLogger.WithFields(logger.Fields{
			"operation": "running",
			"health_port": cfg.App.HealthCheckPort,
		}).Info("RepoSentry is running successfully")

		// Keep running until context is cancelled
		<-ctx.Done()
		
		appLogger.WithFields(logger.Fields{
			"operation": "shutdown",
		}).Info("Initiating graceful shutdown")

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := rt.Stop(shutdownCtx); err != nil {
			runtimeErrChan <- fmt.Errorf("runtime stop failed: %w", err)
		} else {
			runtimeErrChan <- nil
		}
	}()

	// Wait for signals or runtime errors
	for {
		select {
		case sig := <-sigChan:
			appLogger.WithFields(logger.Fields{
				"signal": sig.String(),
			}).Info("Received signal")

			switch sig {
			case syscall.SIGHUP:
				// Reload configuration
				appLogger.Info("Reloading configuration")
				if err := configManager.Reload(); err != nil {
					appLogger.WithFields(logger.Fields{
						"error": err.Error(),
					}).Error("Failed to reload configuration")
				} else {
					appLogger.Info("Configuration reloaded successfully")
					// TODO: Implement selective component restart based on config changes
					if err := rt.Reload(ctx); err != nil {
						appLogger.WithFields(logger.Fields{
							"error": err.Error(),
						}).Error("Failed to reload runtime")
					}
				}
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				appLogger.WithFields(logger.Fields{
					"signal": sig.String(),
				}).Info("Initiating graceful shutdown")
				cancel()
				
				// Wait for runtime to stop
				if err := <-runtimeErrChan; err != nil {
					appLogger.WithFields(logger.Fields{
						"error": err.Error(),
					}).Error("Error during shutdown")
					return err
				}
				
				appLogger.Info("RepoSentry stopped successfully")
				return nil
			}

		case err := <-runtimeErrChan:
			if err != nil {
				appLogger.WithFields(logger.Fields{
					"error": err.Error(),
				}).Error("Runtime error")
				return err
			}
			// Normal shutdown
			appLogger.Info("RepoSentry stopped successfully")
			return nil
		}
	}
}

func initializeLogger() (*logger.Logger, error) {
	// Create logger config
	logConfig := logger.Config{
		Level:  logLevel,
		Format: logFormat,
		Output: "stdout",
	}

	// Set up file output if specified
	if logFile != "" {
		logConfig.Output = logFile // Use the file path directly
		logConfig.File = logger.FileConfig{
			MaxSize:    100, // 100MB
			MaxAge:     30,  // 30 days
			MaxBackups: 5,   // Keep 5 backup files
			Compress:   true,
		}
	}

	// Create and configure logger
	return logger.NewLogger(logConfig)
}

func overrideConfigFromFlags(cfg *types.Config, appLogger *logger.Logger) {
	// Override health check port if specified
	if healthPort > 0 {
		cfg.App.HealthCheckPort = healthPort
		appLogger.WithFields(logger.Fields{
			"port": healthPort,
		}).Debug("Overriding health check port from CLI flag")
	}

	// TODO: Add more flag overrides as needed
}

func handleDaemonMode(appLogger *logger.Logger) error {
	if pidFile == "" {
		return fmt.Errorf("daemon mode requires --pid-file flag")
	}

	// Check if already running
	if isProcessRunning(pidFile) {
		return fmt.Errorf("RepoSentry is already running (PID file exists: %s)", pidFile)
	}

	// Write PID file
	if err := writePIDFile(pidFile); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Ensure PID file is cleaned up on exit
	defer func() {
		if err := os.Remove(pidFile); err != nil {
			appLogger.WithFields(logger.Fields{
				"error": err.Error(),
				"pid_file": pidFile,
			}).Error("Failed to remove PID file")
		}
	}()

	appLogger.WithFields(logger.Fields{
		"pid_file": pidFile,
		"pid":      os.Getpid(),
	}).Info("Started in daemon mode")

	return nil
}

func isProcessRunning(pidFile string) bool {
	// Check if PID file exists and if the process is still running
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return false
	}

	// TODO: Implement proper PID file checking
	// Read PID from file and check if process exists
	return false
}

func writePIDFile(pidFile string) error {
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}
