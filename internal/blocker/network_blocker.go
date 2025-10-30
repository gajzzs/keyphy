package blocker

import (
	"github.com/gajzzs/keyphy/internal/platform"
)

type NetworkBlocker struct {
	manager platform.NetworkManager
}

func NewNetworkBlocker() *NetworkBlocker {
	return &NetworkBlocker{
		manager: platform.NewNetworkManager(),
	}
}

func (nb *NetworkBlocker) BlockWebsite(domain string) error {
	return nb.manager.BlockDomain(domain)
}

func (nb *NetworkBlocker) UnblockWebsite(domain string) error {
	return nb.manager.UnblockDomain(domain)
}

func (nb *NetworkBlocker) BlockIP(ip string) error {
	return nb.manager.BlockIP(ip)
}

func (nb *NetworkBlocker) UnblockIP(ip string) error {
	return nb.manager.UnblockIP(ip)
}

func (nb *NetworkBlocker) UnblockAll() error {
	return nb.manager.UnblockAll()
}

func (nb *NetworkBlocker) ProtectHostsFile() error {
	return nb.manager.ProtectHostsFile()
}

func (nb *NetworkBlocker) UnprotectHostsFile() error {
	return nb.manager.UnprotectHostsFile()
}

// Legacy compatibility methods
func (nb *NetworkBlocker) MonitorNetworkTraffic() error {
	// Network monitoring handled by platform-specific implementation
	return nil
}

func (nb *NetworkBlocker) VerifyHostsFile() error {
	// Hosts file verification handled by platform-specific implementation
	return nil
}