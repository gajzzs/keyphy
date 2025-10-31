//go:build linux
// +build linux

package platform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type linuxDeviceManager struct{}

func newDeviceManager() DeviceManager {
	return &linuxDeviceManager{}
}

func (dm *linuxDeviceManager) ListUSBDevices() ([]Device, error) {
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
			
			if removableFlag == "1" {
				// Check partitions in /dev/
				partitions, _ := filepath.Glob("/dev/" + devName + "*")
				
				for _, partPath := range partitions {
					partName := filepath.Base(partPath)
					
					if partName != devName { // Skip the main device, only partitions
						uuid := dm.getDeviceUUID(partPath)
						name := dm.getDeviceName("/dev/" + devName)
						mounted, mountPoint := dm.getMountInfo(partPath)
						
						// Check for encrypted mapper devices
						encryptedUUID, encryptedMount := dm.getEncryptedDeviceInfo(partPath)
						if encryptedUUID != "" {
							uuid = encryptedUUID
							mountPoint = encryptedMount
							mounted = true
							name += " (encrypted)"
						}
						
						// Always add removable devices, even without UUID
						if uuid == "" {
							uuid = "NO-UUID-" + partName // Fallback identifier
						}
						
						devices = append(devices, Device{
							UUID:       uuid,
							Name:       name,
							Path:       partPath,
							Mounted:    mounted,
							MountPoint: mountPoint,
						})
					}
				}
				
				// Always also add the whole disk as an option
				diskPath := "/dev/" + devName
				uuid := dm.getDeviceUUID(diskPath)
				name := dm.getDeviceName(diskPath)
				mounted, mountPoint := dm.getMountInfo(diskPath)
				
				if uuid == "" {
					uuid = "NO-UUID-" + devName
				}
				
				devices = append(devices, Device{
					UUID:       uuid,
					Name:       name + " (whole disk)",
					Path:       diskPath,
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

func (dm *linuxDeviceManager) getDeviceUUID(devPath string) string {
	// Method 1: Try UUID from /dev/disk/by-uuid/ (for formatted filesystems)
	if uuid := dm.getUUIDFromSymlink(devPath); uuid != "" {
		return uuid
	}
	
	// Method 2: Try PARTUUID from /dev/disk/by-partuuid/ (for GPT partitions)
	if partuuid := dm.getPARTUUIDFromSymlink(devPath); partuuid != "" {
		return partuuid
	}
	
	// Method 3: Try to read UUID directly from sysfs
	if sysfsUUID := dm.getUUIDFromSysfs(devPath); sysfsUUID != "" {
		return sysfsUUID
	}
	
	// Method 4: Use device serial number as unique identifier
	if serial := dm.getDeviceSerial(devPath); serial != "" {
		return "SERIAL-" + serial
	}
	
	// Method 5: Fallback to device name for identification
	return "DEV-" + filepath.Base(devPath)
}

func (dm *linuxDeviceManager) getUUIDFromSymlink(devPath string) string {
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

func (dm *linuxDeviceManager) getPARTUUIDFromSymlink(devPath string) string {
	partuuidDir := "/dev/disk/by-partuuid"
	entries, err := os.ReadDir(partuuidDir)
	if err != nil {
		return ""
	}
	
	for _, entry := range entries {
		linkPath := filepath.Join(partuuidDir, entry.Name())
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
	return ""
}

func (dm *linuxDeviceManager) getUUIDFromSysfs(devPath string) string {
	// Try to read UUID from sysfs partition info
	baseName := filepath.Base(devPath)
	partitionDir := "/sys/class/block/" + baseName
	
	// Check if partition directory exists
	if _, err := os.Stat(partitionDir); err != nil {
		return ""
	}
	
	// Try to read partition UUID from uevent
	ueventPath := filepath.Join(partitionDir, "uevent")
	if data, err := os.ReadFile(ueventPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PARTUUID=") {
				return strings.TrimPrefix(line, "PARTUUID=")
			}
		}
	}
	
	return ""
}

func (dm *linuxDeviceManager) getDeviceSerial(devPath string) string {
	// Get device serial number from sysfs
	baseName := strings.TrimRight(filepath.Base(devPath), "0123456789")
	serialPath := fmt.Sprintf("/sys/block/%s/device/serial", baseName)
	
	if data, err := os.ReadFile(serialPath); err == nil {
		serial := strings.TrimSpace(string(data))
		if serial != "" && serial != "0" {
			return serial
		}
	}
	
	// Try WWN (World Wide Name) as alternative
	wwnPath := fmt.Sprintf("/sys/block/%s/wwid", baseName)
	if data, err := os.ReadFile(wwnPath); err == nil {
		wwn := strings.TrimSpace(string(data))
		if wwn != "" {
			return wwn
		}
	}
	
	return ""
}

func (dm *linuxDeviceManager) getDeviceName(devPath string) string {
	// Get device model from sysfs
	baseName := filepath.Base(devPath)
	// Remove partition numbers to get base device name
	baseName = strings.TrimRight(baseName, "0123456789")
	
	// Method 1: Try to get filesystem label first
	if label := dm.getFilesystemLabel(devPath); label != "" {
		return label
	}
	
	// Method 2: Try device model
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", baseName)
	if data, err := os.ReadFile(modelPath); err == nil {
		model := strings.TrimSpace(string(data))
		if model != "" {
			return model
		}
	}
	
	// Method 3: Try vendor + product combination
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
	if vendor != "" {
		return vendor + " Device"
	}
	if product != "" {
		return product
	}
	
	// Method 4: Fallback to device name
	return "USB Device (" + baseName + ")"
}

func (dm *linuxDeviceManager) getFilesystemLabel(devPath string) string {
	// Try to get label from /dev/disk/by-label/
	labelDir := "/dev/disk/by-label"
	entries, err := os.ReadDir(labelDir)
	if err != nil {
		return ""
	}
	
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
	return ""
}

func (dm *linuxDeviceManager) getMountInfo(devPath string) (bool, string) {
	// Check if device is mounted (including encrypted mounts)
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return false, "(not mounted)"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Direct mount
			if fields[0] == devPath {
				return true, fields[1]
			}
			// Check for encrypted device mounts (veracrypt, luks, etc.)
			if strings.Contains(fields[0], "mapper/") && strings.Contains(line, filepath.Base(devPath)) {
				return true, fields[1] + " (encrypted)"
			}
		}
	}
	return false, "(not mounted)"
}

func (dm *linuxDeviceManager) getEncryptedDeviceInfo(partPath string) (string, string) {
	// Check if this partition has an encrypted mapper device (pure Go approach)
	mapperDevs, err := filepath.Glob("/dev/mapper/*")
	if err != nil {
		return "", ""
	}
	
	partBaseName := filepath.Base(partPath)
	
	for _, mapperDev := range mapperDevs {
		// Check sysfs to find parent device
		mapperName := filepath.Base(mapperDev)
		slavesDir := fmt.Sprintf("/sys/block/dm-%s/slaves", dm.getDMNumber(mapperName))
		
		slaves, err := os.ReadDir(slavesDir)
		if err != nil {
			continue
		}
		
		for _, slave := range slaves {
			if slave.Name() == partBaseName {
				// Found encrypted device based on this partition
				uuid := dm.getDeviceUUID(mapperDev)
				mounted, mountPoint := dm.getMountInfo(mapperDev)
				if mounted {
					mountPoint += " (encrypted)"
				}
				return uuid, mountPoint
			}
		}
	}
	return "", ""
}

func (dm *linuxDeviceManager) getDMNumber(mapperName string) string {
	// Get device mapper number from sysfs
	majorMinorPath := "/sys/class/block/" + mapperName + "/dev"
	if data, err := os.ReadFile(majorMinorPath); err == nil {
		majorMinor := strings.TrimSpace(string(data))
		parts := strings.Split(majorMinor, ":")
		if len(parts) == 2 {
			return parts[1] // Return minor number
		}
	}
	return "0"
}