package blocker

import (
	"fmt"
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
	// Use DNS blocking (primary method)
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.BlockDomain(domain)
		return nil
	}
	// No fallback - DNS system is required for domain blocking
	return fmt.Errorf("DNS system not available for domain blocking")
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	// Use DNS unblocking (primary method)
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.UnblockDomain(domain)
		return nil
	}
	// No fallback - DNS system handles all domain blocking
	return nil
}

func (nb *NetworkBlocker) BlockIP(ip string) error {
	return nb.manager.BlockIP(ip)
}

func (nb *NetworkBlocker) UnblockIP(ip string) error {
	return nb.manager.UnblockIP(ip)
}

func (nb *NetworkBlocker) UnblockAll() error {
	// Unblock all DNS domains
	if nb.usingDNS && nb.dnsManager != nil {
		nb.dnsManager.UnblockAll()
	}
	
	// Unblock all IPs from platform manager
	return nb.manager.UnblockAllIPs()
}

// DNS system handles all domain blocking - no platform manager methods needed

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

// Network monitoring handled by DNS system