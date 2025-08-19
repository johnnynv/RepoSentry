package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage RepoSentry daemon",
	Long:  `Manage RepoSentry daemon process (start, stop, restart, status)`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start RepoSentry as a daemon",
	Long:  `Start RepoSentry as a background daemon process`,
	RunE:  daemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop RepoSentry daemon",
	Long:  `Stop the running RepoSentry daemon process`,
	RunE:  daemonStop,
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart RepoSentry daemon",
	Long:  `Restart the RepoSentry daemon process`,
	RunE:  daemonRestart,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show RepoSentry daemon status",
	Long:  `Show the status of the RepoSentry daemon process`,
	RunE:  daemonStatus,
}

var (
	daemonConfigFile string
	daemonLogLevel   string
	daemonLogFormat  string
	daemonLogFile    string
	daemonPIDFile    string
	daemonHealthPort int
)

func init() {
	// Add daemon command to root
	rootCmd.AddCommand(daemonCmd)
	
	// Add subcommands
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	
	// Flags for daemon start
	daemonStartCmd.Flags().StringVarP(&daemonConfigFile, "config", "c", "", "Configuration file path (required)")
	daemonStartCmd.Flags().StringVarP(&daemonLogLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	daemonStartCmd.Flags().StringVar(&daemonLogFormat, "log-format", "json", "Log format (json, text)")
	daemonStartCmd.Flags().StringVar(&daemonLogFile, "log-file", "/var/log/reposentry.log", "Log file path")
	daemonStartCmd.Flags().StringVar(&daemonPIDFile, "pid-file", "/var/run/reposentry.pid", "PID file path")
	daemonStartCmd.Flags().IntVar(&daemonHealthPort, "health-port", 8080, "Health check server port")
	
	// Mark config as required for start
	daemonStartCmd.MarkFlagRequired("config")
	
	// Flags for stop, restart, status commands
	for _, cmd := range []*cobra.Command{daemonStopCmd, daemonRestartCmd, daemonStatusCmd} {
		cmd.Flags().StringVar(&daemonPIDFile, "pid-file", "/var/run/reposentry.pid", "PID file path")
	}
}

func daemonStart(cmd *cobra.Command, args []string) error {
	// Check if already running
	if pid, running := isDaemonRunning(daemonPIDFile); running {
		return fmt.Errorf("RepoSentry daemon is already running (PID: %d)", pid)
	}

	// Build command arguments
	cmdArgs := []string{
		os.Args[0], // reposentry executable
		"run",
		"--config", daemonConfigFile,
		"--log-level", daemonLogLevel,
		"--log-format", daemonLogFormat,
		"--log-file", daemonLogFile,
		"--daemon",
		"--pid-file", daemonPIDFile,
		"--health-port", strconv.Itoa(daemonHealthPort),
	}

	// Start the daemon process
	process, err := startDaemonProcess(cmdArgs)
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("RepoSentry daemon started successfully (PID: %d)\n", process.Pid)
	fmt.Printf("Configuration: %s\n", daemonConfigFile)
	fmt.Printf("Log file: %s\n", daemonLogFile)
	fmt.Printf("PID file: %s\n", daemonPIDFile)
	fmt.Printf("Health check: http://localhost:%d/health\n", daemonHealthPort)

	return nil
}

func daemonStop(cmd *cobra.Command, args []string) error {
	pid, running := isDaemonRunning(daemonPIDFile)
	if !running {
		return fmt.Errorf("RepoSentry daemon is not running")
	}

	// Send SIGTERM signal
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
	}

	fmt.Printf("Sent shutdown signal to RepoSentry daemon (PID: %d)\n", pid)
	
	// TODO: Wait for process to actually stop and remove PID file
	return nil
}

func daemonRestart(cmd *cobra.Command, args []string) error {
	// Stop if running
	if pid, running := isDaemonRunning(daemonPIDFile); running {
		fmt.Printf("Stopping RepoSentry daemon (PID: %d)...\n", pid)
		if err := daemonStop(cmd, args); err != nil {
			return err
		}
		
		// TODO: Wait for process to actually stop
		fmt.Println("Waiting for daemon to stop...")
		// time.Sleep(2 * time.Second)
	}

	// Start again
	fmt.Println("Starting RepoSentry daemon...")
	return daemonStart(cmd, args)
}

func daemonStatus(cmd *cobra.Command, args []string) error {
	pid, running := isDaemonRunning(daemonPIDFile)
	
	if running {
		fmt.Printf("RepoSentry daemon is running (PID: %d)\n", pid)
		
		// Try to get status from health endpoint if available
		// TODO: Implement health check query
		
		return nil
	} else {
		fmt.Println("RepoSentry daemon is not running")
		return nil
	}
}

func isDaemonRunning(pidFile string) (int, bool) {
	// Check if PID file exists
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false
	}

	// Parse PID
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, false
	}

	// Check if process is actually running
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}

	// Send signal 0 to check if process exists
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Process doesn't exist, remove stale PID file
		os.Remove(pidFile)
		return 0, false
	}

	return pid, true
}

func startDaemonProcess(cmdArgs []string) (*os.Process, error) {
	// Create process attributes for daemon
	procAttr := &os.ProcAttr{
		Dir: "/", // Run from root directory
		Env: os.Environ(),
		Files: []*os.File{
			nil, // stdin
			nil, // stdout (will be redirected to log file)
			nil, // stderr (will be redirected to log file)
		},
	}

	// Start the process
	process, err := os.StartProcess(cmdArgs[0], cmdArgs, procAttr)
	if err != nil {
		return nil, err
	}

	// Detach from parent
	if err := process.Release(); err != nil {
		return nil, fmt.Errorf("failed to detach process: %w", err)
	}

	return process, nil
}
