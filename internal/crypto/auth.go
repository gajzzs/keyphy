package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
)

func GenerateDeviceKey(deviceUUID, deviceName string) string {
	// Combine UUID and name for stronger key generation
	combined := fmt.Sprintf("%s:%s", deviceUUID, deviceName)
	
	// Use PBKDF2 for key derivation
	salt := []byte("keyphy-salt")
	key := pbkdf2.Key([]byte(combined), salt, 10000, 32, sha256.New)
	
	return hex.EncodeToString(key)
}

func ValidateDeviceAuth(deviceUUID, deviceName, expectedKey string) (bool, error) {
	if deviceUUID == "" || deviceName == "" || expectedKey == "" {
		return false, fmt.Errorf("device UUID, name, and expected key cannot be empty")
	}
	
	// Validate that expectedKey is valid hex
	if _, err := hex.DecodeString(expectedKey); err != nil {
		return false, fmt.Errorf("invalid expected key format: %v", err)
	}
	
	generatedKey := GenerateDeviceKey(deviceUUID, deviceName)
	return generatedKey == expectedKey, nil
}

func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}