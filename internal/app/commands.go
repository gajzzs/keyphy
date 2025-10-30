package app

import (
	"fmt"
	"os"
	"strings"
	"os/exec"
	"github.com/spf13/cobra"
	"github.com/manifoldco/promptui"
	"github.com/gajzzs/keyphy/internal/auth"
	"github.com/gajzzs/keyphy/internal/blocker"
	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/crypto"
	"github.com/gajzzs/keyphy/internal/device"
	"github.com/gajzzs/keyphy/internal/service"
)

func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add apps, websites, or file paths to blocking list",
		DisableFlagsInUseLine: true,
	}

	appCmd := &cobra.Command{
		Use:   "app [application-name]",
		Short: "Add an application to blocking list (use executable name, e.g. firefox, chrome, code)",
		Args:  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validateDeviceAuth() {
				return fmt.Errorf("authentication device not connected or invalid")
			}
			
			customPath, _ := cmd.Flags().GetString("path")
			appName := args[0]
			
			fmt.Printf("Adding application to blocking list: %s\n", appName)
			if customPath != "" {
				fmt.Printf("Using custom path: %s\n", customPath)
				if err := config.AddBlockedAppWithPath(appName, customPath); err != nil {
					return err
				}
			} else {
				if err := config.AddBlockedApp(appName); err != nil {
					return err
				}
			}
			fmt.Printf("Application '%s' added to blocking list successfully\n", appName)
			return nil
		},
	}
	appCmd.Flags().String("path", "", "Custom path to executable (e.g. /opt/app/bin/myapp)")

	cmd.AddCommand(
		appCmd,
		&cobra.Command{
			Use:   "website [domain]",
			Short: "Add a website to blocking list (enter domain without www, e.g. youtube.com)",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if !validateDeviceAuth() {
					return fmt.Errorf("authentication device not connected or invalid")
				}
				fmt.Printf("Adding website to blocking list: %s\n", args[0])
				if err := config.AddBlockedWebsite(args[0]); err != nil {
					return err
				}
				fmt.Printf("Website '%s' added to blocking list successfully\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "path [file-or-folder-path]",
			Short: "Add file or folder to blocking list",
			Args:  cobra.ExactArgs(1),
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if !validateDeviceAuth() {
					return fmt.Errorf("authentication device not connected or invalid")
				}
				fmt.Printf("Adding path to blocking list: %s\n", args[0])
				if err := config.AddBlockedPath(args[0]); err != nil {
					return err
				}
				fmt.Printf("Path '%s' added to blocking list successfully\n", args[0])
				return nil
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
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validateDeviceAuth() {
				return fmt.Errorf("authentication device not connected or invalid")
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

func NewResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset keyphy - remove all blocks, restore system, and stop service",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validateDeviceAuth() {
				return fmt.Errorf("authentication device not connected or invalid")
			}
			
			if os.Geteuid() != 0 {
				return execWithSudo("reset")
			}
			
			return performReset()
		},
	}
	
	// Add hidden emergency reset command
	emergencyCmd := &cobra.Command{
		Use:    "emergency",
		Short:  "Emergency reset without device authentication",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return execWithSudo("reset", "emergency")
			}
			
			fmt.Println("WARNING: Emergency reset bypasses device authentication!")
			return performReset()
		},
	}
	
	cmd.AddCommand(emergencyCmd)
	return cmd
}

func performReset() error {
			
			fmt.Println("Resetting keyphy system...")
			
			// Stop service properly
			fmt.Println("Stopping service...")
			sm, err := service.NewServiceManager()
			if err == nil {
				if err := sm.Stop(); err != nil {
					fmt.Printf("Warning: Failed to stop service: %v\n", err)
				} else {
					fmt.Println("Service stopped successfully")
				}
			}
			
			// Remove all blocking rules
			fmt.Println("Removing all blocking rules...")
			networkBlocker := blocker.NewNetworkBlocker()
			if err := networkBlocker.UnblockAll(); err != nil {
				fmt.Printf("Warning: Failed to remove network rules: %v\n", err)
			}
			
			// Restore all app executables
			appBlocker := blocker.NewAppBlocker()
			cfg := config.GetConfig()
			for _, app := range cfg.BlockedApps {
				if err := appBlocker.UnblockApp(app); err != nil {
					fmt.Printf("Warning: Failed to restore %s: %v\n", app, err)
				}
			}
			
			// Restore file permissions
			fileBlocker := blocker.NewFileBlocker()
			for _, path := range cfg.BlockedPaths {
				if err := fileBlocker.UnblockPath(path); err != nil {
					fmt.Printf("Warning: Failed to restore %s: %v\n", path, err)
				}
			}
			
			// Clear all configuration including auth device
			fmt.Println("Clearing configuration...")
			cfg.BlockedApps = []string{}
			cfg.BlockedWebsites = []string{}
			cfg.BlockedPaths = []string{}
			cfg.AuthDevice = ""
			cfg.AuthDeviceName = ""
			cfg.AuthKey = ""
			cfg.AuthMountState = ""
			cfg.EnforceState = false
			if err := config.SaveConfig(); err != nil {
				fmt.Printf("Warning: Failed to clear config: %v\n", err)
			} else {
				fmt.Println("Configuration cleared successfully")
			}
			
	fmt.Println("Keyphy system reset complete - all blocks removed, auth cleared, and service stopped")
	return nil
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
			if cfg.AuthDeviceName != "" {
				fmt.Printf("Auth Device Name: %s\n", cfg.AuthDeviceName)
			}
			if cfg.AuthMountState != "" {
				fmt.Printf("Required Mount State: %s\n", cfg.AuthMountState)
			}
			fmt.Printf("State Enforcement: %t\n", cfg.EnforceState)
			if cfg.AuthKey != "" {
				fmt.Println("Auth Key: [CONFIGURED]")
			} else {
				fmt.Println("Auth Key: [NOT SET]")
			}
			
			// Show cross-platform service status
			if sm, err := service.NewServiceManager(); err == nil {
				if status, err := sm.Status(); err == nil {
					fmt.Printf("Service Status: %s\n", status)
				} else {
					fmt.Println("Service Status: Unknown")
				}
				fmt.Printf("Service Config: %s\n", service.GetServiceConfigPath())
			} else {
				fmt.Println("Service Status: Not Available")
			}
			
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

	listCmd := &cobra.Command{
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
	}
	
	selectCmd := &cobra.Command{
		Use:   "select [device-uuid]",
		Short: "Select device for authentication",
		Args:  cobra.MaximumNArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Scanning for USB devices...")
			devices, err := device.ListUSBDevices()
			if err != nil {
				return err
			}
			
			enforceState, _ := cmd.Flags().GetBool("save-state")
			
			var selectedDevice device.Device
			
			if len(args) == 0 {
				prompt := promptui.Select{
					Label: "Select USB Device",
					Items: devices,
					Templates: &promptui.SelectTemplates{
						Active:   "▶ {{ .Name | cyan }} ({{ .UUID | faint }})",
						Inactive: "  {{ .Name | cyan }} ({{ .UUID | faint }})",
						Selected: "✔ {{ .Name | red | cyan }}",
					},
				}
				i, _, err := prompt.Run()
				if err != nil {
					return err
				}
				selectedDevice = devices[i]
			} else {
				for _, dev := range devices {
					if dev.UUID == args[0] {
						selectedDevice = dev
						break
					}
				}
				if selectedDevice.UUID == "" {
					return fmt.Errorf("device with UUID %s not found", args[0])
				}
			}
			
			dev := selectedDevice
			fmt.Printf("Found device: %s (UUID: %s)\n", dev.Name, dev.UUID)
			
			// Determine mount state
			mountState := "unmounted"
			if dev.MountPoint != "(not mounted)" {
				if strings.Contains(dev.MountPoint, "encrypted") {
					mountState = "mounted-encrypted"
				} else {
					mountState = "mounted"
				}
			}
			
			fmt.Printf("Current state: %s, UUID: %s, Name: %s\n", mountState, dev.UUID, dev.Name)
			
			if enforceState {
				fmt.Printf("State enforcement enabled - device must be %s for authentication\n", mountState)
			} else {
				fmt.Println("State enforcement disabled - device works in any mount state")
			}
			
			fmt.Println("Generating authentication key...")
			cfg := config.GetConfig()
			cfg.AuthDevice = dev.UUID
			cfg.AuthDeviceName = dev.Name
			cfg.AuthMountState = mountState
			cfg.EnforceState = enforceState
			cfg.AuthKey = crypto.GenerateDeviceKey(dev.UUID, dev.Name)
			
			fmt.Printf("Device '%s' selected as authentication device\n", dev.Name)
			return config.SaveConfig()
		},
	}
	
	selectCmd.Flags().Bool("save-state", false, "Enforce exact mount state for authentication")
	
	cmd.AddCommand(
		listCmd,
		selectCmd,
	)

	return cmd
}

func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage keyphy daemon service",
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install keyphy as system service",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("service installation requires root privileges")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				if err := sm.Install(); err != nil {
					return fmt.Errorf("failed to install service: %v", err)
				}
				fmt.Printf("Service installed successfully at: %s\n", service.GetServiceConfigPath())
				return nil
			},
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall keyphy system service",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("service uninstallation requires root privileges")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				if err := sm.Uninstall(); err != nil {
					return fmt.Errorf("failed to uninstall service: %v", err)
				}
				fmt.Println("Service uninstalled successfully")
				return nil
			},
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start keyphy daemon service",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("service management requires root privileges")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				if err := sm.Start(); err != nil {
					return fmt.Errorf("failed to start service: %v", err)
				}
				fmt.Println("Service started successfully")
				return nil
			},
		},
		&cobra.Command{
			Use:    "run-daemon",
			Short:  "Internal command to run daemon (do not use directly)",
			Hidden: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("daemon must be run as root")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				return sm.Run()
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop keyphy daemon service",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("service management requires root privileges")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				if err := sm.Stop(); err != nil {
					return fmt.Errorf("failed to stop service: %v", err)
				}
				fmt.Println("Service stopped successfully")
				return nil
			},
		},
		&cobra.Command{
			Use:   "restart",
			Short: "Restart keyphy daemon service",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return fmt.Errorf("service management requires root privileges")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				if err := sm.Restart(); err != nil {
					return fmt.Errorf("failed to restart service: %v", err)
				}
				fmt.Println("Service restarted successfully")
				return nil
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Check service status",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				status, err := sm.Status()
				if err != nil {
					return fmt.Errorf("failed to get service status: %v", err)
				}
				fmt.Printf("Service Status: %s\n", status)
				fmt.Printf("Config Path: %s\n", service.GetServiceConfigPath())
				return nil
			},
		},
		&cobra.Command{
			Use:   "watchdog",
			Short: "Start service watchdog to auto-restart if killed",
			DisableFlagsInUseLine: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if os.Geteuid() != 0 {
					return execWithSudo("service", "watchdog")
				}
				sm, err := service.NewServiceManager()
				if err != nil {
					return err
				}
				return sm.Watchdog()
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
				return execWithSudo("lock")
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
				return execWithSudo("unlock")
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

func execWithSudo(args ...string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	
	cmdArgs := append([]string{execPath}, args...)
	cmd := exec.Command("sudo", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func validateDeviceAuth() bool {
	auth := auth.NewPhysicalAuth()
	
	fmt.Println("Performing enhanced device authentication...")
	valid, err := auth.AuthenticateDevice()
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		return false
	}
	
	if valid {
		fmt.Println("Physical device authentication successful")
		status := auth.GetAuthStatus()
		if status["tamper_detected"].(bool) {
			fmt.Println("WARNING: Tamper attempt detected but authentication succeeded")
		}
		return true
	}
	
	fmt.Println("Physical device authentication failed")
	return false
}