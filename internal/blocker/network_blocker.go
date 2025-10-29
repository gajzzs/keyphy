package blocker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type NetworkBlocker struct {
	blockedDomains map[string]bool
}

func NewNetworkBlocker() *NetworkBlocker {
	return &NetworkBlocker{
		blockedDomains: make(map[string]bool),
	}
}

func (nb *NetworkBlocker) BlockWebsite(domain string) error {
	nb.blockedDomains[domain] = true
	
	// Add to /etc/hosts to redirect to localhost
	if err := nb.addToHosts(domain); err != nil {
		return fmt.Errorf("failed to add to hosts: %v", err)
	}
	
	// Block DNS queries for domain
	if err := nb.blockDNS(domain); err != nil {
		return fmt.Errorf("failed to block DNS: %v", err)
	}
	
	return nil
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	delete(nb.blockedDomains, domain)
	
	// Remove from /etc/hosts
	if err := nb.removeFromHosts(domain); err != nil {
		return fmt.Errorf("failed to remove from hosts: %v", err)
	}
	
	// Unblock DNS queries
	if err := nb.unblockDNS(domain); err != nil {
		return fmt.Errorf("failed to unblock DNS: %v", err)
	}
	
	return nil
}

func (nb *NetworkBlocker) IsBlocked(domain string) bool {
	return nb.blockedDomains[domain]
}

func (nb *NetworkBlocker) blockDNS(domain string) error {
	// Block DNS queries using iptables
	cmd := exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Also block TCP DNS
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	return cmd.Run()
}

func (nb *NetworkBlocker) unblockDNS(domain string) error {
	// Remove DNS blocking rules
	cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run() // Ignore errors if rule doesn't exist
	
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run() // Ignore errors if rule doesn't exist
	
	return nil
}

func (nb *NetworkBlocker) addToHosts(domain string) error {
	// Read current hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	// Check if domain already blocked
	hostsContent := string(content)
	blockEntry := fmt.Sprintf("127.0.0.1 %s", domain)
	if strings.Contains(hostsContent, blockEntry) {
		return nil // Already blocked
	}
	
	// Add blocking entries
	newContent := hostsContent + fmt.Sprintf("\n# Keyphy block\n127.0.0.1 %s\n127.0.0.1 www.%s\n0.0.0.0 %s\n0.0.0.0 www.%s\n", domain, domain, domain, domain)
	
	return os.WriteFile(hostsFile, []byte(newContent), 0644)
}

func (nb *NetworkBlocker) removeFromHosts(domain string) error {
	// Read current hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	// Remove lines containing the domain
	lines := strings.Split(string(content), "\n")
	var newLines []string
	
	for _, line := range lines {
		if !strings.Contains(line, domain) {
			newLines = append(newLines, line)
		}
	}
	
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(hostsFile, []byte(newContent), 0644)
}

func (nb *NetworkBlocker) MonitorNetworkTraffic() error {
	// Monitor network connections using netstat or ss
	cmd := exec.Command("ss", "-tuln")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Parse connections and check against blocked domains
		if nb.shouldBlockConnection(line) {
			// Block the connection
			nb.blockConnection(line)
		}
	}
	
	return nil
}

func (nb *NetworkBlocker) shouldBlockConnection(connection string) bool {
	for domain := range nb.blockedDomains {
		if strings.Contains(connection, domain) {
			return true
		}
	}
	return false
}

func (nb *NetworkBlocker) blockConnection(connection string) error {
	// Extract connection details and block using iptables
	// This is a simplified implementation
	return nil
}