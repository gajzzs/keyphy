package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gajzzs/keyphy/internal/config"
	"github.com/gajzzs/keyphy/internal/crypto"
	"github.com/gajzzs/keyphy/internal/device"
)

type PhysicalAuth struct {
	deviceFingerprint string
	lastAuthTime      time.Time
	authChallenge     string
	tamperDetected    bool
}

func NewPhysicalAuth() *PhysicalAuth {
	return &PhysicalAuth{}
}

// AuthenticateDevice performs robust physical device authentication
func (pa *PhysicalAuth) AuthenticateDevice() (bool, error) {
	cfg := config.GetConfig()
	if cfg.AuthDevice == "" || cfg.AuthKey == "" {
		return false, fmt.Errorf("no authentication device configured")
	}

	// Generate challenge for this authentication session
	challenge, err := pa.generateChallenge()
	if err != nil {
		return false, fmt.Errorf("failed to generate auth challenge: %v", err)
	}
	pa.authChallenge = challenge

	devices, err := device.ListUSBDevices()
	if err != nil {
		return false, fmt.Errorf("failed to scan USB devices: %v", err)
	}

	for _, dev := range devices {
		if dev.UUID == cfg.AuthDevice {
			// Create device fingerprint for tamper detection
			fingerprint := pa.createDeviceFingerprint(dev)
			
			// Check for device tampering
			if pa.deviceFingerprint != "" && pa.deviceFingerprint != fingerprint {
				pa.tamperDetected = true
				return false, fmt.Errorf("device fingerprint mismatch - potential device cloning detected")
			}
			pa.deviceFingerprint = fingerprint

			// Validate device state if enforcement enabled
			if cfg.EnforceState {
				if !pa.validateDeviceState(dev, cfg) {
					return false, fmt.Errorf("device state validation failed")
				}
			}

			// Perform cryptographic authentication
			valid, err := crypto.ValidateDeviceAuth(dev.UUID, dev.Name, cfg.AuthKey)
			if err != nil {
				return false, fmt.Errorf("cryptographic validation failed: %v", err)
			}

			if valid {
				pa.lastAuthTime = time.Now()
				pa.tamperDetected = false
				
				// Create authentication token for this session
				if err := pa.createSessionToken(dev); err != nil {
					return false, fmt.Errorf("failed to create session token: %v", err)
				}
				
				return true, nil
			}
			return false, fmt.Errorf("device authentication failed")
		}
	}
	
	return false, fmt.Errorf("authentication device not connected")
}

// createDeviceFingerprint creates unique fingerprint to detect device cloning
func (pa *PhysicalAuth) createDeviceFingerprint(dev device.Device) string {
	data := fmt.Sprintf("%s:%s:%s", dev.UUID, dev.Name, dev.DevPath)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// validateDeviceState ensures device is in required mount state
func (pa *PhysicalAuth) validateDeviceState(dev device.Device, cfg *config.Config) bool {
	currentState := "unmounted"
	if dev.MountPoint != "(not mounted)" {
		if strings.Contains(dev.MountPoint, "encrypted") {
			currentState = "mounted-encrypted"
		} else {
			currentState = "mounted"
		}
	}
	
	return currentState == cfg.AuthMountState
}

// generateChallenge creates cryptographic challenge for session
func (pa *PhysicalAuth) generateChallenge() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// createSessionToken creates tamper-evident session token
func (pa *PhysicalAuth) createSessionToken(dev device.Device) error {
	tokenDir := "/tmp/keyphy-auth"
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return err
	}
	
	tokenData := fmt.Sprintf("%s:%s:%d:%s", 
		dev.UUID, pa.authChallenge, time.Now().Unix(), pa.deviceFingerprint)
	
	tokenPath := filepath.Join(tokenDir, "session.token")
	if err := os.WriteFile(tokenPath, []byte(tokenData), 0600); err != nil {
		return err
	}
	
	// Set immutable attribute (Linux)
	os.Chmod(tokenPath, 0400) // Read-only
	return nil
}

// ValidateSession checks if current session is valid
func (pa *PhysicalAuth) ValidateSession() bool {
	tokenPath := "/tmp/keyphy-auth/session.token"
	
	// Check if token exists and is recent (within 5 minutes)
	if stat, err := os.Stat(tokenPath); err == nil {
		if time.Since(stat.ModTime()) < 5*time.Minute {
			// Verify token content
			if data, err := os.ReadFile(tokenPath); err == nil {
				parts := strings.Split(string(data), ":")
				if len(parts) >= 4 {
					// Verify fingerprint matches current device
					cfg := config.GetConfig()
					devices, err := device.ListUSBDevices()
					if err == nil {
						for _, dev := range devices {
							if dev.UUID == cfg.AuthDevice {
								currentFingerprint := pa.createDeviceFingerprint(dev)
								return parts[3] == currentFingerprint
							}
						}
					}
				}
			}
		}
	}
	
	// Clean up invalid token
	os.Remove(tokenPath)
	return false
}

// IsDeviceConnected checks if auth device is currently connected
func (pa *PhysicalAuth) IsDeviceConnected() bool {
	cfg := config.GetConfig()
	if cfg.AuthDevice == "" {
		return false
	}
	
	devices, err := device.ListUSBDevices()
	if err != nil {
		return false
	}
	
	for _, dev := range devices {
		if dev.UUID == cfg.AuthDevice {
			return true
		}
	}
	return false
}

// GetAuthStatus returns current authentication status
func (pa *PhysicalAuth) GetAuthStatus() map[string]interface{} {
	return map[string]interface{}{
		"device_connected":  pa.IsDeviceConnected(),
		"session_valid":     pa.ValidateSession(),
		"last_auth_time":    pa.lastAuthTime,
		"tamper_detected":   pa.tamperDetected,
		"challenge_active":  pa.authChallenge != "",
	}
}

// ClearSession removes current authentication session
func (pa *PhysicalAuth) ClearSession() error {
	tokenPath := "/tmp/keyphy-auth/session.token"
	pa.authChallenge = ""
	pa.lastAuthTime = time.Time{}
	return os.Remove(tokenPath)
}