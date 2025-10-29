package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"keyphy/internal/config"
	"keyphy/internal/crypto"
	"keyphy/internal/device"
)

const (
	SIGUSR1 = syscall.SIGUSR1 // Unlock signal
	SIGUSR2 = syscall.SIGUSR2 // Lock signal
)

func SendUnlockSignal() error {
	// Validate device before sending signal
	if !validateDeviceBeforeSignal() {
		return fmt.Errorf("authentication device not connected or invalid")
	}
	
	pid, err := readPidFile()
	if err != nil {
		return fmt.Errorf("daemon not running: %v", err)
	}
	
	// Check if process is actually running
	if !isProcessRunning(pid) {
		// Clean up stale PID file
		os.Remove("/var/run/keyphy.pid")
		return fmt.Errorf("daemon not running (stale PID file removed)")
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find daemon process: %v", err)
	}
	
	return process.Signal(SIGUSR1)
}

func SendLockSignal() error {
	// Validate device before sending signal
	if !validateDeviceBeforeSignal() {
		return fmt.Errorf("authentication device not connected or invalid")
	}
	
	pid, err := readPidFile()
	if err != nil {
		return fmt.Errorf("daemon not running: %v", err)
	}
	
	// Check if process is actually running
	if !isProcessRunning(pid) {
		// Clean up stale PID file
		os.Remove("/var/run/keyphy.pid")
		return fmt.Errorf("daemon not running (stale PID file removed)")
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find daemon process: %v", err)
	}
	
	return process.Signal(SIGUSR2)
}

func validateDeviceBeforeSignal() bool {
	cfg := config.GetConfig()
	if cfg.AuthDevice == "" || cfg.AuthKey == "" {
		return false
	}
	
	devices, err := device.ListUSBDevices()
	if err != nil {
		return false
	}
	
	for _, dev := range devices {
		if dev.UUID == cfg.AuthDevice {
			valid, err := crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
			if err != nil {
				return false
			}
			return valid
		}
	}
	return false
}

func isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func SendStopSignal() error {
	pid, err := readPidFile()
	if err != nil {
		return fmt.Errorf("daemon not running: %v", err)
	}
	
	// Check if process is actually running
	if !isProcessRunning(pid) {
		// Clean up stale PID file
		os.Remove("/var/run/keyphy.pid")
		return fmt.Errorf("daemon not running (stale PID file removed)")
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find daemon process: %v", err)
	}
	
	// Send SIGTERM to gracefully stop daemon
	return process.Signal(syscall.SIGTERM)
}

func readPidFile() (int, error) {
	pidFile := "/var/run/keyphy.pid"
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	
	return pid, nil
}