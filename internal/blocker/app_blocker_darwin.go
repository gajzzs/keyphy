//go:build darwin
// +build darwin

package blocker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type AppBlocker struct {
	blockedApps map[string]bool
	integrity   map[string]string
	tamperCheck map[string]time.Time
	processMgr  *ProcessManager
	stopChans   map[string]chan bool
}

func NewAppBlocker() *AppBlocker {
	return &AppBlocker{
		blockedApps: make(map[string]bool),
		integrity:   make(map[string]string),
		tamperCheck: make(map[string]time.Time),
		processMgr:  NewProcessManager(),
		stopChans:   make(map[string]chan bool),
	}
}

func (ab *AppBlocker) BlockApp(appName string) error {
	ab.blockedApps[appName] = true
	
	if err := ab.killProcesses(appName); err != nil {
		return fmt.Errorf("failed to kill existing processes: %v", err)
	}
	
	return ab.setupDBusMonitoring(appName)
}

func (ab *AppBlocker) UnblockApp(appName string) error {
	delete(ab.blockedApps, appName)
	
	// Stop monitoring
	if stopChan, exists := ab.stopChans[appName]; exists {
		close(stopChan)
		delete(ab.stopChans, appName)
	}
	
	return ab.restoreOriginalExecutable(appName)
}

func (ab *AppBlocker) IsBlocked(appName string) bool {
	return ab.blockedApps[appName]
}

func (ab *AppBlocker) killProcesses(execPath string) error {
	procs, err := ab.processMgr.FindProcessesByPath(execPath)
	if err != nil {
		return err
	}
	return ab.processMgr.KillProcesses(procs)
}

func (ab *AppBlocker) GetRunningProcesses(appName string) ([]int, error) {
	var pids []int
	
	searchTerm := appName
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		searchTerm = parts[1]
	}
	
	if strings.HasPrefix(searchTerm, "/") {
		baseName := filepath.Base(searchTerm)
		searchTerms := []string{searchTerm, baseName}
		
		for _, term := range searchTerms {
			processPids, err := ab.findProcessesByName(term)
			if err == nil {
				pids = append(pids, processPids...)
			}
		}
	} else {
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
		procName, err := proc.Name()
		if err != nil {
			continue
		}
		
		cmdline, err := proc.Cmdline()
		if err != nil {
			cmdline = procName
		}
		
		exe, _ := proc.Exe()
		
		if procName == name || strings.Contains(cmdline, name) || strings.Contains(exe, name) {
			pids = append(pids, int(proc.Pid))
		}
		
		if strings.Contains(name, ".app/Contents/MacOS/") {
			appName := filepath.Base(name)
			if procName == appName || strings.Contains(exe, appName) {
				pids = append(pids, int(proc.Pid))
			}
		}
	}
	
	return pids, nil
}

func (ab *AppBlocker) BlockProcessLaunch(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}
	
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate process %d: %v", pid, err)
	}
	
	return nil
}

func (ab *AppBlocker) calculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (ab *AppBlocker) restoreOriginalExecutable(appName string) error {
	var execPath string
	
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		execPath = parts[1]
	} else if strings.HasPrefix(appName, "/") {
		execPath = appName
	} else {
		var err error
		execPath, err = exec.LookPath(appName)
		if err != nil {
			return nil
		}
	}
	
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); err == nil {
		if err := exec.Command("cp", "-p", backupPath, execPath).Run(); err != nil {
			return fmt.Errorf("failed to restore executable: %v", err)
		}
		
		os.Remove(backupPath)
		delete(ab.integrity, execPath)
		delete(ab.tamperCheck, execPath)
	}
	
	return nil
}

func (ab *AppBlocker) VerifyIntegrity() error {
	return nil // Simplified for macOS
}

func (ab *AppBlocker) EnforceBlocks() error {
	for appName := range ab.blockedApps {
		if err := ab.createBlockingWrapper(appName); err != nil {
			return fmt.Errorf("failed to re-enforce block for %s: %v", appName, err)
		}
	}
	return nil
}

// macOS-specific app blocking implementation
func (ab *AppBlocker) createBlockingWrapper(appName string) error {
	var execPath string
	var displayName string
	
	if strings.Contains(appName, ":") {
		parts := strings.SplitN(appName, ":", 2)
		displayName = parts[0]
		execPath = parts[1]
	} else if strings.HasPrefix(appName, "/") {
		execPath = appName
		displayName = filepath.Base(appName)
	} else {
		displayName = appName
		var err error
		execPath, err = exec.LookPath(appName)
		if err != nil {
			return fmt.Errorf("executable %s not found", appName)
		}
	}

	// For macOS apps, we need to handle .app bundles differently
	if strings.HasSuffix(execPath, ".app/Contents/MacOS/Obsidian") || strings.Contains(execPath, ".app/Contents/MacOS/") {
		return ab.blockMacOSApp(execPath, displayName)
	}

	// For regular executables, use the Linux approach
	return ab.createLinuxBlockingWrapper(execPath, displayName)
}

// blockMacOSApp handles macOS .app bundles
func (ab *AppBlocker) blockMacOSApp(execPath, displayName string) error {
	// Make the executable non-executable
	if err := os.Chmod(execPath, 0644); err != nil {
		return fmt.Errorf("failed to remove execute permission: %v", err)
	}

	// Create a blocking script in its place
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// Backup original
		if err := exec.Command("cp", "-p", execPath, backupPath).Run(); err != nil {
			return fmt.Errorf("failed to backup executable: %v", err)
		}
	}

	blockScript := fmt.Sprintf(`#!/bin/bash
echo "Access to %s is blocked by Keyphy"
echo "Connect your authentication device and run: keyphy unlock"
exit 1
`, displayName)

	if err := os.WriteFile(execPath, []byte(blockScript), 0755); err != nil {
		return fmt.Errorf("failed to create blocking script: %v", err)
	}

	return nil
}

// createLinuxBlockingWrapper is the original Linux implementation
func (ab *AppBlocker) createLinuxBlockingWrapper(execPath, displayName string) error {
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		hash, err := ab.calculateFileHash(execPath)
		if err != nil {
			return fmt.Errorf("failed to calculate file hash: %v", err)
		}
		ab.integrity[execPath] = hash
		
		if err := exec.Command("cp", "-p", execPath, backupPath).Run(); err != nil {
			return fmt.Errorf("failed to backup executable: %v", err)
		}
	}

	blockScript := fmt.Sprintf(`#!/bin/bash
echo "Access to %s is blocked by Keyphy - Physical device authentication required"
echo "Connect your authentication device and run: keyphy unlock"
exit 1
`, displayName)

	if err := os.WriteFile(execPath, []byte(blockScript), 0755); err != nil {
		return fmt.Errorf("failed to create blocking wrapper: %v", err)
	}

	return nil
}

func (ab *AppBlocker) setupDBusMonitoring(appName string) error {
	execPath := ab.getExecPath(appName)
	
	if err := ab.killProcesses(execPath); err != nil {
		return fmt.Errorf("failed to kill existing processes: %v", err)
	}
	
	if err := ab.createBlockingWrapper(appName); err != nil {
		return fmt.Errorf("failed to create blocking wrapper: %v", err)
	}

	// Start continuous monitoring
	stopChan := make(chan bool)
	ab.stopChans[appName] = stopChan
	go ab.processMgr.MonitorAndKill(execPath, stopChan)
	
	return nil
}

func (ab *AppBlocker) getExecPath(appName string) string {
	if strings.Contains(appName, ":") {
		return strings.SplitN(appName, ":", 2)[1]
	}
	if strings.HasPrefix(appName, "/") {
		return appName
	}
	if path, err := exec.LookPath(appName); err == nil {
		return path
	}
	return appName
}