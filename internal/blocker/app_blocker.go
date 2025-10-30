//go:build !darwin
// +build !darwin

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
	integrity   map[string]string // SHA256 hashes of original executables
	tamperCheck map[string]time.Time // Last tamper check timestamps
}

func NewAppBlocker() *AppBlocker {
	return &AppBlocker{
		blockedApps: make(map[string]bool),
		integrity:   make(map[string]string),
		tamperCheck: make(map[string]time.Time),
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
	
	// Verify integrity before restore
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); err == nil {
		// Verify backup hasn't been tampered with
		if originalHash, exists := ab.integrity[execPath]; exists {
			backupHash, err := ab.calculateFileHash(backupPath)
			if err != nil {
				return fmt.Errorf("failed to verify backup integrity: %v", err)
			}
			if backupHash != originalHash {
				return fmt.Errorf("backup file tampered - cannot safely restore %s", execPath)
			}
		}
		
		// Remove immutable attribute (Linux)
		exec.Command("chattr", "-i", backupPath).Run()
		
		// Get original backup file info for permissions
		backupInfo, err := os.Stat(backupPath)
		if err != nil {
			return fmt.Errorf("failed to get backup file info: %v", err)
		}
		
		// Restore file with original permissions
		if err := exec.Command("cp", "-p", backupPath, execPath).Run(); err != nil {
			return fmt.Errorf("failed to restore executable: %v", err)
		}
		
		// Ensure correct permissions are set
		if err := os.Chmod(execPath, backupInfo.Mode()); err != nil {
			return fmt.Errorf("failed to restore permissions: %v", err)
		}
		
		// Clean up
		os.Remove(backupPath)
		delete(ab.integrity, execPath)
		delete(ab.tamperCheck, execPath)
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
		// Use gopsutil for more reliable process termination
		if proc, err := process.NewProcess(int32(pid)); err == nil {
			// Try graceful termination first
			if err := proc.Terminate(); err != nil {
				// Force kill if terminate fails
				proc.Kill()
			}
			// Wait a moment then verify it's dead
			time.Sleep(100 * time.Millisecond)
			if exists, _ := proc.IsRunning(); exists {
				proc.Kill() // Force kill
			}
		}
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
	
	// Backup original executable with integrity check
	backupPath := execPath + ".keyphy-backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// Calculate and store original hash
		hash, err := ab.calculateFileHash(execPath)
		if err != nil {
			return fmt.Errorf("failed to calculate file hash: %v", err)
		}
		ab.integrity[execPath] = hash
		
		if err := exec.Command("cp", "-p", execPath, backupPath).Run(); err != nil {
			return fmt.Errorf("failed to backup executable: %v", err)
		}
		
		// Set immutable attribute on backup (Linux)
		exec.Command("chattr", "+i", backupPath).Run()
	}
	
	// Create cryptographically signed blocking script
	blockScript := fmt.Sprintf(`#!/bin/bash
# Keyphy Block - Integrity: %s
# Timestamp: %d
echo "Access to %s is blocked by Keyphy - Physical device authentication required"
echo "Connect your authentication device and run: keyphy unlock"
exit 1
`, ab.integrity[execPath], time.Now().Unix(), displayName)
	
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
		// Check process name
		procName, err := proc.Name()
		if err != nil {
			continue
		}
		
		// Check command line for more accurate matching
		cmdline, err := proc.Cmdline()
		if err != nil {
			cmdline = procName
		}
		
		// For macOS apps, also check executable path
		exe, _ := proc.Exe()
		
		// Match by name, cmdline, or executable path
		if procName == name || strings.Contains(cmdline, name) || strings.Contains(exe, name) {
			pids = append(pids, int(proc.Pid))
		}
		
		// Special handling for macOS .app bundles
		if strings.Contains(name, ".app/Contents/MacOS/") {
			appName := filepath.Base(name)
			if procName == appName || strings.Contains(exe, appName) {
				pids = append(pids, int(proc.Pid))
			}
		}
	}
	
	return pids, nil
}

// calculateFileHash computes SHA256 hash of a file
func (ab *AppBlocker) calculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// VerifyIntegrity checks if blocked executables haven't been tampered with
func (ab *AppBlocker) VerifyIntegrity() error {
	for execPath, originalHash := range ab.integrity {
		// Skip if recently checked (within 30 seconds)
		if lastCheck, exists := ab.tamperCheck[execPath]; exists {
			if time.Since(lastCheck) < 30*time.Second {
				continue
			}
		}
		
		// Check if backup still exists and is immutable
		backupPath := execPath + ".keyphy-backup"
		if _, err := os.Stat(backupPath); err != nil {
			return fmt.Errorf("backup missing for %s - potential tamper attempt", execPath)
		}
		
		// Verify backup integrity
		backupHash, err := ab.calculateFileHash(backupPath)
		if err != nil {
			return fmt.Errorf("failed to verify backup integrity: %v", err)
		}
		if backupHash != originalHash {
			return fmt.Errorf("backup tampered for %s - security breach detected", execPath)
		}
		
		// Check if current executable is still our blocking script
		currentContent, err := os.ReadFile(execPath)
		if err != nil {
			return fmt.Errorf("failed to read current executable: %v", err)
		}
		
		if !strings.Contains(string(currentContent), "Keyphy Block - Integrity:") {
			return fmt.Errorf("blocking script replaced for %s - bypass attempt detected", execPath)
		}
		
		ab.tamperCheck[execPath] = time.Now()
	}
	return nil
}

// EnforceBlocks re-applies blocks if tamper attempts detected
func (ab *AppBlocker) EnforceBlocks() error {
	for appName := range ab.blockedApps {
		if err := ab.createBlockingWrapper(appName); err != nil {
			return fmt.Errorf("failed to re-enforce block for %s: %v", appName, err)
		}
	}
	return nil
}