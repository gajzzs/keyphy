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
	salt := []byte("keyphy-salt-2024")
	key := pbkdf2.Key([]byte(combined), salt, 10000, 32, sha256.New)
	
	return hex.EncodeToString(key)
}

func ValidateDeviceAuth(deviceUUID, deviceName, expectedKey string) bool {
	generatedKey := GenerateDeviceKey(deviceUUID, deviceName)
	return generatedKey == expectedKey
}

func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}