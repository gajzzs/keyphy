//go:build linux
// +build linux

package platform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

type linuxDeviceManager struct{}

func newDeviceManager() DeviceManager {
	return &linuxDeviceManager{}
}

func (dm *linuxDeviceManager) ListUSBDevices() ([]Device, error) {
	// Use gopsutil to get disk partitions
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	
	var devices []Device
	for _, partition := range partitions {
		// Focus on removable devices (USB drives)
		if strings.HasPrefix(partition.Device, "/dev/sd") {
			uuid := dm.getUUIDFromDevice(partition.Device)
			label := dm.getLabelFromDevice(partition.Device)
			
			if uuid != "" {
				mounted := partition.Mountpoint != ""
				mountPoint := partition.Mountpoint
				if !mounted {
					mountPoint = "(not mounted)"
				}
				
				devices = append(devices, Device{
					UUID:       uuid,
					Name:       label,
					Path:       partition.Device,
					Mounted:    mounted,
					MountPoint: mountPoint,
				})
			}
		}
	}
	
	return devices, nil
}

func (dm *linuxDeviceManager) GetDeviceInfo(uuid string) (*Device, error) {
	devices, err := dm.ListUSBDevices()
	if err != nil {
		return nil, err
	}
	
	for _, device := range devices {
		if device.UUID == uuid {
			return &device, nil
		}
	}
	
	return nil, nil
}

func (dm *linuxDeviceManager) IsDeviceMounted(uuid string) (bool, error) {
	// Check /proc/mounts for UUID
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return false, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, uuid) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func (dm *linuxDeviceManager) getUUIDFromDevice(devPath string) string {
	// Read UUID from /dev/disk/by-uuid/
	uuidDir := "/dev/disk/by-uuid"
	entries, err := os.ReadDir(uuidDir)
	if err != nil {
		return ""
	}
	
	for _, entry := range entries {
		linkPath := filepath.Join(uuidDir, entry.Name())
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		
		// Resolve relative path
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(linkPath), target)
		}
		target = filepath.Clean(target)
		
		if target == devPath {
			return entry.Name()
		}
	}
	return ""
}

func (dm *linuxDeviceManager) getLabelFromDevice(devPath string) string {
	// Try to get label from /dev/disk/by-label/
	labelDir := "/dev/disk/by-label"
	entries, err := os.ReadDir(labelDir)
	if err == nil {
		for _, entry := range entries {
			linkPath := filepath.Join(labelDir, entry.Name())
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}
			
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(linkPath), target)
			}
			target = filepath.Clean(target)
			
			if target == devPath {
				return entry.Name()
			}
		}
	}
	
	// Fallback: get device model from sysfs
	baseName := strings.TrimLeft(devPath, "/dev/")
	baseName = strings.TrimRight(baseName, "0123456789")
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", baseName)
	
	if data, err := os.ReadFile(modelPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	
	return "Unknown Device"
}