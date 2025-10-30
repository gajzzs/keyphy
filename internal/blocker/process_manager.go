package blocker

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessManager struct{}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

func (pm *ProcessManager) FindProcessesByPath(execPath string) ([]*process.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var matches []*process.Process
	baseName := filepath.Base(execPath)

	for _, proc := range processes {
		exe, _ := proc.Exe()
		name, _ := proc.Name()
		
		if exe == execPath || name == baseName || strings.Contains(exe, execPath) {
			matches = append(matches, proc)
		}
	}

	return matches, nil
}

func (pm *ProcessManager) KillProcesses(processes []*process.Process) error {
	for _, proc := range processes {
		if err := proc.Terminate(); err != nil {
			proc.Kill()
		}
		
		// Verify termination
		time.Sleep(100 * time.Millisecond)
		if exists, _ := proc.IsRunning(); exists {
			proc.Kill()
		}
	}
	return nil
}

func (pm *ProcessManager) MonitorAndKill(execPath string, stopChan <-chan bool) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			if procs, err := pm.FindProcessesByPath(execPath); err == nil {
				pm.KillProcesses(procs)
			}
		}
	}
}