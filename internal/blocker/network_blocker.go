package blocker

import (
	"github.com/gajzzs/keyphy/internal/dns"
	"github.com/gajzzs/keyphy/internal/platform"
)

// check if keyphy dameom is run by manually or using default service provider

type NetworkBlocker struct {
	manager    platform.NetworkManager
	dnsManager *dns.DNSManager
	usingDNS   bool
}

func NewNetworkBlocker() *NetworkBlocker {
	return &NetworkBlocker{
		manager:    platform.NewNetworkManager(),
		dnsManager: nil, // Only initialize in daemon
		usingDNS:   false,
	}
}

// StartDNSSystem initializes and starts DNS system (daemon only)
func (nb *NetworkBlocker) StartDNSSystem() error {
	if nb.dnsManager != nil {
		return nil // Already started
	}
	
	nb.dnsManager = dns.NewDNSManager()
	if err := nb.dnsManager.Start(); err != nil {
		return err
	}
	
	nb.usingDNS = true
	return nil
}

func (nb *NetworkBlocker) BlockWebsite(domain string) error {
	// Use DNS blocking if available (more secure)
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.BlockDomain(domain)
		return nil
	}
	
	// Fallback to platform manager (hosts file + iptables/pfctl)
	return nb.manager.BlockDomain(domain)
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	// Use DNS unblocking if available
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.UnblockDomain(domain)
		return nil
	}
	
	// Fallback to platform manager
	return nb.manager.UnblockDomain(domain)
}

func (nb *NetworkBlocker) BlockIP(ip string) error {
	return nb.manager.BlockIP(ip)
}

func (nb *NetworkBlocker) UnblockIP(ip string) error {
	return nb.manager.UnblockIP(ip)
}

func (nb *NetworkBlocker) UnblockAll() error {
	// Unblock all DNS domains if using DNS
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.UnblockAll()
	}
	
	// Also unblock platform manager
	return nb.manager.UnblockAll()
}

func (nb *NetworkBlocker) ProtectHostsFile() error {
	return nb.manager.ProtectHostsFile()
}

func (nb *NetworkBlocker) UnprotectHostsFile() error {
	return nb.manager.UnprotectHostsFile()
}

// Stop DNS system when network blocker is stopped
func (nb *NetworkBlocker) Stop() error {
	if nb.usingDNS && nb.dnsManager != nil {
		return nb.dnsManager.Stop()
	}
	return nil
}

// Check if using DNS blocking
func (nb *NetworkBlocker) IsUsingDNS() bool {
	return nb.usingDNS
}

// Legacy compatibility methods
func (nb *NetworkBlocker) MonitorNetworkTraffic() error {
	// Network monitoring handled by DNS system or platform-specific implementation
	return nil
}

func (nb *NetworkBlocker) VerifyHostsFile() error {
	// Hosts file verification handled by DNS system or platform-specific implementation
	return nil
}