package service

import (
	"fmt"
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
	
	// PID file cleanup handled by service package
	
	return nil
}





