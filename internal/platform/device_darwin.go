//go:build darwin
// +build darwin

package platform

import (
	"os/exec"
	"strings"
	"howett.net/plist"
)

type macDeviceManager struct{}

func newDeviceManager() DeviceManager {
	return &macDeviceManager{}
}

type diskutilDevice struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	VolumeName       string `plist:"VolumeName"`
	VolumeUUID       string `plist:"VolumeUUID"`
	MountPoint       string `plist:"MountPoint"`
}

type diskutilOutput struct {
	AllDisksAndPartitions []struct {
		Partitions []diskutilDevice `plist:"Partitions"`
	} `plist:"AllDisksAndPartitions"`
}

func (dm *macDeviceManager) ListUSBDevices() ([]Device, error) {
	// Use diskutil to list devices
	cmd := exec.Command("diskutil", "list", "-plist", "external")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var diskutil diskutilOutput
	if _, err := plist.Unmarshal(output, &diskutil); err != nil {
		return nil, err
	}
	
	var devices []Device
	for _, disk := range diskutil.AllDisksAndPartitions {
		for _, partition := range disk.Partitions {
			// Consistent UUID strategy: prefer VolumeUUID, fallback to DeviceIdentifier
			uuid := partition.VolumeUUID
			if uuid == "" {
				uuid = "NO-UUID-" + partition.DeviceIdentifier // Consistent with Linux
			}
			
			// Consistent naming: prefer VolumeName, fallback to DeviceIdentifier
			name := partition.VolumeName
			if name == "" {
				name = partition.DeviceIdentifier
			}
			
			mounted := partition.MountPoint != ""
			mountPoint := partition.MountPoint
			if mountPoint == "" {
				mountPoint = "(not mounted)"
			}
			
			devices = append(devices, Device{
				UUID:       uuid,
				Name:       name,
				Path:       "/dev/" + partition.DeviceIdentifier,
				Mounted:    mounted,
				MountPoint: mountPoint,
			})
		}
	}
	
	return devices, nil
}

func (dm *macDeviceManager) GetDeviceInfo(uuid string) (*Device, error) {
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

func (dm *macDeviceManager) IsDeviceMounted(uuid string) (bool, error) {
	cmd := exec.Command("diskutil", "info", uuid)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	
	return strings.Contains(string(output), "Mounted:               Yes"), nil
}