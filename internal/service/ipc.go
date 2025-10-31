package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/crypto"
	"github.com/gajzzs/keyphy/internal/device"
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
	
	// Use service manager to check status
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %v", err)
	}
	
	// Check if service is running
	status, err := sm.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	
	if status != "Running" {
		return fmt.Errorf("daemon service is not running (status: %s)", status)
	}
	
	// Find keyphy daemon process and send signal
	return sendSignalToKeyphyDaemon(SIGUSR1)
}

func SendLockSignal() error {
	// Validate device before sending signal
	if !validateDeviceBeforeSignal() {
		return fmt.Errorf("authentication device not connected or invalid")
	}
	
	// Use service manager to check status
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %v", err)
	}
	
	// Check if service is running
	status, err := sm.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	
	if status != "Running" {
		return fmt.Errorf("daemon service is not running (status: %s)", status)
	}
	
	// Find keyphy daemon process and send signal
	return sendSignalToKeyphyDaemon(SIGUSR2)
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

func IsProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// sendSignalToKeyphyDaemon finds the running keyphy daemon and sends a signal
func sendSignalToKeyphyDaemon(sig syscall.Signal) error {
	// Use ps to find keyphy daemon process
	cmd := exec.Command("pgrep", "-f", "keyphy service run-daemon")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("keyphy daemon not found: %v", err)
	}
	
	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return fmt.Errorf("keyphy daemon not running")
	}
	
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid daemon PID: %v", err)
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find daemon process: %v", err)
	}
	
	return process.Signal(sig)
}

// Public functions for external access
func ReadPidFile() (int, error) {
	// Legacy function kept for compatibility
	return 0, fmt.Errorf("PID file access deprecated - use service package")
}