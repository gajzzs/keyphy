package blocker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
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
	var execPath string
	
	// Handle custom path format (name:path) or full path
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		execPath = parts[1]
	} else if strings.HasPrefix(appName, "/") {
		// Full path provided directly
		execPath = appName
	} else {
		// Find the executable path
		var err error
		execPath, err = exec.LookPath(appName)
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
	pids, err := ab.GetRunningProcesses(appName)
	if err != nil {
		return err
	}
	
	for _, pid := range pids {
		process, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		process.Kill()
	}
	return nil
}

func (ab *AppBlocker) setupDBusMonitoring(appName string) error {
	// Create executable wrapper that blocks the app
	return ab.createBlockingWrapper(appName)
}

func (ab *AppBlocker) createBlockingWrapper(appName string) error {
	var execPath string
	var displayName string
	
	// Check if appName contains custom path (format: "name:path")
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		displayName = parts[0]
		execPath = parts[1]
		if _, err := os.Stat(execPath); err != nil {
			return fmt.Errorf("custom executable path %s not found", execPath)
		}
	} else if strings.HasPrefix(appName, "/") {
		// Full path provided directly
		execPath = appName
		displayName = filepath.Base(appName)
		if _, err := os.Stat(execPath); err != nil {
			return fmt.Errorf("executable path %s not found", execPath)
		}
	} else {
		// Find the actual executable path
		displayName = appName
		var err error
		execPath, err = exec.LookPath(appName)
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
`, displayName)
	
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
	
	// Extract actual app name from path if needed
	searchTerm := appName
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		searchTerm = parts[1] // Use the path part
	}
	
	// For AppImages and full paths, search by basename too
	if strings.HasPrefix(searchTerm, "/") {
		baseName := filepath.Base(searchTerm)
		// Search for both full path and basename (for AppImages)
		searchTerms := []string{searchTerm, baseName}
		if strings.Contains(baseName, "VeraCrypt") {
			searchTerms = append(searchTerms, "veracrypt") // AppImages often change case
		}
		
		for _, term := range searchTerms {
			processPids, err := ab.findProcessesByName(term)
			if err == nil {
				pids = append(pids, processPids...)
			}
		}
	} else {
		// For app names, use original logic
		if strings.ContainsAny(searchTerm, ";|&$`(){}[]<>*?~") {
			return nil, fmt.Errorf("invalid characters in app name: %s", searchTerm)
		}
		processPids, err := ab.findProcessesByName(searchTerm)
		if err == nil {
			pids = append(pids, processPids...)
		}
	}
	
	return pids, nil
}

func (ab *AppBlocker) findProcessesByName(name string) ([]int, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}
	
	var pids []int
	for _, proc := range processes {
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}
		
		if strings.Contains(cmdline, name) {
			pids = append(pids, int(proc.Pid))
		}
	}
	
	return pids, nil
}