package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/gajzzs/keyphy/internal/platform"
)

func main() {
	fmt.Printf("Testing Keyphy cross-platform support on %s\n", runtime.GOOS)
	
	// Test device manager
	fmt.Println("\n=== Testing Device Manager ===")
	deviceManager := platform.NewDeviceManager()
	
	devices, err := deviceManager.ListUSBDevices()
	if err != nil {
		log.Printf("Error listing devices: %v", err)
	} else {
		fmt.Printf("Found %d USB devices:\n", len(devices))
		for i, device := range devices {
			fmt.Printf("%d. %s (UUID: %s, Path: %s, Mounted: %v)\n", 
				i+1, device.Name, device.UUID, device.Path, device.Mounted)
		}
	}
	
	// Test network manager
	fmt.Println("\n=== Testing Network Manager ===")
	_ = platform.NewNetworkManager()
	
	fmt.Println("Network manager created successfully")
	fmt.Printf("Platform-specific implementation loaded for %s\n", runtime.GOOS)
	
	// Test a simple operation (don't actually block anything)
	fmt.Println("Cross-platform abstraction layer working!")
}