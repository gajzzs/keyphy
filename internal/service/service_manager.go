package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/kardianos/service"
	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/crypto"
	"github.com/gajzzs/keyphy/internal/device"
)

type ServiceManager struct {
	service service.Service
	daemon  *Daemon
}

type program struct {
	daemon *Daemon
}

func (p *program) Start(s service.Service) error {
	log.Println("Starting keyphy service...")
	return p.daemon.Start(s)
}

func (p *program) Stop(s service.Service) error {
	log.Println("Stopping keyphy service...")
	return p.daemon.Stop(s)
}

func NewServiceManager() (*ServiceManager, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %v", err)
	}

	svcConfig := &service.Config{
		Name:        "keyphy",
		DisplayName: "Keyphy Access Control",
		Description: "System access control using external device authentication",
		Executable:  execPath,
		Arguments:   []string{"service", "run-daemon"},
		Option: service.KeyValue{
			"RunAtLoad": true,
			"KeepAlive": true,
		},
	}

	daemon := NewDaemon()
	prg := &program{daemon: daemon}
	
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %v", err)
	}

	return &ServiceManager{
		service: svc,
		daemon:  daemon,
	}, nil
}

func (sm *ServiceManager) Install() error {
	if os.Geteuid() != 0 {
		_, err := sm.execWithSudo("install")
		return err
	}
	return sm.service.Install()
}

func (sm *ServiceManager) Uninstall() error {
	return sm.service.Uninstall()
}

func (sm *ServiceManager) Start() error {
	if os.Geteuid() != 0 {
		_, err := sm.execWithSudo("start")
		return err
	}
	
	// Check if service is installed first
	if !sm.isServiceInstalled() {
		return fmt.Errorf("service is not installed. Please run 'keyphy service install' first")
	}
	
	return sm.service.Start()
}

func (sm *ServiceManager) Stop() error {
	if os.Geteuid() != 0 {
		_, err := sm.execWithSudo("stop")
		return err
	}
	
	// Require device authentication to stop service
	if !sm.validateDeviceAuth() {
		return fmt.Errorf("authentication device required to stop service")
	}
	
	// Try direct launchctl commands first to handle conflicts
	if err := sm.stopWithLaunchctl(); err == nil {
		return nil
	}
	
	return sm.service.Stop()
}

func (sm *ServiceManager) Restart() error {
	if os.Geteuid() != 0 {
		_, err := sm.execWithSudo("restart")
		return err
	}
	
	// Stop first, then start
	sm.Stop()
	return sm.Start()
}

func (sm *ServiceManager) Status() (string, error) {
	// Check if running as root, if not, re-exec with sudo
	if os.Geteuid() != 0 {
		return sm.execWithSudo("status")
	}
	
	status, err := sm.service.Status()
	if err != nil {
		return "Unknown", err
	}
	
	switch status {
	case service.StatusRunning:
		return "Running", nil
	case service.StatusStopped:
		return "Stopped", nil
	case service.StatusUnknown:
		return "Unknown", nil
	default:
		return fmt.Sprintf("Status(%d)", int(status)), nil
	}
}

func (sm *ServiceManager) Run() error {
	return sm.service.Run()
}

func (sm *ServiceManager) Watchdog() error {
	if os.Geteuid() != 0 {
		_, err := sm.execWithSudo("watchdog")
		return err
	}
	
	if !sm.isServiceInstalled() {
		return fmt.Errorf("service is not installed. Please run 'keyphy service install' first")
	}
	
	fmt.Println("Starting keyphy service watchdog...")
	go sm.runWatchdog()
	fmt.Println("Watchdog started - service will auto-restart if killed")
	return nil
}

func (sm *ServiceManager) runWatchdog() {
	for {
		// Check service status every 10 seconds
		time.Sleep(10 * time.Second)
		
		status, err := sm.service.Status()
		if err != nil || status != service.StatusRunning {
			fmt.Println("Service stopped unexpectedly, restarting...")
			if err := sm.service.Start(); err != nil {
				fmt.Printf("Failed to restart service: %v\n", err)
			} else {
				fmt.Println("Service restarted successfully")
			}
		}
	}
}

func (sm *ServiceManager) stopWithLaunchctl() error {
	// Try to stop any running keyphy services
	cmds := [][]string{
		{"launchctl", "bootout", "system/com.keyphy.daemon"},
		{"launchctl", "unload", "/Library/LaunchDaemons/com.keyphy.daemon.plist"},
	}
	
	for _, cmdArgs := range cmds {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Run() // Ignore errors, just try to clean up
	}
	
	// Kill processes safely using PID file
	if pid, err := readPidFile(); err == nil {
		if process, err := os.FindProcess(pid); err == nil {
			process.Kill()
		}
	}
	
	return nil
}

func (sm *ServiceManager) isServiceInstalled() bool {
	configPath := GetServiceConfigPath()
	_, err := os.Stat(configPath)
	return err == nil
}

func (sm *ServiceManager) validateDeviceAuth() bool {
	// Import validation logic from commands.go
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
			return err == nil && valid
		}
	}
	return false
}

func (sm *ServiceManager) execWithSudo(action string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}
	
	cmd := exec.Command("sudo", execPath, "service", action)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command failed with exit code %d", exitError.ExitCode())
		}
		return "", err
	}
	
	return string(output), nil
}

// GetServiceConfigPath returns platform-specific service config path
func GetServiceConfigPath() string {
	switch service.Platform() {
	case "linux-systemd":
		return "/etc/systemd/system/keyphy.service"
	case "darwin-launchd":
		return "/Library/LaunchDaemons/com.keyphy.daemon.plist"
	case "windows-service":
		return "Registry: HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\keyphy"
	default:
		return "Unknown platform"
	}
}