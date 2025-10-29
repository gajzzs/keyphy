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
			if strings.TrimSpace(string(data)) == "1" {
				// Check partitions
				partitions, _ := filepath.Glob(blockDev + "*")
				for _, partition := range partitions {
					partName := filepath.Base(partition)
					partPath := "/dev/" + partName
					
					if partName != devName { // Skip the main device, only partitions
						uuid := getDeviceUUID(partPath)
						name := getDeviceName("/dev/" + devName)
						mountPoint := getMountPoint(partPath)
						
						if uuid != "" { // Only add if UUID exists
							devices = append(devices, Device{
								UUID:       uuid,
								Name:       name,
								MountPoint: mountPoint,
								DevPath:    partPath,
							})
						}
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
	// Use blkid to get UUID
	cmd := exec.Command("blkid", "-s", "UUID", "-o", "value", devPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
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
	// Check if device is mounted
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "(not mounted)"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == devPath {
			return fields[1]
		}
	}
	return "(not mounted)"
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