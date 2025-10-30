//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type macNetworkManager struct {
	blockedDomains map[string]bool
}

func newNetworkManager() NetworkManager {
	return &macNetworkManager{
		blockedDomains: make(map[string]bool),
	}
}

func (nm *macNetworkManager) BlockDomain(domain string) error {
	nm.blockedDomains[domain] = true
	
	// Add to hosts file (same as Linux)
	if err := nm.addToHosts(domain); err != nil {
		return err
	}
	
	// Use pfctl for network blocking
	return nm.blockWithPfctl(domain)
}

func (nm *macNetworkManager) UnblockDomain(domain string) error {
	delete(nm.blockedDomains, domain)
	
	// Remove from hosts file
	if err := nm.removeFromHosts(domain); err != nil {
		return err
	}
	
	// Remove pfctl rules
	return nm.unblockWithPfctl(domain)
}

func (nm *macNetworkManager) BlockIP(ip string) error {
	// Create pfctl rule to block IP
	rule := fmt.Sprintf("block out to %s", ip)
	cmd := exec.Command("pfctl", "-a", "keyphy", "-f", "-")
	cmd.Stdin = strings.NewReader(rule)
	return cmd.Run()
}

func (nm *macNetworkManager) UnblockIP(ip string) error {
	// Remove pfctl rule
	cmd := exec.Command("pfctl", "-a", "keyphy", "-F", "rules")
	return cmd.Run()
}

func (nm *macNetworkManager) blockWithPfctl(domain string) error {
	// Create pfctl rules for both IPv4 and IPv6
	rules := `
block out inet proto tcp to any port 80
block out inet proto tcp to any port 443
block out inet proto udp to any port 53
block out inet6 proto tcp to any port 80
block out inet6 proto tcp to any port 443
block out inet6 proto udp to any port 53
`
	
	cmd := exec.Command("pfctl", "-a", "keyphy", "-f", "-")
	cmd.Stdin = strings.NewReader(rules)
	return cmd.Run()
}

func (nm *macNetworkManager) unblockWithPfctl(domain string) error {
	// Flush keyphy anchor rules
	cmd := exec.Command("pfctl", "-a", "keyphy", "-F", "rules")
	return cmd.Run()
}

func (nm *macNetworkManager) addToHosts(domain string) error {
	nm.UnprotectHostsFile()
	defer nm.ProtectHostsFile()
	
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	hostsContent := string(content)
	keyphyBlock := fmt.Sprintf("# Keyphy block %s", domain)
	if strings.Contains(hostsContent, keyphyBlock) {
		return nil
	}
	
	newContent := hostsContent + fmt.Sprintf("\n# Keyphy block %s\n127.0.0.1 %s\n127.0.0.1 www.%s\n0.0.0.0 %s\n0.0.0.0 www.%s\n::1 %s\n::1 www.%s\n", domain, domain, domain, domain, domain, domain, domain)
	
	return os.WriteFile(hostsFile, []byte(newContent), 0600)
}

func (nm *macNetworkManager) removeFromHosts(domain string) error {
	nm.UnprotectHostsFile()
	defer nm.ProtectHostsFile()
	
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inKeyphyBlock := false
	keyphyBlockMarker := fmt.Sprintf("# Keyphy block %s", domain)
	
	for _, line := range lines {
		if line == keyphyBlockMarker {
			inKeyphyBlock = true
			continue
		}
		if inKeyphyBlock && (strings.Contains(line, domain) || strings.HasPrefix(line, "127.0.0.1") || strings.HasPrefix(line, "0.0.0.0") || strings.HasPrefix(line, "::1")) {
			continue
		}
		if inKeyphyBlock && strings.HasPrefix(line, "#") {
			inKeyphyBlock = false
		}
		newLines = append(newLines, line)
	}
	
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(hostsFile, []byte(newContent), 0600)
}

func (nm *macNetworkManager) UnblockAll() error {
	// Flush all pfctl rules
	cmd := exec.Command("pfctl", "-a", "keyphy", "-F", "all")
	cmd.Run()
	
	// Remove from hosts file
	for domain := range nm.blockedDomains {
		nm.removeFromHosts(domain)
	}
	
	nm.blockedDomains = make(map[string]bool)
	return nil
}

func (nm *macNetworkManager) ProtectHostsFile() error {
	// Use chflags instead of chattr on macOS
	cmd := exec.Command("chflags", "uchg", "/etc/hosts")
	return cmd.Run()
}

func (nm *macNetworkManager) UnprotectHostsFile() error {
	// Remove immutable flag on macOS
	cmd := exec.Command("chflags", "nouchg", "/etc/hosts")
	return cmd.Run()
}