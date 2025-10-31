//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	
	"os/exec"
	"strings"
)

type macNetworkManager struct {
	blockedIPs map[string]bool
}

func newNetworkManager() NetworkManager {
	return &macNetworkManager{
		blockedIPs: make(map[string]bool),
	}
}



func (nm *macNetworkManager) BlockIP(ip string) error {
	nm.blockedIPs[ip] = true
	rule := fmt.Sprintf("block out to %s", ip)
	cmd := exec.Command("pfctl", "-a", "keyphy", "-f", "-")
	cmd.Stdin = strings.NewReader(rule)
	return cmd.Run()
}

func (nm *macNetworkManager) UnblockIP(ip string) error {
	delete(nm.blockedIPs, ip)
	cmd := exec.Command("pfctl", "-a", "keyphy", "-F", "rules")
	return cmd.Run()
}





func (nm *macNetworkManager) UnblockAllIPs() error {
	// Remove all blocked IPs
	for ip := range nm.blockedIPs {
		nm.UnblockIP(ip)
	}
	nm.blockedIPs = make(map[string]bool)
	return nil
}
