package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gajzzs/keyphy/internal/dns"
)

func main() {
	fmt.Println("Testing Keyphy DNS Server")
	
	// Create DNS server
	server := dns.NewDNSServer()
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start DNS server: %v", err)
	}
	defer server.Stop()
	
	fmt.Println("DNS server started (will try ports 53, 5353, 6666)")
	
	// Add some blocked domains
	server.BlockDomain("youtube.com")
	server.BlockDomain("facebook.com")
	server.BlockDomain("twitter.com")
	
	fmt.Println("Blocked domains: youtube.com, facebook.com, twitter.com")
	fmt.Println("DNS server is running. Test with:")
	fmt.Println("Test with (check logs for actual port):")
	fmt.Println("  dig @127.0.0.1 -p <PORT> youtube.com")
	fmt.Println("  dig @127.0.0.1 -p <PORT> google.com")
	fmt.Println("Press Ctrl+C to stop")
	
	// Keep running
	for {
		time.Sleep(1 * time.Second)
	}
}