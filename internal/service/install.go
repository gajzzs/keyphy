package service

import (
	"fmt"
	"github.com/kardianos/service"
)

func InstallService() error {
	s, err := getService()
	if err != nil {
		return err
	}
	if err := s.Install(); err != nil {
		return fmt.Errorf("failed to install service: %v", err)
	}
	fmt.Println("Service installed and enabled for auto-start")
	return nil
}

func UninstallService() error {
	s, err := getService()
	if err != nil {
		return err
	}
	if err := s.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %v", err)
	}
	fmt.Println("Service uninstalled")
	return nil
}

func getService() (service.Service, error) {
	daemon := NewDaemon()
	config := &service.Config{
		Name:        "keyphy",
		DisplayName: "Keyphy Access Control Daemon",
		Description: "Keyphy blocks access to applications, websites, and files until authenticated with external USB device",
	}
	return service.New(daemon, config)
}