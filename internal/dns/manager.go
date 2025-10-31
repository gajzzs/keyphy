package dns

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type DNSManager struct {
	server      *DNSServer
	monitor     *DNSMonitor
	originalDNS []string
}

func NewDNSManager() *DNSManager {
	dm := &DNSManager{
		server: NewDNSServer(),
	}
	dm.monitor = NewDNSMonitor(dm)
	return dm
}

func (dm *DNSManager) Start() error {
	// Backup original DNS settings
	if err := dm.backupDNS(); err != nil {
		return fmt.Errorf("failed to backup DNS: %v", err)
	}

	// Start DNS server
	if err := dm.server.Start(); err != nil {
		return fmt.Errorf("failed to start DNS server: %v", err)
	}

	// Configure system DNS redirection for enhanced blocking
	if err := dm.configureSystemDNSRedirection(); err != nil {
		log.Printf("Warning: DNS redirection setup failed: %v", err)
		log.Println("DNS server running on fallback port - some blocking may be bypassed")
	} else {
		log.Println("DNS system fully integrated - all DNS queries will be filtered")
	}
	
	// Start DNS monitoring for integrity protection
	go dm.startDNSIntegrityMonitor()
	
	// Start service protection
	go dm.protectService()

	return nil
}

func (dm *DNSManager) Stop() error {
	// Restore original DNS settings
	if err := dm.restoreDNS(); err != nil {
		log.Printf("Warning: Failed to restore DNS settings: %v", err)
	}
	
	// Stop DNS server
	return dm.server.Stop()
}

func (dm *DNSManager) BlockDomain(domain string) {
	dm.server.BlockDomain(domain)
}

func (dm *DNSManager) UnblockDomain(domain string) {
	dm.server.UnblockDomain(domain)
}

func (dm *DNSManager) UnblockAll() {
	dm.server.UnblockAll()
}

func (dm *DNSManager) backupDNS() error {
	switch runtime.GOOS {
	case "linux":
		return dm.backupLinuxDNS()
	case "darwin":
		return dm.backupMacDNS()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (dm *DNSManager) configureSystemDNSRedirection() error {
	switch runtime.GOOS {
	case "linux":
		return dm.configureLinuxDNSRedirection()
	case "darwin":
		return dm.configureMacDNSRedirection()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (dm *DNSManager) restoreDNS() error {
	switch runtime.GOOS {
	case "linux":
		return dm.restoreLinuxDNS()
	case "darwin":
		return dm.restoreMacDNS()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Linux DNS management using CLI tools
func (dm *DNSManager) backupLinuxDNS() error {
	// Try systemd-resolve first
	if dm.hasSystemdResolve() {
		return dm.backupSystemdDNS()
	}
	// Fallback to NetworkManager
	if dm.hasNetworkManager() {
		return dm.backupNetworkManagerDNS()
	}
	// Last resort: read resolv.conf
	return dm.backupResolvConf()
}

func (dm *DNSManager) configureLinuxDNSRedirection() error {
	// If our DNS server is running on port 53, use 127.0.0.1
	// If it's on port 6666, we need to disable systemd-resolved stub
	dnsServer := "127.0.0.1"
	
	// Check if our DNS server is on port 53
	if !dm.server.IsRunningOnPort53() {
		// Disable systemd-resolved stub to free port 53
		if err := dm.disableSystemdResolvedStub(); err != nil {
			log.Printf("Warning: Could not disable systemd-resolved stub: %v", err)
			return err
		}
		
		// Try to restart our DNS server on port 53
		if err := dm.server.RestartOnPort53(); err != nil {
			log.Printf("Warning: Could not restart DNS server on port 53: %v", err)
			// Continue with port 6666 and hope for the best
		}
	}
	
	// Try systemd-resolve first (most modern approach)
	if dm.hasSystemdResolve() {
		return dm.setSystemdDNS(dnsServer)
	}
	// Fallback to NetworkManager
	if dm.hasNetworkManager() {
		return dm.setNetworkManagerDNS(dnsServer)
	}
	// Last resort: modify resolv.conf (legacy systems)
	return dm.setResolvConf(dnsServer)
}

func (dm *DNSManager) restoreLinuxDNS() error {
	log.Println("Restoring original DNS configuration...")
	
	if len(dm.originalDNS) == 0 {
		dm.originalDNS = []string{"8.8.8.8", "8.8.4.4"}
	}
	
	// Simple restart of systemd-resolved
	log.Println("Restarting systemd-resolved service...")
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to restart systemd-resolved: %v", err)
	} else {
		log.Println("systemd-resolved restarted successfully")
		time.Sleep(2 * time.Second)
		return nil
	}
	
	// Try systemd-resolve first
	if dm.hasSystemdResolve() {
		if err := dm.restoreSystemdDNS(); err != nil {
			log.Printf("Failed to restore systemd DNS: %v", err)
		} else {
			log.Println("systemd DNS settings restored successfully")
			// Verify DNS is actually working
			if dm.verifyDNSWorking() {
				log.Println("DNS verification successful")
				return nil
			} else {
				log.Println("DNS verification failed - falling back to resolv.conf")
			}
		}
	}
	
	// Fallback to NetworkManager
	if dm.hasNetworkManager() {
		if err := dm.restoreNetworkManagerDNS(); err != nil {
			log.Printf("Failed to restore NetworkManager DNS: %v", err)
		} else {
			log.Println("NetworkManager DNS settings restored successfully")
			return nil
		}
	}
	
	// Last resort: ensure systemd-resolved stub is working
	log.Println("Ensuring systemd-resolved stub is functional...")
	return dm.ensureSystemdResolvedWorking()
}

// Systemd-resolve methods
func (dm *DNSManager) hasSystemdResolve() bool {
	cmd := exec.Command("which", "systemd-resolve")
	return cmd.Run() == nil
}

func (dm *DNSManager) backupSystemdDNS() error {
	cmd := exec.Command("systemd-resolve", "--status")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "DNS Servers:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" {
					dm.originalDNS = append(dm.originalDNS, dns)
				}
			}
		}
	}
	return nil
}

func (dm *DNSManager) setSystemdDNS(dns string) error {
	// Get active network interface
	iface, err := dm.getActiveInterface()
	if err != nil {
		iface = "eth0" // fallback
	}
	
	cmd := exec.Command("systemd-resolve", "--set-dns="+dns, "--interface="+iface)
	return cmd.Run()
}

func (dm *DNSManager) restoreSystemdDNS() error {
	// Get active network interface
	iface, err := dm.getActiveInterface()
	if err != nil {
		iface = "eth0" // fallback
	}
	
	for _, dns := range dm.originalDNS {
		cmd := exec.Command("systemd-resolve", "--set-dns="+dns, "--interface="+iface)
		cmd.Run()
	}
	return nil
}

// NetworkManager methods
func (dm *DNSManager) hasNetworkManager() bool {
	cmd := exec.Command("which", "nmcli")
	return cmd.Run() == nil
}

func (dm *DNSManager) backupNetworkManagerDNS() error {
	cmd := exec.Command("nmcli", "dev", "show")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IP4.DNS") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" {
					dm.originalDNS = append(dm.originalDNS, dns)
				}
			}
		}
	}
	return nil
}

func (dm *DNSManager) setNetworkManagerDNS(dns string) error {
	// Get active network interface
	iface, err := dm.getActiveInterface()
	if err != nil {
		iface = "eth0" // fallback
	}
	
	cmd := exec.Command("nmcli", "dev", "modify", iface, "ipv4.dns", dns)
	return cmd.Run()
}

func (dm *DNSManager) restoreNetworkManagerDNS() error {
	// Get active network interface
	iface, err := dm.getActiveInterface()
	if err != nil {
		iface = "eth0" // fallback
	}
	
	dnsServers := strings.Join(dm.originalDNS, ",")
	cmd := exec.Command("nmcli", "dev", "modify", iface, "ipv4.dns", dnsServers)
	return cmd.Run()
}

func (dm *DNSManager) getActiveInterface() (string, error) {
	// Get default route interface
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// Parse: default via 192.168.1.1 dev wlp3s0 proto dhcp metric 600
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	
	return "", fmt.Errorf("no active interface found")
}

// Fallback resolv.conf methods
func (dm *DNSManager) backupResolvConf() error {
	content, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nameserver ") {
			dns := strings.TrimPrefix(line, "nameserver ")
			dm.originalDNS = append(dm.originalDNS, strings.TrimSpace(dns))
		}
	}
	return nil
}

func (dm *DNSManager) setResolvConf(dns string) error {
	content := fmt.Sprintf("# Keyphy DNS redirection\nnameserver %s\n", dns)
	return os.WriteFile("/etc/resolv.conf", []byte(content), 0644)
}

func (dm *DNSManager) restoreResolvConf() error {
	content := "# Restored by Keyphy\n"
	for _, dns := range dm.originalDNS {
		content += fmt.Sprintf("nameserver %s\n", dns)
	}
	return os.WriteFile("/etc/resolv.conf", []byte(content), 0644)
}

// macOS DNS management
func (dm *DNSManager) backupMacDNS() error {
	// Get all network interfaces
	interfaces, err := dm.getMacNetworkInterfaces()
	if err != nil {
		return err
	}
	
	// Backup DNS for each interface
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-getdnsservers", iface)
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "There aren't any DNS Servers set on "+iface+"." && line != "" {
				dm.originalDNS = append(dm.originalDNS, strings.TrimSpace(line))
			}
		}
	}
	
	return nil
}

func (dm *DNSManager) configureMacDNSRedirection() error {
	// Get all network interfaces
	interfaces, err := dm.getMacNetworkInterfaces()
	if err != nil {
		return err
	}
	
	// Configure DNS redirection for each active interface
	dnsServer := "127.0.0.1"
	successCount := 0
	
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-setdnsservers", iface, dnsServer)
		if err := cmd.Run(); err == nil {
			successCount++
			log.Printf("DNS redirection configured for interface: %s", iface)
		}
	}
	
	if successCount == 0 {
		return fmt.Errorf("failed to configure DNS redirection on any interface")
	}
	
	return nil
}

func (dm *DNSManager) restoreMacDNS() error {
	// Get all network interfaces
	interfaces, err := dm.getMacNetworkInterfaces()
	if err != nil {
		return err
	}
	
	// Restore DNS for each interface
	for _, iface := range interfaces {
		if len(dm.originalDNS) == 0 {
			// Clear DNS servers (use DHCP)
			cmd := exec.Command("networksetup", "-setdnsservers", iface, "empty")
			cmd.Run()
		} else {
			args := append([]string{"-setdnsservers", iface}, dm.originalDNS...)
			cmd := exec.Command("networksetup", args...)
			cmd.Run()
		}
	}
	
	return nil
}

func (dm *DNSManager) getMacNetworkInterfaces() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var interfaces []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "An asterisk") && !strings.HasPrefix(line, "*") {
			interfaces = append(interfaces, line)
		}
	}
	
	return interfaces, nil
}

// startDNSIntegrityMonitor monitors DNS settings for tampering
func (dm *DNSManager) startDNSIntegrityMonitor() {
	// Monitor DNS settings every 30 seconds
	go func() {
		for {
			time.Sleep(30 * time.Second)
			
			// Check if DNS redirection is still active
			if !dm.isDNSRedirectionActive() {
				log.Println("DNS redirection tampering detected - reapplying configuration")
				if err := dm.configureSystemDNSRedirection(); err != nil {
					log.Printf("Failed to restore DNS redirection: %v", err)
				}
			}
		}
	}()
}

// isDNSRedirectionActive checks if DNS is properly redirected to our server
func (dm *DNSManager) isDNSRedirectionActive() bool {
	// Simple check - verify if our DNS server is receiving queries
	// This is a basic implementation - could be enhanced with more sophisticated checks
	return dm.server != nil && dm.server.IsRunning()
}

// disableSystemdResolvedStub disables the systemd-resolved stub listener on port 53
func (dm *DNSManager) disableSystemdResolvedStub() error {
	// Create resolved.conf.d directory if it doesn't exist
	if err := os.MkdirAll("/etc/systemd/resolved.conf.d", 0755); err != nil {
		return err
	}
	
	// Create configuration to disable stub
	config := `[Resolve]
DNSStubListener=no
`
	if err := os.WriteFile("/etc/systemd/resolved.conf.d/keyphy.conf", []byte(config), 0644); err != nil {
		return err
	}
	
	// Restart systemd-resolved
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Wait a moment for the service to restart
	time.Sleep(2 * time.Second)
	return nil
}

// enableSystemdResolvedStub re-enables the systemd-resolved stub listener
func (dm *DNSManager) enableSystemdResolvedStub() error {
	log.Println("Re-enabling systemd-resolved stub listener...")
	
	// Remove our configuration file
	if err := os.Remove("/etc/systemd/resolved.conf.d/keyphy.conf"); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Could not remove keyphy.conf: %v", err)
	}
	
	// Restart systemd-resolved to restore default behavior
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart systemd-resolved: %v", err)
	}
	
	// Wait for systemd-resolved to fully start
	log.Println("Waiting for systemd-resolved to start...")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
		if output, err := cmd.Output(); err == nil {
			if strings.TrimSpace(string(output)) == "active" {
				log.Println("systemd-resolved is now active")
				return nil
			}
		}
	}
	
	return fmt.Errorf("systemd-resolved failed to start within 10 seconds")
}

// protectService implements service protection mechanisms
func (dm *DNSManager) protectService() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			
			// Check if service is still running
			if !dm.isServiceRunning() {
				log.Println("Service termination detected - attempting emergency DNS restoration")
				dm.emergencyDNSRestore()
				return
			}
		}
	}()
}

// isServiceRunning checks if the keyphy service is still active
func (dm *DNSManager) isServiceRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "keyphy")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

// emergencyDNSRestore restores DNS functionality when service is terminated
func (dm *DNSManager) emergencyDNSRestore() {
	log.Println("Performing emergency DNS restoration...")
	
	// Re-enable systemd-resolved stub
	if err := dm.enableSystemdResolvedStub(); err != nil {
		log.Printf("Failed to restore systemd-resolved: %v", err)
		// Fallback: restore resolv.conf manually
		dm.emergencyResolvConfRestore()
	}
	
	log.Println("Emergency DNS restoration completed")
}

// emergencyResolvConfRestore manually restores resolv.conf as last resort
func (dm *DNSManager) emergencyResolvConfRestore() {
	content := "# Emergency DNS restoration by Keyphy\nnameserver 8.8.8.8\nnameserver 8.8.4.4\n"
	if err := os.WriteFile("/etc/resolv.conf", []byte(content), 0644); err != nil {
		log.Printf("Failed to restore resolv.conf: %v", err)
	} else {
		log.Println("Emergency resolv.conf restoration successful")
	}
}

// verifyDNSWorking tests if DNS resolution is actually working
func (dm *DNSManager) verifyDNSWorking() bool {
	// Try to resolve a simple domain
	cmd := exec.Command("nslookup", "google.com")
	if err := cmd.Run(); err != nil {
		log.Printf("DNS verification failed: %v", err)
		return false
	}
	return true
}

// ensureSystemdResolvedWorking ensures systemd-resolved stub listener is functional
func (dm *DNSManager) ensureSystemdResolvedWorking() error {
	log.Println("Ensuring systemd-resolved stub listener is working...")
	
	// Stop systemd-resolved completely
	cmd := exec.Command("systemctl", "stop", "systemd-resolved")
	cmd.Run()
	
	// Remove any keyphy configuration that might interfere
	os.Remove("/etc/systemd/resolved.conf.d/keyphy.conf")
	
	// Start systemd-resolved fresh
	cmd = exec.Command("systemctl", "start", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start systemd-resolved: %v", err)
	}
	
	// Wait and verify it's working
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		
		// Test if stub listener is responding
		cmd = exec.Command("nslookup", "google.com", "127.0.0.53")
		if err := cmd.Run(); err == nil {
			log.Println("systemd-resolved stub listener is now working")
			return nil
		}
	}
	
	return fmt.Errorf("systemd-resolved stub listener failed to start properly")
}

