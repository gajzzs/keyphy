package service

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

const (
	SIGUSR1 = syscall.SIGUSR1 // Unlock signal
	SIGUSR2 = syscall.SIGUSR2 // Lock signal
)

func SendUnlockSignal() error {
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