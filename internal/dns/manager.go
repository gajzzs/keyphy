package dns

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type DNSManager struct {
	server      *DNSServer
	originalDNS []string
}

func NewDNSManager() *DNSManager {
	return &DNSManager{
		server: NewDNSServer(),
	}
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

	// Hijack system DNS
	if err := dm.hijackSystemDNS(); err != nil {
		dm.server.Stop()
		return fmt.Errorf("failed to hijack DNS: %v", err)
	}

	return nil
}

func (dm *DNSManager) Stop() error {
	// Restore original DNS
	dm.restoreDNS()
	
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

// Linux DNS management
func (dm *DNSManager) backupLinuxDNS() error {
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

func (dm *DNSManager) hijackLinuxDNS() error {
	// Create new resolv.conf pointing to localhost
	newContent := "# Keyphy DNS hijack\nnameserver 127.0.0.1\n"
	return os.WriteFile("/etc/resolv.conf", []byte(newContent), 0644)
}

func (dm *DNSManager) restoreLinuxDNS() error {
	if len(dm.originalDNS) == 0 {
		// Fallback to common DNS servers
		dm.originalDNS = []string{"8.8.8.8", "8.8.4.4"}
	}
	
	content := "# Restored by Keyphy\n"
	for _, dns := range dm.originalDNS {
		content += fmt.Sprintf("nameserver %s\n", dns)
	}
	
	return os.WriteFile("/etc/resolv.conf", []byte(content), 0644)
}

// macOS DNS management
func (dm *DNSManager) backupMacDNS() error {
	cmd := exec.Command("networksetup", "-getdnsservers", "Wi-Fi")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "There aren't any DNS Servers set on Wi-Fi." {
			dm.originalDNS = append(dm.originalDNS, strings.TrimSpace(line))
		}
	}
	
	return nil
}

func (dm *DNSManager) hijackMacDNS() error {
	cmd := exec.Command("networksetup", "-setdnsservers", "Wi-Fi", "127.0.0.1")
	return cmd.Run()
}

func (dm *DNSManager) restoreMacDNS() error {
	if len(dm.originalDNS) == 0 {
		// Clear DNS servers (use DHCP)
		cmd := exec.Command("networksetup", "-setdnsservers", "Wi-Fi", "empty")
		return cmd.Run()
	}
	
	args := append([]string{"-setdnsservers", "Wi-Fi"}, dm.originalDNS...)
	cmd := exec.Command("networksetup", args...)
	return cmd.Run()
}