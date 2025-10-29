package blocker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type NetworkBlocker struct {
	blockedDomains map[string]bool
	ipBlocksAdded  bool
	dohBlocksAdded bool
}

func NewNetworkBlocker() *NetworkBlocker {
	return &NetworkBlocker{
		blockedDomains: make(map[string]bool),
	}
}

func (nb *NetworkBlocker) BlockWebsite(domain string) error {
	nb.blockedDomains[domain] = true
	
	// Block both domain and www subdomain
	domains := []string{domain}
	if !strings.HasPrefix(domain, "www.") {
		domains = append(domains, "www."+domain)
	}
	
	for _, d := range domains {
		// Add to /etc/hosts
		fmt.Printf("Adding %s to hosts file...\n", d)
		if err := nb.addToHosts(d); err != nil {
			return fmt.Errorf("failed to add %s to hosts: %v", d, err)
		}
		
		// Block DNS queries
		fmt.Printf("Creating iptables rules for %s...\n", d)
		if err := nb.blockDNS(d); err != nil {
			return fmt.Errorf("failed to block DNS for %s: %v", d, err)
		}
	}
	
	fmt.Printf("Website blocking rules created successfully for %s\n", domain)
	return nil
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	delete(nb.blockedDomains, domain)
	
	// Remove from /etc/hosts
	fmt.Printf("Removing %s from hosts file...\n", domain)
	if err := nb.removeFromHosts(domain); err != nil {
		return fmt.Errorf("failed to remove from hosts: %v", err)
	}
	
	// Unblock DNS queries
	fmt.Printf("Removing iptables rules for %s...\n", domain)
	if err := nb.unblockDNS(domain); err != nil {
		return fmt.Errorf("failed to unblock DNS: %v", err)
	}
	
	fmt.Printf("Website unblocking completed for %s\n", domain)
	return nil
}

func (nb *NetworkBlocker) IsBlocked(domain string) bool {
	return nb.blockedDomains[domain]
}

func (nb *NetworkBlocker) blockDNS(domain string) error {
	// Check if rule already exists to prevent duplicates
	if nb.ruleExists(domain) {
		return nil
	}
	
	// Block DNS queries using iptables
	cmd := exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Also block TCP DNS
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Block HTTPS connections to domain
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "tcp", "--dport", "443", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Block HTTP connections to domain  
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "tcp", "--dport", "80", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Block systemd-resolved on port 53 (127.0.0.53)
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-d", "127.0.0.53", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-d", "127.0.0.53", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Block direct IP access to YouTube (Google IP ranges) - only once
	if (strings.Contains(domain, "youtube") || strings.Contains(domain, "google")) && !nb.ipBlocksAdded {
		youtubeIPs := []string{"142.250.0.0/15", "172.217.0.0/16", "216.58.192.0/19", "74.125.0.0/16"}
		for _, ip := range youtubeIPs {
			cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-d", ip, "-j", "DROP")
			cmd.Run() // Ignore errors
		}
		nb.ipBlocksAdded = true
	}
	
	// Block DNS-over-HTTPS servers - only once
	if !nb.dohBlocksAdded {
		dohServers := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9", "208.67.222.222"}
		for _, server := range dohServers {
			cmd = exec.Command("iptables", "-I", "OUTPUT", "1", "-p", "tcp", "-d", server, "--dport", "443", "-j", "DROP")
			cmd.Run() // Ignore errors
		}
		nb.dohBlocksAdded = true
	}
	
	return nil
}

func (nb *NetworkBlocker) unblockDNS(domain string) error {
	// Remove DNS blocking rules
	cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Remove HTTPS/HTTP blocking
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "443", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "80", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Remove systemd-resolved blocks
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-d", "127.0.0.53", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	cmd = exec.Command("iptables", "-D", "OUTPUT", "-d", "127.0.0.53", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	cmd.Run()
	
	// Remove IP blocks for YouTube
	if strings.Contains(domain, "youtube") || strings.Contains(domain, "google") {
		youtubeIPs := []string{"142.250.0.0/15", "172.217.0.0/16", "216.58.192.0/19", "74.125.0.0/16"}
		for _, ip := range youtubeIPs {
			cmd = exec.Command("iptables", "-D", "OUTPUT", "-d", ip, "-j", "DROP")
			cmd.Run()
		}
	}
	
	return nil
}

func (nb *NetworkBlocker) addToHosts(domain string) error {
	// Temporarily remove immutable flag
	nb.UnprotectHostsFile()
	defer nb.ProtectHostsFile()
	
	// Read current hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	// Check if domain already blocked (check for exact keyphy block)
	hostsContent := string(content)
	keyphyBlock := fmt.Sprintf("# Keyphy block %s", domain)
	if strings.Contains(hostsContent, keyphyBlock) {
		return nil // Already blocked
	}
	
	// Add blocking entries with unique marker
	newContent := hostsContent + fmt.Sprintf("\n# Keyphy block %s\n127.0.0.1 %s\n127.0.0.1 www.%s\n0.0.0.0 %s\n0.0.0.0 www.%s\n", domain, domain, domain, domain, domain)
	
	return os.WriteFile(hostsFile, []byte(newContent), 0600)
}

func (nb *NetworkBlocker) removeFromHosts(domain string) error {
	// Temporarily remove immutable flag
	nb.UnprotectHostsFile()
	defer nb.ProtectHostsFile()
	
	// Read current hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	// Remove keyphy block section for this domain
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inKeyphyBlock := false
	keyphyBlockMarker := fmt.Sprintf("# Keyphy block %s", domain)
	
	for _, line := range lines {
		if line == keyphyBlockMarker {
			inKeyphyBlock = true
			continue
		}
		if inKeyphyBlock && (strings.Contains(line, domain) || strings.HasPrefix(line, "127.0.0.1") || strings.HasPrefix(line, "0.0.0.0")) {
			continue // Skip keyphy block lines
		}
		if inKeyphyBlock && strings.HasPrefix(line, "#") {
			inKeyphyBlock = false // End of this block
		}
		newLines = append(newLines, line)
	}
	
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(hostsFile, []byte(newContent), 0600)
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

func (nb *NetworkBlocker) VerifyHostsFile() error {
	// Check if blocked domains are still in hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	hostsContent := string(content)
	modified := false
	
	// Check each blocked domain
	for domain := range nb.blockedDomains {
		blockEntry := fmt.Sprintf("127.0.0.1 %s", domain)
		if !strings.Contains(hostsContent, blockEntry) {
			// Domain block was removed, restore it
			if err := nb.addToHosts(domain); err != nil {
				return fmt.Errorf("failed to restore hosts entry for %s: %v", domain, err)
			}
			modified = true
		}
	}
	
	if modified {
		// Also restore DNS blocks
		for domain := range nb.blockedDomains {
			nb.blockDNS(domain)
		}
	}
	
	return nil
}

func (nb *NetworkBlocker) ProtectHostsFile() error {
	// Make hosts file immutable to prevent tampering
	cmd := exec.Command("chattr", "+i", "/etc/hosts")
	return cmd.Run()
}

func (nb *NetworkBlocker) UnprotectHostsFile() error {
	// Remove immutable flag from hosts file
	cmd := exec.Command("chattr", "-i", "/etc/hosts")
	return cmd.Run()
}

func (nb *NetworkBlocker) ruleExists(domain string) bool {
	// Check if iptables rule already exists for this domain
	cmd := exec.Command("iptables", "-C", "OUTPUT", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
	return cmd.Run() == nil
}

func (nb *NetworkBlocker) UnblockAll() error {
	// Remove all keyphy-related iptables rules
	fmt.Println("Removing all iptables rules...")
	nb.removeAllIptablesRules()
	
	// Clear blocked domains map
	nb.blockedDomains = make(map[string]bool)
	nb.ipBlocksAdded = false
	nb.dohBlocksAdded = false
	
	// Clean hosts file
	fmt.Println("Cleaning hosts file...")
	nb.UnprotectHostsFile()
	defer nb.ProtectHostsFile()
	
	// Remove all keyphy entries from hosts file
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inKeyphyBlock := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "# Keyphy block") {
			inKeyphyBlock = true
			continue
		}
		if inKeyphyBlock && (strings.HasPrefix(line, "127.0.0.1") || strings.HasPrefix(line, "0.0.0.0")) {
			continue
		}
		if inKeyphyBlock && line == "" {
			inKeyphyBlock = false
		}
		newLines = append(newLines, line)
	}
	
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(hostsFile, []byte(newContent), 0600); err != nil {
		return err
	}
	fmt.Println("All network blocking rules removed successfully")
	return nil
}

func (nb *NetworkBlocker) removeAllIptablesRules() {
	// Remove all Google IP blocks
	youtubeIPs := []string{"142.250.0.0/15", "172.217.0.0/16", "216.58.192.0/19", "74.125.0.0/16"}
	for _, ip := range youtubeIPs {
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-d", ip, "-j", "DROP")
			if cmd.Run() != nil {
				break // No more rules to delete
			}
		}
	}
	
	// Remove all DoH server blocks
	dohServers := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9", "208.67.222.222"}
	for _, server := range dohServers {
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "-d", server, "--dport", "443", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
	}
	
	// Remove all string matching rules (DNS, HTTP, HTTPS)
	domains := []string{"youtube.com", "www.youtube.com"}
	for _, domain := range domains {
		// Remove DNS rules
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
		// Remove HTTP/HTTPS rules
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "80", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-p", "tcp", "--dport", "443", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
		// Remove systemd-resolved rules
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-d", "127.0.0.53", "-p", "udp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
		for {
			cmd := exec.Command("iptables", "-D", "OUTPUT", "-d", "127.0.0.53", "-p", "tcp", "--dport", "53", "-m", "string", "--string", domain, "--algo", "bm", "-j", "DROP")
			if cmd.Run() != nil {
				break
			}
		}
	}
}