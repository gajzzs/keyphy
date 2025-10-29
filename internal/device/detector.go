package device

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Device struct {
	UUID       string
	Name       string
	MountPoint string
	DevPath    string
}

func ListUSBDevices() ([]Device, error) {
	var devices []Device

	// Check all removable block devices
	blockDevs, err := filepath.Glob("/sys/block/*")
	if err != nil {
		return devices, err
	}

	for _, blockDev := range blockDevs {
		devName := filepath.Base(blockDev)
		
		// Check if device is removable
		removablePath := filepath.Join(blockDev, "removable")
		if data, err := os.ReadFile(removablePath); err == nil {
			removableFlag := strings.TrimSpace(string(data))
			fmt.Printf("DEBUG: Device %s removable=%s\n", devName, removableFlag)
			if removableFlag == "1" {
				// Check partitions
				partitions, _ := filepath.Glob(blockDev + "*")
				fmt.Printf("DEBUG: Found %d partitions for %s\n", len(partitions), devName)
				for _, partition := range partitions {
					partName := filepath.Base(partition)
					partPath := "/dev/" + partName
					fmt.Printf("DEBUG: Checking partition %s\n", partName)
					
					if partName != devName { // Skip the main device, only partitions
						uuid := getDeviceUUID(partPath)
						name := getDeviceName("/dev/" + devName)
						mountPoint := getMountPoint(partPath)
						fmt.Printf("DEBUG: UUID=%s, Name=%s, Mount=%s\n", uuid, name, mountPoint)
						
						// Check for encrypted mapper devices
						encryptedUUID, encryptedMount := getEncryptedDeviceInfo(partPath)
						if encryptedUUID != "" {
							uuid = encryptedUUID
							mountPoint = encryptedMount
							name += " (encrypted)"
							fmt.Printf("DEBUG: Found encrypted: UUID=%s, Mount=%s\n", uuid, mountPoint)
						}
						
						// Always add removable devices, even without UUID
						if uuid == "" {
							uuid = "NO-UUID-" + partName // Fallback identifier
						}
						
						devices = append(devices, Device{
							UUID:       uuid,
							Name:       name,
							MountPoint: mountPoint,
							DevPath:    partPath,
						})
						fmt.Printf("DEBUG: Added device: %+v\n", devices[len(devices)-1])
					}
				}
			}
		}
	}

	return devices, nil
}

func isRemovableDevice(devName string) bool {
	// Check if device is removable
	removablePath := fmt.Sprintf("/sys/block/%s/removable", devName)
	data, err := os.ReadFile(removablePath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

func getDeviceUUID(devPath string) string {
	// Try UUID first with sudo
	cmd := exec.Command("sudo", "blkid", "-s", "UUID", "-o", "value", devPath)
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return strings.TrimSpace(string(output))
	}
	
	// Fallback to PARTUUID with sudo
	cmd = exec.Command("sudo", "blkid", "-s", "PARTUUID", "-o", "value", devPath)
	output, err = cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return strings.TrimSpace(string(output))
	}
	
	return ""
}

func getDeviceName(devPath string) string {
	// Get device model from sysfs
	baseName := filepath.Base(devPath)
	// Remove partition numbers to get base device name
	baseName = strings.TrimRight(baseName, "0123456789")
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", baseName)
	
	if data, err := os.ReadFile(modelPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	
	// Try vendor + product as fallback
	vendorPath := fmt.Sprintf("/sys/block/%s/device/vendor", baseName)
	productPath := fmt.Sprintf("/sys/block/%s/device/product", baseName)
	
	vendor := ""
	product := ""
	
	if data, err := os.ReadFile(vendorPath); err == nil {
		vendor = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(productPath); err == nil {
		product = strings.TrimSpace(string(data))
	}
	
	if vendor != "" && product != "" {
		return vendor + " " + product
	}
	
	return baseName
}

func getMountPoint(devPath string) string {
	// Check if device is mounted (including encrypted mounts)
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "(not mounted)"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Direct mount
			if fields[0] == devPath {
				return fields[1]
			}
			// Check for encrypted device mounts (veracrypt, luks, etc.)
			if strings.Contains(fields[0], "mapper/") && strings.Contains(line, filepath.Base(devPath)) {
				return fields[1] + " (encrypted)"
			}
		}
	}
	return "(not mounted)"
}

func getEncryptedDeviceInfo(partPath string) (string, string) {
	// Check if this partition has an encrypted mapper device
	mapperDevs, err := filepath.Glob("/dev/mapper/*")
	if err != nil {
		return "", ""
	}
	
	for _, mapperDev := range mapperDevs {
		// Check if mapper device is based on this partition
		cmd := exec.Command("sudo", "lsblk", "-no", "PKNAME", mapperDev)
		output, err := cmd.Output()
		if err == nil {
			parentDev := strings.TrimSpace(string(output))
			if "/dev/"+parentDev == partPath {
				// Get UUID of encrypted device
				uuid := getDeviceUUID(mapperDev)
				mountPoint := getMountPoint(mapperDev)
				return uuid, mountPoint
			}
		}
	}
	return "", ""
}

func IsDeviceConnected(uuid string) bool {
	devices, err := ListUSBDevices()
	if err != nil {
		return false
	}
	
	for _, device := range devices {
		if device.UUID == uuid {
			return true
		}
	}
	return false
}