package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"os/exec"
	"github.com/kardianos/service"
	"github.com/gajzzs/keyphy/internal/blocker"
	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/crypto"
	"github.com/gajzzs/keyphy/internal/device"
	"github.com/gajzzs/keyphy/internal/system"
)

type Daemon struct {
	appBlocker     *blocker.AppBlocker
	networkBlocker *blocker.NetworkBlocker
	fileBlocker    *blocker.FileBlocker
	sysMonitor     *system.SystemMonitor
	protection     *ServiceProtection
	running        bool
	blocksActive   bool
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewDaemon() *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		appBlocker:     blocker.NewAppBlocker(),
		networkBlocker: blocker.NewNetworkBlocker(),
		fileBlocker:    blocker.NewFileBlocker(),
		sysMonitor:     system.NewSystemMonitor(),
		protection:     NewServiceProtection(),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (d *Daemon) Start(s service.Service) error {
	return d.startDaemon()
}

func (d *Daemon) startDaemon() error {
	if d.running {
		return fmt.Errorf("daemon already running")
	}

	d.running = true
	log.Println("Keyphy daemon starting...")

	// Enable process protection
	if err := d.enableProcessProtection(); err != nil {
		log.Printf("Warning: Could not enable process protection: %v", err)
	}

	// Only start DNS system if there are websites to block
	cfg := config.GetConfig()
	if len(cfg.BlockedWebsites) > 0 {
		if err := d.networkBlocker.StartDNSSystem(); err != nil {
			log.Printf("Warning: DNS system failed to start: %v", err)
			log.Println("Falling back to platform-specific network blocking")
		}
	} else {
		log.Println("No websites to block - DNS system not started")
	}
	
	// Apply initial blocks
	if err := d.applyBlocks(); err != nil {
		return fmt.Errorf("failed to apply blocks: %v", err)
	}
	d.blocksActive = true

	// Start service protection
	if err := d.protection.Start(); err != nil {
		log.Printf("Warning: Service protection failed to start: %v", err)
	}
	
	// Start monitoring goroutines
	go d.monitorDevices()
	go d.monitorNetwork()
	go d.monitorProcesses()
	go d.monitorConfigFile()
	go d.monitorIntegrity()
	go d.monitorSystemActivity()
	go d.handleSignals()
	go d.selfProtection()

	// PID file managed by service package

	log.Println("Keyphy daemon started successfully")
	return nil
}

func (d *Daemon) Stop(s service.Service) error {
	return d.stopDaemon()
}

func (d *Daemon) stopDaemon() error {
	log.Println("Stopping keyphy daemon...")
	d.cancel()
	d.running = false

	// Stop service protection
	if d.protection != nil {
		d.protection.Stop()
		d.protection.RemoveProtection()
	}

	// Remove all blocks when stopping
	if err := d.removeAllBlocks(); err != nil {
		return err
	}
	
	// Stop network blocker (including DNS system)
	if err := d.networkBlocker.Stop(); err != nil {
		log.Printf("Warning: Failed to stop network blocker: %v", err)
	}
	
	return nil
}

func (d *Daemon) UnlockWithAuth() error {
	if !d.validateDeviceAuth() {
		return fmt.Errorf("authentication device not connected or invalid")
	}
	
	log.Println("Device authenticated, removing all blocks")
	if err := d.removeAllBlocks(); err != nil {
		return fmt.Errorf("failed to remove blocks: %v", err)
	}
	d.blocksActive = false
	
	return nil
}

func (d *Daemon) LockWithAuth() error {
	if !d.validateDeviceAuth() {
		return fmt.Errorf("authentication device not connected or invalid")
	}
	
	log.Println("Device authenticated, applying all blocks")
	if err := d.applyBlocks(); err != nil {
		return fmt.Errorf("failed to apply blocks: %v", err)
	}
	d.blocksActive = true
	
	return nil
}

func (d *Daemon) validateDeviceAuth() bool {
	cfg := config.GetConfig()
	devices, err := device.ListUSBDevices()
	if err != nil {
		return false
	}

	for _, dev := range devices {
		if dev.UUID == cfg.AuthDevice {
			valid, err := crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
			if err != nil {
				return false
			}
			return valid
		}
	}
	return false
}

func (d *Daemon) applyBlocks() error {
	cfg := config.GetConfig()

	log.Println("Applying blocking rules...")
	// Block applications
	for _, app := range cfg.BlockedApps {
		log.Printf("Blocking application: %s", app)
		if err := d.appBlocker.BlockApp(app); err != nil {
			log.Printf("Failed to block app %s: %v", app, err)
		} else {
			log.Printf("Successfully blocked application: %s", app)
		}
	}

	// Block websites - start DNS system if needed
	if len(cfg.BlockedWebsites) > 0 {
		// Ensure DNS system is running
		if !d.networkBlocker.IsUsingDNS() {
			if err := d.networkBlocker.StartDNSSystem(); err != nil {
				log.Printf("Warning: DNS system failed to start: %v", err)
			}
		}
		
		for _, website := range cfg.BlockedWebsites {
			log.Printf("Blocking website: %s", website)
			if err := d.networkBlocker.BlockWebsite(website); err != nil {
				log.Printf("Failed to block website %s: %v", website, err)
			} else {
				log.Printf("Successfully blocked website: %s", website)
			}
		}
	}

	// Block file paths
	for _, path := range cfg.BlockedPaths {
		log.Printf("Blocking path: %s", path)
		if err := d.fileBlocker.BlockPath(path); err != nil {
			log.Printf("Failed to block path %s: %v", path, err)
		} else {
			log.Printf("Successfully blocked path: %s", path)
		}
	}
	
	// Block IP addresses
	for _, ip := range cfg.BlockedIPs {
		log.Printf("Blocking IP: %s", ip)
		if err := d.networkBlocker.BlockIP(ip); err != nil {
			log.Printf("Failed to block IP %s: %v", ip, err)
		} else {
			log.Printf("Successfully blocked IP: %s", ip)
		}
	}

	log.Println("All blocking rules applied successfully")
	return nil
}

func (d *Daemon) removeAllBlocks() error {
	cfg := config.GetConfig()

	log.Println("Removing all blocking rules and restoring system state...")
	
	// DNS system handles restoration automatically
	
	// Unblock websites (removes hosts entries and iptables rules)
	for _, website := range cfg.BlockedWebsites {
		log.Printf("Unblocking website: %s", website)
		if err := d.networkBlocker.UnblockWebsite(website); err != nil {
			log.Printf("Failed to unblock website %s: %v", website, err)
		} else {
			log.Printf("Successfully unblocked website: %s", website)
		}
	}
	
	// Remove all network blocks
	if err := d.networkBlocker.UnblockAll(); err != nil {
		log.Printf("Warning: Failed to remove all network blocks: %v", err)
	}

	// Unblock applications and restore original permissions
	for _, app := range cfg.BlockedApps {
		log.Printf("Unblocking application: %s", app)
		if err := d.appBlocker.UnblockApp(app); err != nil {
			log.Printf("Failed to unblock app %s: %v", app, err)
		} else {
			log.Printf("Successfully unblocked application: %s", app)
		}
	}

	// Unblock file paths and restore original permissions
	for _, path := range cfg.BlockedPaths {
		log.Printf("Unblocking path: %s", path)
		if err := d.fileBlocker.UnblockPath(path); err != nil {
			log.Printf("Failed to unblock path %s: %v", path, err)
		} else {
			log.Printf("Successfully unblocked path: %s", path)
		}
	}
	
	// Unblock IP addresses
	for _, ip := range cfg.BlockedIPs {
		log.Printf("Unblocking IP: %s", ip)
		if err := d.networkBlocker.UnblockIP(ip); err != nil {
			log.Printf("Failed to unblock IP %s: %v", ip, err)
		} else {
			log.Printf("Successfully unblocked IP: %s", ip)
		}
	}

	log.Println("All blocking rules removed and system state restored successfully")
	return nil
}

func (d *Daemon) monitorDevices() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	lastDeviceState := false

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			cfg := config.GetConfig()
			if cfg.AuthDevice != "" && cfg.AuthKey != "" {
				currentDeviceState := d.validateDeviceAuth()
				
				// Only log state changes
				if currentDeviceState != lastDeviceState {
					if currentDeviceState {
						log.Println("Auth device connected and authenticated")
					} else {
						log.Println("Auth device disconnected or authentication failed")
						// Apply blocks when device is removed
						if err := d.applyBlocks(); err != nil {
							log.Printf("Failed to apply blocks after device disconnection: %v", err)
						} else {
							d.blocksActive = true
						}
					}
					lastDeviceState = currentDeviceState
				}
			}
		}
	}
}

func (d *Daemon) monitorNetwork() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// Network monitoring handled by DNS system
		}
	}
}

func (d *Daemon) monitorProcesses() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// Only monitor processes if blocks are active
			if !d.blocksActive {
				continue
			}
			cfg := config.GetConfig()
			
			// Use gopsutil for enhanced process monitoring
			if blockedPids, err := d.sysMonitor.MonitorProcesses(cfg.BlockedApps); err == nil {
				for _, pid := range blockedPids {
					if err := d.appBlocker.BlockProcessLaunch(int(pid)); err != nil {
						log.Printf("Failed to block process %d: %v", pid, err)
					}
				}
			}
		}
	}
}

func (d *Daemon) monitorSystemActivity() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if !d.blocksActive {
				continue
			}
			
			// Monitor for suspicious activity
			if alerts, err := d.sysMonitor.DetectSuspiciousActivity(); err == nil {
				for _, alert := range alerts {
					log.Printf("SECURITY ALERT: %s", alert)
				}
			}
			
			// Log system info periodically
			if sysInfo, err := d.sysMonitor.GetSystemInfo(); err == nil {
				if cpuPercent, ok := sysInfo["cpu_percent"].(float64); ok && cpuPercent > 80 {
					log.Printf("High CPU usage: %.2f%%", cpuPercent)
				}
				if memPercent, ok := sysInfo["memory_percent"].(float64); ok && memPercent > 90 {
					log.Printf("High memory usage: %.2f%%", memPercent)
				}
			}
		}
	}
}

func (d *Daemon) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-d.ctx.Done():
			return
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGUSR1:
				log.Println("Received unlock signal")
				if !d.validateDeviceAuth() {
					log.Println("Unlock denied - auth device not connected or invalid")
					continue
				}
				log.Println("Device authenticated, removing blocks...")
				if err := d.removeAllBlocks(); err != nil {
					log.Printf("Failed to remove blocks: %v", err)
				} else {
					log.Println("Blocks removed successfully")
					d.blocksActive = false
				}
			case syscall.SIGUSR2:
				log.Println("Received lock signal")
				if !d.validateDeviceAuth() {
					log.Println("Lock denied - auth device not connected or invalid")
					continue
				}
				log.Println("Device authenticated, applying blocks...")
				if err := d.applyBlocks(); err != nil {
					log.Printf("Failed to apply blocks: %v", err)
				} else {
					log.Println("Blocks applied successfully")
					d.blocksActive = true
				}
			case syscall.SIGTERM, syscall.SIGINT:
				// Require auth device for termination
				if !d.validateDeviceAuth() {
					log.Println("Termination attempt blocked - auth device required")
					continue // Ignore termination signal
				}
				log.Println("Auth device verified, shutting down...")
				d.stopDaemon()
				os.Exit(0)
			}
		}
	}
}

func (d *Daemon) monitorConfigFile() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	lastModTime := time.Time{}
	configFile := "/etc/keyphy/config.json"
	
	// Get initial modification time
	if stat, err := os.Stat(configFile); err == nil {
		lastModTime = stat.ModTime()
	}
	
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if stat, err := os.Stat(configFile); err == nil {
				if !stat.ModTime().Equal(lastModTime) {
					log.Println("WARNING: Config file modification detected!")
					log.Printf("Previous: %v, Current: %v", lastModTime, stat.ModTime())
					
					// Reload config and reapply blocks
					log.Println("Reloading configuration and reapplying blocks...")
					config.InitConfig()
					d.applyBlocks()
					
					lastModTime = stat.ModTime()
				}
			} else {
				log.Printf("Config file monitoring error: %v", err)
			}
		}
	}
}

func (d *Daemon) enableProcessProtection() error {
	// Set process name to make it harder to identify
	exec.Command("prctl", "PR_SET_NAME", "[kworker/0:1]").Run()
	return nil
}

func (d *Daemon) monitorIntegrity() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if !d.blocksActive {
				continue
			}
			
			// Verify integrity of blocked executables
			if err := d.appBlocker.VerifyIntegrity(); err != nil {
				log.Printf("SECURITY ALERT: Integrity violation detected - %v", err)
				log.Println("Re-enforcing all blocks...")
				
				// Re-enforce blocks immediately
				if err := d.appBlocker.EnforceBlocks(); err != nil {
					log.Printf("CRITICAL: Failed to re-enforce blocks: %v", err)
				} else {
					log.Println("Blocks re-enforced successfully")
				}
			}
		}
	}
}

func (d *Daemon) selfProtection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// Restart if killed without auth
			if !d.running {
				log.Println("Daemon killed unexpectedly, restarting...")
				d.startDaemon()
			}
		}
	}
}