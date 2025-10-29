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

	// Check all block devices for USB connections
	blockDevs, err := filepath.Glob("/sys/block/sd*")
	if err != nil {
		return devices, err
	}

	for _, blockDev := range blockDevs {
		devName := filepath.Base(blockDev)
		devPath := "/dev/" + devName
		
		if isUSBDevice(devPath) {
			// Check partitions
			partitions, _ := filepath.Glob(blockDev + "*")
			for _, partition := range partitions {
				partName := filepath.Base(partition)
				partPath := "/dev/" + partName
				
				if partName != devName { // Skip the main device, only partitions
					uuid := getDeviceUUID(partPath)
					name := getDeviceName(devPath)
					mountPoint := getMountPoint(partPath)
					
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

	return devices, nil
}

func isUSBDevice(devPath string) bool {
	// Check if device is connected via USB by examining sysfs
	baseName := strings.TrimPrefix(filepath.Base(devPath), "sd")
	baseName = strings.TrimSuffix(baseName, "1") // Remove partition number
	baseName = "sd" + strings.TrimRight(baseName, "0123456789")
	sysPath := fmt.Sprintf("/sys/block/%s", baseName)
	
	// Follow symlinks to find if it's USB
	realPath, err := filepath.EvalSymlinks(sysPath)
	if err != nil {
		return false
	}
	
	return strings.Contains(realPath, "usb")
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
	baseName := strings.TrimPrefix(filepath.Base(devPath), "sd")
	baseName = strings.TrimRight(baseName, "0123456789") // Remove partition numbers
	baseName = "sd" + baseName
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", baseName)
	
	if data, err := os.ReadFile(modelPath); err == nil {
		return strings.TrimSpace(string(data))
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