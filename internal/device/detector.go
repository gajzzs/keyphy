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

	// Read from /proc/mounts to find mounted USB devices
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			devPath := fields[0]
			mountPoint := fields[1]
			fsType := fields[2]

			// Check if it's a USB device
			if strings.HasPrefix(devPath, "/dev/sd") && (fsType == "vfat" || fsType == "ext4" || fsType == "ntfs") {
				if isUSBDevice(devPath) {
					uuid := getDeviceUUID(devPath)
					name := getDeviceName(devPath)
					devices = append(devices, Device{
						UUID:       uuid,
						Name:       name,
						MountPoint: mountPoint,
						DevPath:    devPath,
					})
				}
			}
		}
	}

	return devices, scanner.Err()
}

func isUSBDevice(devPath string) bool {
	// Check if device is connected via USB by examining sysfs
	baseName := filepath.Base(devPath)
	sysPath := fmt.Sprintf("/sys/block/%s", strings.TrimPrefix(baseName, "/dev/"))
	
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
	baseName := strings.TrimPrefix(devPath, "/dev/")
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", baseName)
	
	if data, err := os.ReadFile(modelPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	
	return baseName
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