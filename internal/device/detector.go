package device

import (
	"github.com/gajzzs/keyphy/internal/platform"
)

type Device struct {
	UUID       string
	Name       string
	MountPoint string
	DevPath    string
}

func ListUSBDevices() ([]Device, error) {
	deviceManager := platform.NewDeviceManager()
	platformDevices, err := deviceManager.ListUSBDevices()
	if err != nil {
		return nil, err
	}
	
	// Convert platform.Device to device.Device
	var devices []Device
	for _, pd := range platformDevices {
		devices = append(devices, Device{
			UUID:       pd.UUID,
			Name:       pd.Name,
			MountPoint: pd.MountPoint,
			DevPath:    pd.Path,
		})
	}
	return devices, nil
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