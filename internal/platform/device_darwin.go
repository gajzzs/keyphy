//go:build darwin
// +build darwin

package platform

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type macDeviceManager struct{}

func newDeviceManager() DeviceManager {
	return &macDeviceManager{}
}

type diskutilDevice struct {
	DeviceIdentifier string `json:"DeviceIdentifier"`
	VolumeName       string `json:"VolumeName"`
	VolumeUUID       string `json:"VolumeUUID"`
	MountPoint       string `json:"MountPoint"`
}

type diskutilOutput struct {
	AllDisksAndPartitions []struct {
		Partitions []diskutilDevice `json:"Partitions"`
	} `json:"AllDisksAndPartitions"`
}

func (dm *macDeviceManager) ListUSBDevices() ([]Device, error) {
	// Use diskutil to list devices
	cmd := exec.Command("diskutil", "list", "-plist", "external")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	var diskutil diskutilOutput
	if err := json.Unmarshal(output, &diskutil); err != nil {
		return nil, err
	}
	
	var devices []Device
	for _, disk := range diskutil.AllDisksAndPartitions {
		for _, partition := range disk.Partitions {
			if partition.VolumeUUID != "" {
				mounted := partition.MountPoint != ""
				devices = append(devices, Device{
					UUID:       partition.VolumeUUID,
					Name:       partition.VolumeName,
					Path:       "/dev/" + partition.DeviceIdentifier,
					Mounted:    mounted,
					MountPoint: partition.MountPoint,
				})
			}
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