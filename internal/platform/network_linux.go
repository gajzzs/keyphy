//go:build linux
// +build linux

package platform

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
)

type linuxNetworkManager struct {
	blockedDomains map[string]bool
}

func newNetworkManager() NetworkManager {
	return &linuxNetworkManager{
		blockedDomains: make(map[string]bool),
	}
}

func (nm *linuxNetworkManager) BlockDomain(domain string) error {
	nm.blockedDomains[domain] = true
	
	// Add to hosts file
	if err := nm.addToHosts(domain); err != nil {
		return err
	}
	
	// Add iptables rules
	return nm.blockDNS(domain)
}

func (nm *linuxNetworkManager) UnblockDomain(domain string) error {
	delete(nm.blockedDomains, domain)
	
	// Remove from hosts file
	if err := nm.removeFromHosts(domain); err != nil {
		return err
	}
	
	// Remove iptables rules
	return nm.unblockDNS(domain)
}

func (nm *linuxNetworkManager) BlockIP(ip string) error {
	// Parse IP address
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	// Create route to blackhole
	route := &netlink.Route{
		Dst: &net.IPNet{
			IP:   ipAddr,
			Mask: net.CIDRMask(32, 32),
		},
		Type: syscall.RTN_BLACKHOLE,
	}
	
	return netlink.RouteAdd(route)
}

func (nm *linuxNetworkManager) UnblockIP(ip string) error {
	// Parse IP address
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	// Remove blackhole route
	route := &netlink.Route{
		Dst: &net.IPNet{
			IP:   ipAddr,
			Mask: net.CIDRMask(32, 32),
		},
		Type: syscall.RTN_BLACKHOLE,
	}
	
	return netlink.RouteDel(route)
}

func (nm *linuxNetworkManager) blockDNS(domain string) error {
	// DNS blocking via hosts file is sufficient for most cases
	// For advanced blocking, we rely on hosts file modification
	// which is more reliable than iptables string matching
	return nil
}

func (nm *linuxNetworkManager) unblockDNS(domain string) error {
	// DNS unblocking handled by hosts file removal
	return nil
}

func (nm *linuxNetworkManager) addToHosts(domain string) error {
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
	
	newContent := hostsContent + fmt.Sprintf("\n# Keyphy block %s\n127.0.0.1 %s\n127.0.0.1 www.%s\n0.0.0.0 %s\n0.0.0.0 www.%s\n", domain, domain, domain, domain, domain)
	
	return os.WriteFile(hostsFile, []byte(newContent), 0600)
}

func (nm *linuxNetworkManager) removeFromHosts(domain string) error {
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
		if inKeyphyBlock && (strings.Contains(line, domain) || strings.HasPrefix(line, "127.0.0.1") || strings.HasPrefix(line, "0.0.0.0")) {
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

func (nm *linuxNetworkManager) UnblockAll() error {
	// Remove all iptables rules for blocked domains
	for domain := range nm.blockedDomains {
		nm.unblockDNS(domain)
		nm.removeFromHosts(domain)
	}
	
	nm.blockedDomains = make(map[string]bool)
	return nil
}

func (nm *linuxNetworkManager) ProtectHostsFile() error {
	// Use syscall to set immutable attribute
	file, err := os.Open("/etc/hosts")
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Set FS_IMMUTABLE_FL flag
	const FS_IOC_SETFLAGS = 0x40086602
	const FS_IMMUTABLE_FL = 0x00000010
	
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(FS_IOC_SETFLAGS),
		uintptr(FS_IMMUTABLE_FL))
	
	if errno != 0 {
		return errno
	}
	return nil
}

func (nm *linuxNetworkManager) UnprotectHostsFile() error {
	// Use syscall to remove immutable attribute
	file, err := os.Open("/etc/hosts")
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Remove FS_IMMUTABLE_FL flag
	const FS_IOC_SETFLAGS = 0x40086602
	
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(FS_IOC_SETFLAGS),
		uintptr(0x00000000))
	
	if errno != 0 {
		return errno
	}
	return nil
}