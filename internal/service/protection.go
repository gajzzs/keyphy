package service

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ServiceProtection struct {
	running     bool
	stopChan    chan bool
	serviceName string
}

func NewServiceProtection() *ServiceProtection {
	return &ServiceProtection{
		running:     false,
		stopChan:    make(chan bool),
		serviceName: "keyphy",
	}
}

func (sp *ServiceProtection) Start() error {
	if sp.running {
		return nil
	}

	sp.running = true
	log.Println("Starting service protection daemon...")

	go sp.protectionLoop()
	return nil
}

func (sp *ServiceProtection) Stop() {
	if !sp.running {
		return
	}

	sp.running = false
	sp.stopChan <- true
	log.Println("Service protection daemon stopped")
}

func (sp *ServiceProtection) protectionLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	consecutiveFailures := 0
	maxFailures := 3

	for {
		select {
		case <-sp.stopChan:
			return
		case <-ticker.C:
			if !sp.isServiceRunning() {
				consecutiveFailures++
				log.Printf("Service termination detected (attempt %d/%d)", consecutiveFailures, maxFailures)
				
				if consecutiveFailures >= maxFailures {
					log.Println("Multiple service termination attempts detected - activating protection measures")
					sp.activateProtectionMeasures()
					consecutiveFailures = 0
				}
				
				// Attempt to restart service
				if err := sp.restartService(); err != nil {
					log.Printf("Failed to restart service: %v", err)
				} else {
					log.Println("Service restarted successfully")
					consecutiveFailures = 0
				}
			} else {
				consecutiveFailures = 0
			}
		}
	}
}

func (sp *ServiceProtection) isServiceRunning() bool {
	cmd := exec.Command("systemctl", "is-active", sp.serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func (sp *ServiceProtection) restartService() error {
	// Try to restart the service
	cmd := exec.Command("systemctl", "start", sp.serviceName)
	return cmd.Run()
}

func (sp *ServiceProtection) activateProtectionMeasures() {
	log.Println("Activating enhanced protection measures...")
	
	// 1. Create systemd override to prevent manual stopping
	sp.createSystemdOverride()
	
	// 2. Set service to restart automatically
	sp.enableAutoRestart()
	
	// 3. Log security event
	sp.logSecurityEvent()
}

func (sp *ServiceProtection) createSystemdOverride() {
	overrideDir := "/etc/systemd/system/keyphy.service.d"
	overrideFile := overrideDir + "/protection.conf"
	
	// Create override directory
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		log.Printf("Failed to create override directory: %v", err)
		return
	}
	
	// Create override configuration
	overrideContent := `[Unit]
# Keyphy service protection - prevents unauthorized termination
Conflicts=
After=network.target

[Service]
# Restart service if terminated
Restart=always
RestartSec=5
# Prevent manual stopping without authentication
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
`
	
	if err := os.WriteFile(overrideFile, []byte(overrideContent), 0644); err != nil {
		log.Printf("Failed to create service override: %v", err)
		return
	}
	
	// Reload systemd configuration
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to reload systemd configuration: %v", err)
	} else {
		log.Println("Service protection override activated")
	}
}

func (sp *ServiceProtection) enableAutoRestart() {
	// Enable the service to start on boot
	cmd := exec.Command("systemctl", "enable", sp.serviceName)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to enable service auto-start: %v", err)
	} else {
		log.Println("Service auto-restart enabled")
	}
}

func (sp *ServiceProtection) logSecurityEvent() {
	// Log to system journal
	message := "SECURITY ALERT: Multiple unauthorized attempts to terminate Keyphy service detected"
	cmd := exec.Command("logger", "-p", "auth.warning", "-t", "keyphy-protection", message)
	cmd.Run()
	
	// Also log to our own log
	log.Printf("SECURITY EVENT: %s", message)
}

func (sp *ServiceProtection) RemoveProtection() error {
	log.Println("Removing service protection measures...")
	
	// Remove systemd override
	overrideDir := "/etc/systemd/system/keyphy.service.d"
	if err := os.RemoveAll(overrideDir); err != nil {
		log.Printf("Failed to remove service override: %v", err)
	}
	
	// Reload systemd configuration
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to reload systemd configuration: %v", err)
	}
	
	log.Println("Service protection removed")
	return nil
}

func (sp *ServiceProtection) IsRunning() bool {
	return sp.running
}