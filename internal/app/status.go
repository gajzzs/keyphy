package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/gajzzs/keyphy/internal/auth"
	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/service"
	"github.com/gajzzs/keyphy/internal/system"
)

func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show comprehensive system status",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			auth := auth.NewPhysicalAuth()
			sysMonitor := system.NewSystemMonitor()
			
			fmt.Println("Keyphy System Status")
			fmt.Println("====================")
			
			// Authentication status
			fmt.Println("\nAuthentication Device:")
			if cfg.AuthDevice != "" {
				fmt.Printf("  Device UUID: %s\n", cfg.AuthDevice)
				fmt.Printf("  Device Name: %s\n", cfg.AuthDeviceName)
				fmt.Printf("  State Enforcement: %t\n", cfg.EnforceState)
				if cfg.EnforceState {
					fmt.Printf("  Required State: %s\n", cfg.AuthMountState)
				}
				
				authStatus := auth.GetAuthStatus()
				fmt.Printf("  Connected: %t\n", authStatus["device_connected"])
				fmt.Printf("  Session Valid: %t\n", authStatus["session_valid"])
				if authStatus["tamper_detected"].(bool) {
					fmt.Println("  WARNING: TAMPER DETECTED")
				}
			} else {
				fmt.Println("  No authentication device configured")
			}
			
			// Service status
			fmt.Println("\nService Status:")
			if sm, err := service.NewServiceManager(); err == nil {
				if status, err := sm.Status(); err == nil {
					fmt.Printf("  Status: %s\n", status)
				} else {
					fmt.Println("  Status: Unknown")
				}
				fmt.Printf("  Config: %s\n", service.GetServiceConfigPath())
			} else {
				fmt.Println("  Status: Not Available")
			}
			
			// System information
			fmt.Println("\nSystem Information:")
			if sysInfo, err := sysMonitor.GetSystemInfo(); err == nil {
				if hostname, ok := sysInfo["hostname"].(string); ok {
					fmt.Printf("  Hostname: %s\n", hostname)
				}
				if os, ok := sysInfo["os"].(string); ok {
					fmt.Printf("  OS: %s\n", os)
				}
				if cpuPercent, ok := sysInfo["cpu_percent"].(float64); ok {
					fmt.Printf("  CPU Usage: %.2f%%\n", cpuPercent)
				}
				if memPercent, ok := sysInfo["memory_percent"].(float64); ok {
					fmt.Printf("  Memory Usage: %.2f%%\n", memPercent)
				}
				if diskPercent, ok := sysInfo["disk_percent"].(float64); ok {
					fmt.Printf("  Disk Usage: %.2f%%\n", diskPercent)
				}
			}
			
			// Blocked items count
			fmt.Println("\nBlocking Rules:")
			fmt.Printf("  Applications: %d\n", len(cfg.BlockedApps))
			fmt.Printf("  Websites: %d\n", len(cfg.BlockedWebsites))
			fmt.Printf("  Paths: %d\n", len(cfg.BlockedPaths))
			
			// Security alerts
			fmt.Println("\nSecurity Status:")
			if alerts, err := sysMonitor.DetectSuspiciousActivity(); err == nil {
				if len(alerts) > 0 {
					fmt.Println("  Active Alerts:")
					for _, alert := range alerts {
						fmt.Printf("    - %s\n", alert)
					}
				} else {
					fmt.Println("  No suspicious activity detected")
				}
			}
			
			return nil
		},
	}
}