package dns

import (
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type DNSMonitor struct {
	manager     *DNSManager
	running     bool
	stopChan    chan bool
	expectedDNS string
}

func NewDNSMonitor(manager *DNSManager) *DNSMonitor {
	return &DNSMonitor{
		manager:     manager,
		running:     false,
		stopChan:    make(chan bool),
		expectedDNS: "127.0.0.1",
	}
}

func (dm *DNSMonitor) Start() error {
	if dm.running {
		return nil
	}

	dm.running = true
	log.Println("Starting DNS monitor service")

	go dm.monitorLoop()
	return nil
}

func (dm *DNSMonitor) Stop() {
	if !dm.running {
		return
	}

	dm.running = false
	dm.stopChan <- true
	log.Println("DNS monitor service stopped")
}

func (dm *DNSMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dm.stopChan:
			return
		case <-ticker.C:
			if err := dm.checkAndFixDNS(); err != nil {
				log.Printf("DNS monitor error: %v", err)
			}
		}
	}
}

func (dm *DNSMonitor) checkAndFixDNS() error {
	currentDNS, err := dm.getCurrentSystemDNS()
	if err != nil {
		return err
	}

	// Check if our DNS server is still the primary DNS
	if !dm.isOurDNSActive(currentDNS) {
		log.Println("DNS tampering detected! Restoring Keyphy DNS server...")
		
		// Re-configure system DNS redirection
		if err := dm.manager.configureSystemDNSRedirection(); err != nil {
			log.Printf("Failed to restore DNS redirection: %v", err)
			return err
		}
		
		log.Println("DNS redirection restored successfully")
	}

	return nil
}

func (dm *DNSMonitor) getCurrentSystemDNS() ([]string, error) {
	switch runtime.GOOS {
	case "linux":
		return dm.getLinuxDNS()
	case "darwin":
		return dm.getMacDNS()
	default:
		return nil, nil
	}
}

func (dm *DNSMonitor) getLinuxDNS() ([]string, error) {
	// Try systemd-resolve first
	if dm.hasSystemdResolve() {
		return dm.getSystemdDNS()
	}
	
	// Fallback to resolv.conf
	return dm.getResolvConfDNS()
}

func (dm *DNSMonitor) hasSystemdResolve() bool {
	cmd := exec.Command("which", "systemd-resolve")
	return cmd.Run() == nil
}

func (dm *DNSMonitor) getSystemdDNS() ([]string, error) {
	cmd := exec.Command("systemd-resolve", "--status")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var dnsServers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "DNS Servers:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" {
					dnsServers = append(dnsServers, dns)
				}
			}
		}
	}

	return dnsServers, nil
}

func (dm *DNSMonitor) getResolvConfDNS() ([]string, error) {
	cmd := exec.Command("cat", "/etc/resolv.conf")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var dnsServers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nameserver ") {
			dns := strings.TrimPrefix(line, "nameserver ")
			dnsServers = append(dnsServers, strings.TrimSpace(dns))
		}
	}

	return dnsServers, nil
}

func (dm *DNSMonitor) getMacDNS() ([]string, error) {
	// Get DNS from primary network interface
	cmd := exec.Command("scutil", "--dns")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var dnsServers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "nameserver[0]") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" {
					dnsServers = append(dnsServers, dns)
				}
			}
		}
	}

	return dnsServers, nil
}

func (dm *DNSMonitor) isOurDNSActive(currentDNS []string) bool {
	if len(currentDNS) == 0 {
		return false
	}

	// Check if 127.0.0.1 is the first/primary DNS server
	return currentDNS[0] == dm.expectedDNS
}

func (dm *DNSMonitor) IsRunning() bool {
	return dm.running
}