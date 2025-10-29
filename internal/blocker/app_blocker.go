package blocker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
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
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Monitor systemd for new service starts
	rule := fmt.Sprintf("type='signal',interface='org.freedesktop.systemd1.Manager',member='JobNew'")
	call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
	if call.Err != nil {
		return call.Err
	}

	// In a real implementation, this would run in a goroutine
	// and continuously monitor for new process launches
	return nil
}

func (ab *AppBlocker) BlockProcessLaunch(pid int) error {
	// Send SIGTERM to block process launch
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	
	return process.Signal(syscall.SIGTERM)
}

func (ab *AppBlocker) GetRunningProcesses(appName string) ([]int, error) {
	var pids []int
	
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