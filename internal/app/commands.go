package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"keyphy/internal/config"
	"keyphy/internal/crypto"
	"keyphy/internal/device"
	"keyphy/internal/service"
)

func NewBlockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Block apps, websites, or file paths",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "app [application-name]",
			Short: "Block an application",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return config.AddBlockedApp(args[0])
			},
		},
		&cobra.Command{
			Use:   "website [domain]",
			Short: "Block a website",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return config.AddBlockedWebsite(args[0])
			},
		},
		&cobra.Command{
			Use:   "path [file-or-folder-path]",
			Short: "Block file or folder access",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return config.AddBlockedPath(args[0])
			},
		},
	)

	return cmd
}

func NewUnblockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unblock [item]",
		Short: "Remove blocking rule for app, website, or path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.RemoveBlocked(args[0])
		},
	}
}

func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all blocked items",
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
			fmt.Printf("Service Enabled: %t\n", cfg.ServiceEnabled)
			
			return nil
		},
	}
}

func NewDeviceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Manage authentication devices",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List available USB devices",
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
			RunE: func(cmd *cobra.Command, args []string) error {
				devices, err := device.ListUSBDevices()
				if err != nil {
					return err
				}
				
				for _, dev := range devices {
					if dev.UUID == args[0] {
						cfg := config.GetConfig()
						cfg.AuthDevice = dev.UUID
						cfg.AuthKey = crypto.GenerateDeviceKey(dev.UUID, dev.Name)
						fmt.Printf("Selected device: %s\n", dev.Name)
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
	}

	daemon := service.NewDaemon()

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start",
			Short: "Start keyphy daemon",
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("daemon must be run as root")
				}
				return daemon.Start()
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop keyphy daemon",
			RunE: func(cmd *cobra.Command, args []string) error {
				return daemon.Stop()
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Check daemon status",
			RunE: func(cmd *cobra.Command, args []string) error {
				running, err := service.GetDaemonStatus()
				if err != nil {
					return err
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