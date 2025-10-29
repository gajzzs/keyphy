package app

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"keyphy/internal/blocker"
	"keyphy/internal/config"
	"keyphy/internal/crypto"
	"keyphy/internal/device"
	"keyphy/internal/service"
)

func NewBlockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Block apps, websites, or file paths",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "app [application-name]",
			Short: "Block an application",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if !validateDeviceAuth() {
					return fmt.Errorf("authentication device not connected or invalid")
				}
				fmt.Printf("Blocking application: %s\n", args[0])
				if err := config.AddBlockedApp(args[0]); err != nil {
					return err
				}
				fmt.Printf("Application '%s' blocked successfully\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "website [domain]",
			Short: "Block a website",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if !validateDeviceAuth() {
					return fmt.Errorf("authentication device not connected or invalid")
				}
				fmt.Printf("Blocking website: %s\n", args[0])
				if err := config.AddBlockedWebsite(args[0]); err != nil {
					return err
				}
				fmt.Printf("Website '%s' blocked successfully\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "path [file-or-folder-path]",
			Short: "Block file or folder access",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if !validateDeviceAuth() {
					return fmt.Errorf("authentication device not connected or invalid")
				}
				fmt.Printf("Blocking path: %s\n", args[0])
				if err := config.AddBlockedPath(args[0]); err != nil {
					return err
				}
				fmt.Printf("Path '%s' blocked successfully\n", args[0])
				return nil
			},
		},
	)

	return cmd
}

func NewUnblockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unblock [item]",
		Short: "Remove blocking rule for app, website, or path (use 'all' to remove everything)",
		Args:  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validateDeviceAuth() {
				return fmt.Errorf("authentication device not connected or invalid")
			}
			if args[0] == "all" {
				fmt.Println("Removing all blocking rules...")
				// Remove active iptables rules
				networkBlocker := blocker.NewNetworkBlocker()
				if err := networkBlocker.UnblockAll(); err != nil {
					fmt.Printf("Warning: Failed to remove iptables rules: %v\n", err)
				}
				// Clear config
				cfg := config.GetConfig()
				cfg.BlockedApps = []string{}
				cfg.BlockedWebsites = []string{}
				cfg.BlockedPaths = []string{}
				fmt.Println("All blocking rules removed successfully")
				return config.SaveConfig()
			}
			// Remove specific item from active rules and config
			fmt.Printf("Unblocking: %s\n", args[0])
			networkBlocker := blocker.NewNetworkBlocker()
			if err := networkBlocker.UnblockWebsite(args[0]); err != nil {
				fmt.Printf("Warning: Failed to remove iptables rules for %s: %v\n", args[0], err)
			}
			if err := config.RemoveBlocked(args[0]); err != nil {
				return err
			}
			fmt.Printf("'%s' unblocked successfully\n", args[0])
			return nil
		},
	}
}

func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all blocked items",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			
			fmt.Println("Blocked Applications:")
			for _, app := range cfg.BlockedApps {
				fmt.Printf("  - %s\n", app)
			}
			
			fmt.Println("\nBlocked Websites:")
			for _, website := range cfg.BlockedWebsites {
				fmt.Printf("  - %s\n", website)
			}
			
			fmt.Println("\nBlocked Paths:")
			for _, path := range cfg.BlockedPaths {
				fmt.Printf("  - %s\n", path)
			}
			
			fmt.Printf("\nAuth Device: %s\n", cfg.AuthDevice)
			if cfg.AuthKey != "" {
				fmt.Println("Auth Key: [CONFIGURED]")
			} else {
				fmt.Println("Auth Key: [NOT SET]")
			}
			
			// Show actual service status instead of config flag
			running, _ := service.GetDaemonStatus()
			serviceStatus := service.GetServiceStatus()
			if running {
				fmt.Println("Service Status: Running")
			} else {
				fmt.Println("Service Status: Stopped")
			}
			fmt.Printf("Systemd Status: %s", serviceStatus)
			
			return nil
		},
	}
}

func NewDeviceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Manage authentication devices",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List available USB devices",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				devices, err := device.ListUSBDevices()
				if err != nil {
					return err
				}
				
				fmt.Println("Available USB Devices:")
				for i, dev := range devices {
					fmt.Printf("%d. %s (UUID: %s)\n", i+1, dev.Name, dev.UUID)
					fmt.Printf("   Path: %s\n", dev.DevPath)
					fmt.Printf("   Mount: %s\n\n", dev.MountPoint)
				}
				
				return nil
			},
		},
		&cobra.Command{
			Use:   "select [device-uuid]",
			Short: "Select device for authentication",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Scanning for USB devices...")
				devices, err := device.ListUSBDevices()
				if err != nil {
					return err
				}
				
				for _, dev := range devices {
					if dev.UUID == args[0] {
						fmt.Printf("Found device: %s (UUID: %s)\n", dev.Name, dev.UUID)
						fmt.Println("Generating authentication key...")
						cfg := config.GetConfig()
						cfg.AuthDevice = dev.UUID
						cfg.AuthKey = crypto.GenerateDeviceKey(dev.UUID, dev.Name)
						fmt.Printf("Device '%s' selected as authentication device\n", dev.Name)
						return config.SaveConfig()
					}
				}
				return fmt.Errorf("device with UUID %s not found", args[0])
			},
		},
	)

	return cmd
}

func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage keyphy daemon service",
		DisableFlagsInUseLine: true,
	}

	daemon := service.NewDaemon()

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start",
			Short: "Start keyphy daemon",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("daemon must be run as root")
				}
				fmt.Println("Starting keyphy daemon...")
				if err := daemon.Start(); err != nil {
					return err
				}
				fmt.Println("Keyphy daemon started successfully")
				// Keep daemon running
				select {}
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop keyphy daemon",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("stop requires root privileges")
				}
				fmt.Println("Stopping keyphy daemon...")
				if err := service.SendStopSignal(); err != nil {
					return fmt.Errorf("failed to send stop signal: %v", err)
				}
				fmt.Println("Keyphy daemon stopped successfully")
				return nil
			},
		},

		&cobra.Command{
			Use:   "status",
			Short: "Check daemon status",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				running, err := service.GetDaemonStatus()
				if err != nil {
					return fmt.Errorf("failed to get daemon status: %v", err)
				}
				if running {
					fmt.Println("Daemon status: Running")
				} else {
					fmt.Println("Daemon status: Stopped")
				}
				fmt.Printf("Service status: %s", service.GetServiceStatus())
				return nil
			},
		},
	)

	return cmd
}

func NewLockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lock",
		Short: "Lock all blocks (requires auth device)",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("lock requires root privileges")
			}
			fmt.Println("Sending lock signal to daemon...")
			if err := service.SendLockSignal(); err != nil {
				return fmt.Errorf("failed to send lock signal: %v", err)
			}
			fmt.Println("Lock signal sent successfully - all blocks are now active")
			return nil
		},
	}
}

func NewUnlockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock",
		Short: "Unlock all blocks (requires auth device)",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("unlock requires root privileges")
			}
			fmt.Println("Sending unlock signal to daemon...")
			if err := service.SendUnlockSignal(); err != nil {
				return fmt.Errorf("failed to send unlock signal: %v", err)
			}
			fmt.Println("Unlock signal sent successfully - all blocks are now disabled")
			return nil
		},
	}
}

func validateDeviceAuth() bool {
	cfg := config.GetConfig()
	if cfg.AuthDevice == "" || cfg.AuthKey == "" {
		fmt.Println("No authentication device configured")
		return false
	}
	
	fmt.Println("Checking for authentication device...")
	devices, err := device.ListUSBDevices()
	if err != nil {
		fmt.Printf("Failed to scan USB devices: %v\n", err)
		return false
	}
	
	for _, dev := range devices {
		if dev.UUID == cfg.AuthDevice {
			fmt.Printf("Found authentication device: %s\n", dev.Name)
			fmt.Println("Validating device authentication...")
			valid, err := crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
			if err != nil {
				fmt.Printf("Authentication validation failed: %v\n", err)
				return false
			}
			if valid {
				fmt.Println("Device authentication successful")
			} else {
				fmt.Println("Device authentication failed")
			}
			return valid
		}
	}
	fmt.Println("Authentication device not connected")
	return false
}