package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johnnynv/RepoSentry/internal/config"
	"github.com/johnnynv/RepoSentry/internal/runtime"
	"github.com/johnnynv/RepoSentry/pkg/logger"
	"github.com/johnnynv/RepoSentry/pkg/types"
	"github.com/spf13/cobra"
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
	configFile string
	logLevel   string
	logFormat  string
	logFile    string
	healthPort int
	daemonMode bool
	pidFile    string
)

func init() {
	// Add run command to root
	rootCmd.AddCommand(runCmd)

	// Configuration flags
	runCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "l", "debug", "Log level (debug, info, warn, error)")
	runCmd.Flags().StringVar(&logFormat, "log-format", "json", "Log format (json, text)")
	runCmd.Flags().StringVar(&logFile, "log-file", "./logs/reposentry.log", "Log file path")
	runCmd.Flags().IntVar(&healthPort, "health-port", 8080, "Health check server port (0 to disable)")
	runCmd.Flags().BoolVarP(&daemonMode, "daemon", "d", false, "Run in daemon mode (background)")
	runCmd.Flags().StringVar(&pidFile, "pid-file", "", "PID file path (daemon mode only)")

	// Config has sensible defaults, no longer required
}

func runRepoSentry(cmd *cobra.Command, args []string) error {
	// Initialize enterprise logger system
	loggerConfig := logger.DefaultConfig()
	if logLevel != "" {
		loggerConfig.Level = logLevel
	}
	if logFile != "" {
		loggerConfig.Output = logFile
	}

	loggerManager, err := logger.NewManager(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger manager: %w", err)
	}

	// Create business logger for future use
	_ = logger.NewBusinessLogger(loggerManager)

	// Create startup context
	ctx := context.Background()
	startupCtx := logger.WithContext(ctx, logger.LogContext{
		Component: "app",
		Module:    "startup",
		Operation: "run",
	})

	startupLogger := loggerManager.WithGoContext(startupCtx)
	startupLogger.WithFields(logger.Fields{
		"config":    configFile,
		"log_level": logLevel,
		"daemon":    daemonMode,
	}).Info("Starting RepoSentry with enterprise logging")

	// Handle daemon mode
	if daemonMode {
		if err := handleDaemonMode(startupLogger); err != nil {
			return fmt.Errorf("failed to start in daemon mode: %w", err)
		}
	}

	// Load configuration
	configManager := config.NewManager(loggerManager.GetRootLogger())
	if err := configManager.Load(configFile); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Reconfigure logger based on configuration file if needed
	if loggerConfig := configManager.GetLoggerConfig(); loggerConfig.Output != "" {
		startupLogger.Info("Logger already configured with enterprise system")
	}

	// Get configuration
	cfg := configManager.Get()
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}

	// Override configuration with CLI flags
	overrideConfigFromFlags(cfg, startupLogger)

	// Create runtime factory and runtime with logger manager
	factory := runtime.NewDefaultRuntimeFactory()
	rt, err := factory.CreateRuntime(cfg, loggerManager)
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

		startupLogger.WithFields(logger.Fields{
			"operation":   "running",
			"health_port": cfg.App.HealthCheckPort,
		}).Info("RepoSentry is running successfully")

		// Keep running until context is cancelled
		<-ctx.Done()

		startupLogger.WithFields(logger.Fields{
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
			startupLogger.WithFields(logger.Fields{
				"signal": sig.String(),
			}).Info("Received signal")

			switch sig {
			case syscall.SIGHUP:
				// Reload configuration
				startupLogger.Info("Reloading configuration")
				if err := configManager.Reload(); err != nil {
					startupLogger.WithFields(logger.Fields{
						"error": err.Error(),
					}).Error("Failed to reload configuration")
				} else {
					startupLogger.Info("Configuration reloaded successfully")
					// TODO: Implement selective component restart based on config changes
					if err := rt.Reload(ctx); err != nil {
						startupLogger.WithFields(logger.Fields{
							"error": err.Error(),
						}).Error("Failed to reload runtime")
					}
				}
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				startupLogger.WithFields(logger.Fields{
					"signal": sig.String(),
				}).Info("Initiating graceful shutdown")
				cancel()

				// Wait for runtime to stop
				if err := <-runtimeErrChan; err != nil {
					startupLogger.WithFields(logger.Fields{
						"error": err.Error(),
					}).Error("Error during shutdown")
					return err
				}

				startupLogger.Info("RepoSentry stopped successfully")
				return nil
			}

		case err := <-runtimeErrChan:
			if err != nil {
				startupLogger.WithFields(logger.Fields{
					"error": err.Error(),
				}).Error("Runtime error")
				return err
			}
			// Normal shutdown
			startupLogger.Info("RepoSentry stopped successfully")
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

func reconfigureLogger(configManager *config.Manager, currentLogger *logger.Logger) error {
	// Get logger configuration from config manager
	logConfig := configManager.GetLoggerConfig()

	// Log the reconfiguration attempt
	currentLogger.WithFields(logger.Fields{
		"level":  logConfig.Level,
		"format": logConfig.Format,
		"output": logConfig.Output,
	}).Info("Attempting to reconfigure logger from configuration file")

	// Note: Logger reconfiguration requires more sophisticated implementation
	// For now, we'll use the CLI flags which are already working
	// The configuration file logger settings will be used in future versions

	return nil
}

func overrideConfigFromFlags(cfg *types.Config, startupLogger *logger.Entry) {
	// Override health check port if specified
	if healthPort > 0 {
		cfg.App.HealthCheckPort = healthPort
		startupLogger.WithFields(logger.Fields{
			"port": healthPort,
		}).Debug("Overriding health check port from CLI flag")
	}

	// TODO: Add more flag overrides as needed
}

func handleDaemonMode(startupLogger *logger.Entry) error {
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
			startupLogger.WithFields(logger.Fields{
				"error":    err.Error(),
				"pid_file": pidFile,
			}).Error("Failed to remove PID file")
		}
	}()

	startupLogger.WithFields(logger.Fields{
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
