package platform

// NetworkManager provides cross-platform IP blocking only
// Domain blocking is handled by DNS system
type NetworkManager interface {
	BlockIP(ip string) error
	UnblockIP(ip string) error
	UnblockAllIPs() error
}

// NewNetworkManager creates a platform-specific network manager
func NewNetworkManager() NetworkManager {
	return newNetworkManager()
}