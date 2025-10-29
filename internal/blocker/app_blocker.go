package blocker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

)

type AppBlocker struct {
	blockedApps map[string]bool
}

func NewAppBlocker() *AppBlocker {
	return &AppBlocker{
		blockedApps: make(map[string]bool),
	}
}

func (ab *AppBlocker) BlockApp(appName string) error {
	ab.blockedApps[appName] = true
	
	// Kill existing processes
	if err := ab.killProcesses(appName); err != nil {
		return fmt.Errorf("failed to kill existing processes: %v", err)
	}
	
	// Set up D-Bus monitoring for new launches
	return ab.setupDBusMonitoring(appName)
}

func (ab *AppBlocker) UnblockApp(appName string) error {
	delete(ab.blockedApps, appName)
	return ab.restoreOriginalExecutable(appName)
}

func (ab *AppBlocker) restoreOriginalExecutable(appName string) error {
	// Find the executable path
	execPath, err := exec.LookPath(appName)
	if err != nil {
		commonPaths := []string{
			"/usr/bin/" + appName,
			"/usr/local/bin/" + appName,
			"/snap/bin/" + appName,
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path + ".keyphy-backup"); err == nil {
				execPath = path
				break
			}
		}
		if execPath == "" {
			return nil // No backup found, nothing to restore
		}
	}
	
	// Restore from backup
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); err == nil {
		if err := exec.Command("cp", backupPath, execPath).Run(); err != nil {
			return fmt.Errorf("failed to restore executable: %v", err)
		}
		// Remove backup
		os.Remove(backupPath)
	}
	
	return nil
}

func (ab *AppBlocker) IsBlocked(appName string) bool {
	return ab.blockedApps[appName]
}

func (ab *AppBlocker) killProcesses(appName string) error {
	cmd := exec.Command("pkill", "-f", appName)
	return cmd.Run()
}

func (ab *AppBlocker) setupDBusMonitoring(appName string) error {
	// Create executable wrapper that blocks the app
	return ab.createBlockingWrapper(appName)
}

func (ab *AppBlocker) createBlockingWrapper(appName string) error {
	// Find the actual executable path
	execPath, err := exec.LookPath(appName)
	if err != nil {
		// App not found in PATH, try common locations
		commonPaths := []string{
			"/usr/bin/" + appName,
			"/usr/local/bin/" + appName,
			"/snap/bin/" + appName,
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				execPath = path
				break
			}
		}
		if execPath == "" {
			return fmt.Errorf("executable %s not found", appName)
		}
	}
	
	// Backup original executable
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		if err := exec.Command("cp", execPath, backupPath).Run(); err != nil {
			return fmt.Errorf("failed to backup executable: %v", err)
		}
	}
	
	// Create blocking script
	blockScript := fmt.Sprintf(`#!/bin/bash
echo "Access to %s is blocked by Keyphy"
exit 1
`, appName)
	
	// Replace executable with blocking script
	if err := os.WriteFile(execPath, []byte(blockScript), 0755); err != nil {
		return fmt.Errorf("failed to create blocking wrapper: %v", err)
	}
	
	return nil
}

func (ab *AppBlocker) BlockProcessLaunch(pid int) error {
	// Send SIGTERM to block process launch
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}
	
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate process %d: %v", pid, err)
	}
	
	return nil
}

func (ab *AppBlocker) GetRunningProcesses(appName string) ([]int, error) {
	var pids []int
	
	// Sanitize appName to prevent command injection
	if strings.ContainsAny(appName, ";|&$`(){}[]<>*?~") {
		return nil, fmt.Errorf("invalid characters in app name: %s", appName)
	}
	
	cmd := exec.Command("pgrep", "-f", appName)
	output, err := cmd.Output()
	if err != nil {
		return pids, nil // No processes found
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil {
			pids = append(pids, pid)
		}
	}
	
	return pids, nil
}