package platform

// NetworkManager provides cross-platform network blocking
type NetworkManager interface {
	BlockDomain(domain string) error
	UnblockDomain(domain string) error
	BlockIP(ip string) error
	UnblockIP(ip string) error
	UnblockAll() error
	ProtectHostsFile() error
	UnprotectHostsFile() error
}

// NewNetworkManager creates a platform-specific network manager
func NewNetworkManager() NetworkManager {
	return newNetworkManager()
}