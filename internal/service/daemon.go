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
	"keyphy/internal/blocker"
	"keyphy/internal/config"
	"keyphy/internal/crypto"
	"keyphy/internal/device"
)

type Daemon struct {
	appBlocker     *blocker.AppBlocker
	networkBlocker *blocker.NetworkBlocker
	fileBlocker    *blocker.FileBlocker
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
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (d *Daemon) Start() error {
	if d.running {
		return fmt.Errorf("daemon already running")
	}

	d.running = true
	log.Println("Keyphy daemon starting...")

	// Enable process protection
	if err := d.enableProcessProtection(); err != nil {
		log.Printf("Warning: Could not enable process protection: %v", err)
	}

	// Apply initial blocks
	if err := d.applyBlocks(); err != nil {
		return fmt.Errorf("failed to apply blocks: %v", err)
	}
	d.blocksActive = true

	// Start monitoring goroutines
	go d.monitorDevices()
	go d.monitorNetwork()
	go d.monitorProcesses()
	go d.monitorConfigFile()
	go d.handleSignals()
	go d.selfProtection()

	// Create PID file
	if err := CreatePidFile(); err != nil {
		log.Printf("Warning: Could not create PID file: %v", err)
	} else {
		log.Printf("PID file created: /var/run/keyphy.pid (PID: %d)", os.Getpid())
	}

	log.Println("Keyphy daemon started successfully")
	return nil
}

func (d *Daemon) Stop() error {
	log.Println("Stopping keyphy daemon...")
	d.cancel()
	d.running = false

	// Remove PID file
	if err := RemovePidFile(); err != nil {
		log.Printf("Warning: Failed to remove PID file: %v", err)
	} else {
		log.Println("PID file removed successfully")
	}

	// Remove all blocks when stopping
	return d.removeAllBlocks()
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

	// Block websites
	for _, website := range cfg.BlockedWebsites {
		log.Printf("Blocking website: %s", website)
		if err := d.networkBlocker.BlockWebsite(website); err != nil {
			log.Printf("Failed to block website %s: %v", website, err)
		} else {
			log.Printf("Successfully blocked website: %s", website)
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

	log.Println("All blocking rules applied successfully")
	return nil
}

func (d *Daemon) removeAllBlocks() error {
	cfg := config.GetConfig()

	log.Println("Removing all blocking rules...")
	// Unblock applications
	for _, app := range cfg.BlockedApps {
		log.Printf("Unblocking application: %s", app)
		if err := d.appBlocker.UnblockApp(app); err != nil {
			log.Printf("Failed to unblock app %s: %v", app, err)
		} else {
			log.Printf("Successfully unblocked application: %s", app)
		}
	}

	// Unblock websites
	for _, website := range cfg.BlockedWebsites {
		log.Printf("Unblocking website: %s", website)
		if err := d.networkBlocker.UnblockWebsite(website); err != nil {
			log.Printf("Failed to unblock website %s: %v", website, err)
		} else {
			log.Printf("Successfully unblocked website: %s", website)
		}
	}

	// Unblock file paths
	for _, path := range cfg.BlockedPaths {
		log.Printf("Unblocking path: %s", path)
		if err := d.fileBlocker.UnblockPath(path); err != nil {
			log.Printf("Failed to unblock path %s: %v", path, err)
		} else {
			log.Printf("Successfully unblocked path: %s", path)
		}
	}

	log.Println("All blocking rules removed successfully")
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
			if err := d.networkBlocker.MonitorNetworkTraffic(); err != nil {
				log.Printf("Network monitoring error: %v", err)
			}
			// Monitor hosts file integrity
			if err := d.networkBlocker.VerifyHostsFile(); err != nil {
				log.Printf("Hosts file verification failed: %v", err)
			}
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
			for _, app := range cfg.BlockedApps {
				if pids, err := d.appBlocker.GetRunningProcesses(app); err == nil {
					for _, pid := range pids {
						if err := d.appBlocker.BlockProcessLaunch(pid); err != nil {
							log.Printf("Failed to block process %d: %v", pid, err)
						}
					}
				}
			}
		}
	}
}

func (d *Daemon) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	for {
		select {
		case <-d.ctx.Done():
			return
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGUSR1:
				log.Println("Received unlock signal")
				log.Println("Removing blocks...")
				if err := d.removeAllBlocks(); err != nil {
					log.Printf("Failed to remove blocks: %v", err)
				} else {
					log.Println("Blocks removed successfully")
					d.blocksActive = false
				}
			case syscall.SIGUSR2:
				log.Println("Received lock signal")
				log.Println("Applying blocks...")
				if err := d.applyBlocks(); err != nil {
					log.Printf("Failed to apply blocks: %v", err)
				} else {
					log.Println("Blocks applied successfully")
					d.blocksActive = true
				}
			case syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL:
				// Require auth device for termination
				if !d.validateDeviceAuth() {
					log.Println("Termination attempt blocked - auth device required")
					continue // Ignore termination signal
				}
				log.Println("Auth device verified, shutting down...")
				d.Stop()
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
				d.Start()
			}
		}
	}
}