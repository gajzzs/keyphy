package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Apply initial blocks
	if err := d.applyBlocks(); err != nil {
		return fmt.Errorf("failed to apply blocks: %v", err)
	}

	// Start monitoring goroutines
	go d.monitorDevices()
	go d.monitorNetwork()
	go d.monitorProcesses()
	go d.handleSignals()

	// Create PID file
	if err := CreatePidFile(); err != nil {
		log.Printf("Warning: Could not create PID file: %v", err)
	}

	log.Println("Keyphy daemon started successfully")
	return nil
}

func (d *Daemon) Stop() error {
	log.Println("Stopping keyphy daemon...")
	d.cancel()
	d.running = false

	// Remove PID file
	RemovePidFile()

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
			return crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
		}
	}
	return false
}

func (d *Daemon) applyBlocks() error {
	cfg := config.GetConfig()

	// Block applications
	for _, app := range cfg.BlockedApps {
		if err := d.appBlocker.BlockApp(app); err != nil {
			log.Printf("Failed to block app %s: %v", app, err)
		}
	}

	// Block websites
	for _, website := range cfg.BlockedWebsites {
		if err := d.networkBlocker.BlockWebsite(website); err != nil {
			log.Printf("Failed to block website %s: %v", website, err)
		}
	}

	// Block file paths
	for _, path := range cfg.BlockedPaths {
		if err := d.fileBlocker.BlockPath(path); err != nil {
			log.Printf("Failed to block path %s: %v", path, err)
		}
	}

	return nil
}

func (d *Daemon) removeAllBlocks() error {
	cfg := config.GetConfig()

	// Unblock applications
	for _, app := range cfg.BlockedApps {
		if err := d.appBlocker.UnblockApp(app); err != nil {
			log.Printf("Failed to unblock app %s: %v", app, err)
		}
	}

	// Unblock websites
	for _, website := range cfg.BlockedWebsites {
		if err := d.networkBlocker.UnblockWebsite(website); err != nil {
			log.Printf("Failed to unblock website %s: %v", website, err)
		}
	}

	// Unblock file paths
	for _, path := range cfg.BlockedPaths {
		if err := d.fileBlocker.UnblockPath(path); err != nil {
			log.Printf("Failed to unblock path %s: %v", path, err)
		}
	}

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
						d.applyBlocks()
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
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)

	for {
		select {
		case <-d.ctx.Done():
			return
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGUSR1:
				log.Println("Received unlock signal")
				if d.validateDeviceAuth() {
					log.Println("Device authenticated, removing blocks")
					d.removeAllBlocks()
				} else {
					log.Println("Device authentication failed")
				}
			case syscall.SIGUSR2:
				log.Println("Received lock signal")
				if d.validateDeviceAuth() {
					log.Println("Device authenticated, applying blocks")
					d.applyBlocks()
				} else {
					log.Println("Device authentication failed")
				}
			}
		}
	}
}