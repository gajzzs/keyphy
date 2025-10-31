//go:build linux
// +build linux

package platform

import (
	"fmt"
	"net"
	
	"syscall"

	"github.com/vishvananda/netlink"
)

type linuxNetworkManager struct {
	blockedIPs map[string]bool
}

func newNetworkManager() NetworkManager {
	return &linuxNetworkManager{
		blockedIPs: make(map[string]bool),
	}
}



func (nm *linuxNetworkManager) BlockIP(ip string) error {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	nm.blockedIPs[ip] = true
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
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	delete(nm.blockedIPs, ip)
	route := &netlink.Route{
		Dst: &net.IPNet{
			IP:   ipAddr,
			Mask: net.CIDRMask(32, 32),
		},
		Type: syscall.RTN_BLACKHOLE,
	}
	
	return netlink.RouteDel(route)
}

// DNS blocking handled by DNS system



func (nm *linuxNetworkManager) UnblockAllIPs() error {
	// Remove all blocked IP routes
	for ip := range nm.blockedIPs {
		nm.UnblockIP(ip)
	}
	nm.blockedIPs = make(map[string]bool)
	return nil
}
