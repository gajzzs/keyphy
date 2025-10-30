package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
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
		Name:        "com.keyphy.daemon",
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
	return sm.service.Install()
}

func (sm *ServiceManager) Uninstall() error {
	return sm.service.Uninstall()
}

func (sm *ServiceManager) Start() error {
	return sm.service.Start()
}

func (sm *ServiceManager) Stop() error {
	return sm.service.Stop()
}

func (sm *ServiceManager) Restart() error {
	return sm.service.Restart()
}

func (sm *ServiceManager) Status() (string, error) {
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

// GetServiceConfigPath returns platform-specific service config path
func GetServiceConfigPath() string {
	switch service.Platform() {
	case "linux-systemd":
		return "/etc/systemd/system/com.keyphy.daemon.service"
	case "darwin-launchd":
		return "/Library/LaunchDaemons/com.keyphy.daemon.plist"
	case "windows-service":
		return "Registry: HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\com.keyphy.daemon"
	default:
		return "Unknown platform"
	}
}