package dns

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

	// Skip DNS hijacking for now (too aggressive)
	// TODO: Implement gentler DNS integration
	log.Println("DNS server started - system DNS hijacking disabled for stability")
	
	// Skip DNS monitor for now
	// TODO: Implement after DNS hijacking is stable

	return nil
}

func (dm *DNSManager) Stop() error {
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

func (dm *DNSManager) hijackSystemDNS() error {
	switch runtime.GOOS {
	case "linux":
		return dm.hijackLinuxDNS()
	case "darwin":
		return dm.hijackMacDNS()
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

func (dm *DNSManager) hijackLinuxDNS() error {
	// Try systemd-resolve first
	if dm.hasSystemdResolve() {
		return dm.setSystemdDNS("127.0.0.1")
	}
	// Fallback to NetworkManager
	if dm.hasNetworkManager() {
		return dm.setNetworkManagerDNS("127.0.0.1")
	}
	// Last resort: modify resolv.conf
	return dm.setResolvConf("127.0.0.1")
}

func (dm *DNSManager) restoreLinuxDNS() error {
	if len(dm.originalDNS) == 0 {
		dm.originalDNS = []string{"8.8.8.8", "8.8.4.4"}
	}
	
	// Try systemd-resolve first
	if dm.hasSystemdResolve() {
		return dm.restoreSystemdDNS()
	}
	// Fallback to NetworkManager
	if dm.hasNetworkManager() {
		return dm.restoreNetworkManagerDNS()
	}
	// Last resort: restore resolv.conf
	return dm.restoreResolvConf()
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
	content := fmt.Sprintf("# Keyphy DNS hijack\nnameserver %s\n", dns)
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

func (dm *DNSManager) hijackMacDNS() error {
	// Get all network interfaces
	interfaces, err := dm.getMacNetworkInterfaces()
	if err != nil {
		return err
	}
	
	// Set DNS for each active interface
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-setdnsservers", iface, "127.0.0.1")
		cmd.Run() // Continue even if one fails
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