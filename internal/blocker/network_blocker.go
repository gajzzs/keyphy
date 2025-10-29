package blocker

import (
	"fmt"
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
	
	// Add iptables rule to block domain
	if err := nb.addIptablesRule(domain); err != nil {
		return fmt.Errorf("failed to add iptables rule: %v", err)
	}
	
	// Add to /etc/hosts as backup
	return nb.addToHosts(domain)
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	delete(nb.blockedDomains, domain)
	
	// Remove iptables rule
	if err := nb.removeIptablesRule(domain); err != nil {
		return fmt.Errorf("failed to remove iptables rule: %v", err)
	}
	
	// Remove from /etc/hosts
	return nb.removeFromHosts(domain)
}

func (nb *NetworkBlocker) IsBlocked(domain string) bool {
	return nb.blockedDomains[domain]
}

func (nb *NetworkBlocker) addIptablesRule(domain string) error {
	// Block outgoing connections to domain
	cmd := exec.Command("iptables", "-A", "OUTPUT", "-d", domain, "-j", "DROP")
	return cmd.Run()
}

func (nb *NetworkBlocker) removeIptablesRule(domain string) error {
	// Remove blocking rule
	cmd := exec.Command("iptables", "-D", "OUTPUT", "-d", domain, "-j", "DROP")
	return cmd.Run()
}

func (nb *NetworkBlocker) addToHosts(domain string) error {
	// Add domain to /etc/hosts pointing to localhost
	hostsEntry := fmt.Sprintf("127.0.0.1 %s\n", domain)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' >> /etc/hosts", hostsEntry))
	return cmd.Run()
}

func (nb *NetworkBlocker) removeFromHosts(domain string) error {
	// Remove domain from /etc/hosts
	cmd := exec.Command("sed", "-i", fmt.Sprintf("/%s/d", domain), "/etc/hosts")
	return cmd.Run()
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