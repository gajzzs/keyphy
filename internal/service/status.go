package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

func GetDaemonStatus() (bool, error) {
	// First check PID file method
	if pid, err := ReadPidFile(); err == nil {
		if IsProcessRunning(pid) {
			return true, nil
		}
		// Clean up stale PID file
		os.Remove(GetUniquePidFile())
	}
	
	// Fallback: check for running processes using gopsutil
	processes, err := process.Processes()
	if err != nil {
		return false, nil
	}
	
	runningCount := 0
	for _, proc := range processes {
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}
		
		if strings.Contains(cmdline, "keyphy service run-daemon") {
			runningCount++
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
	
	if err := os.WriteFile(pidFile, []byte(data), 0600); err != nil {
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
	// Kill all keyphy daemon processes using gopsutil
	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to get processes: %v", err)
	}
	
	killedCount := 0
	for _, proc := range processes {
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}
		
		if strings.Contains(cmdline, "keyphy service run-daemon") {
			if err := proc.Kill(); err == nil {
				killedCount++
			}
		}
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
	return ReadPidFile()
}

func IsProcessRunningExternal(pid int) bool {
	return IsProcessRunning(pid)
}

