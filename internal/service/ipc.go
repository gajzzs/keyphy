package service

import (
	"fmt"
	"os"
	"strconv"
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
			return crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
		}
	}
	return false
}

func readPidFile() (int, error) {
	pidFile := "/var/run/keyphy.pid"
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}
	
	return pid, nil
}