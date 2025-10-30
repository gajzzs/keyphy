package platform

// Device represents a USB device
type Device struct {
	UUID       string
	Name       string
	Path       string
	Mounted    bool
	MountPoint string
}

// DeviceManager provides cross-platform USB device management
type DeviceManager interface {
	ListUSBDevices() ([]Device, error)
	GetDeviceInfo(uuid string) (*Device, error)
	IsDeviceMounted(uuid string) (bool, error)
}

// NewDeviceManager creates a platform-specific device manager
func NewDeviceManager() DeviceManager {
	return newDeviceManager()
}