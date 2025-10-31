package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gajzzs/keyphy/internal/dns"
)

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("This program must be run as root to manage system DNS")
	}

	fmt.Println("Testing Keyphy DNS System with Monitoring")
	
	// Create DNS manager
	dnsManager := dns.NewDNSManager()
	
	// Start DNS system (server + hijacking + monitoring)
	fmt.Println("Starting DNS system...")
	if err := dnsManager.Start(); err != nil {
		log.Fatalf("Failed to start DNS system: %v", err)
	}
	
	// Add blocked domains
	dnsManager.BlockDomain("youtube.com")
	dnsManager.BlockDomain("facebook.com")
	dnsManager.BlockDomain("twitter.com")
	
	fmt.Println("DNS system started with monitoring!")
	fmt.Println("Blocked domains: youtube.com, facebook.com, twitter.com")
	fmt.Println("")
	fmt.Println("Test the system:")
	fmt.Println("1. Try: nslookup youtube.com")
	fmt.Println("2. Try: nslookup google.com")
	fmt.Println("3. Try changing system DNS and watch it get reverted")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop")
	
	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for signal
	<-sigChan
	
	fmt.Println("\nShutting down DNS system...")
	if err := dnsManager.Stop(); err != nil {
		log.Printf("Error stopping DNS system: %v", err)
	}
	
	fmt.Println("DNS system stopped and original DNS restored")
}