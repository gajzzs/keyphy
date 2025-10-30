package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func GetDaemonStatus() (bool, error) {
	// First check PID file method
	if pid, err := readPidFile(); err == nil {
		if isProcessRunning(pid) {
			return true, nil
		}
		// Clean up stale PID file
		os.Remove("/var/run/keyphy.pid")
	}
	
	// Fallback: check for running processes
	cmd := exec.Command("pgrep", "-f", "keyphy service run-daemon")
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	
	// Verify processes are actually running
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	runningCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
				if isProcessRunning(pid) {
					runningCount++
				}
			}
		}
	}
	
	return runningCount > 0, nil
}

func GetServiceStatus() string {
	// Check systemd service status
	cmd := exec.Command("systemctl", "is-active", "keyphy")
	output, err := cmd.Output()
	if err != nil {
		return "inactive"
	}
	return string(output)
}

func CreatePidFile() error {
	pidFile := GetUniquePidFile()
	pid := os.Getpid()
	data := fmt.Sprintf("%d\n", pid)
	
	// Ensure /var/run directory exists and is writable
	if err := os.MkdirAll("/var/run", 0755); err != nil {
		return fmt.Errorf("failed to create /var/run directory: %v", err)
	}
	
	if err := os.WriteFile(pidFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %v", err)
	}
	
	return nil
}



func RemovePidFile() error {
	pidFile := GetUniquePidFile()
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to remove
	}
	return os.Remove(pidFile)
}

func StopAllDaemons() error {
	// Kill all keyphy daemon processes
	cmd := exec.Command("pkill", "-f", "keyphy service run-daemon")
	if err := cmd.Run(); err != nil {
		// If no processes found, that's actually success
		if err.Error() == "exit status 1" {
			return nil // No processes to kill
		}
		return fmt.Errorf("failed to kill daemon processes: %v", err)
	}
	
	// Clean up PID file
	RemovePidFile()
	
	return nil
}

func StartDaemonBackground() error {
	fmt.Println("Starting keyphy daemon...")
	
	// Fork process to run in background
	cmd := exec.Command(os.Args[0], "service", "run-daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %v", err)
	}
	
	fmt.Println("Keyphy daemon started successfully")
	return nil
}

// External access functions for other packages
func ReadPidFileExternal() (int, error) {
	return readPidFile()
}

func IsProcessRunningExternal(pid int) bool {
	return isProcessRunning(pid)
}

