package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Config struct {
	BlockedApps     []string `json:"blocked_apps"`
	BlockedWebsites []string `json:"blocked_websites"`
	BlockedPaths    []string `json:"blocked_paths"`
	AuthDevice      string   `json:"auth_device"`
	AuthKey         string   `json:"auth_key"`
	AuthDeviceName  string   `json:"auth_device_name"`
	AuthMountState  string   `json:"auth_mount_state"`
	EnforceState    bool     `json:"enforce_state"`
}

var (
	ConfigDir  = getUniqueConfigDir()
	ConfigFile = filepath.Join(ConfigDir, "config.json")
	config     *Config
)

func getUniqueConfigDir() string {
	baseDir := "/etc/keyphy"
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return baseDir
	}
	
	// Check if it's a file, not directory
	if info, err := os.Stat(baseDir); err == nil && !info.IsDir() {
		// File exists, create unique directory name
		for i := 1; i < 100; i++ {
			uniqueName := fmt.Sprintf("%s_%d", baseDir, i)
			if _, err := os.Stat(uniqueName); os.IsNotExist(err) {
				return uniqueName
			}
		}
	}
	return baseDir
}

func InitConfig() error {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return err
	}

	config = &Config{
		BlockedApps:     []string{},
		BlockedWebsites: []string{},
		BlockedPaths:    []string{},
	}

	if _, err := os.Stat(ConfigFile); err == nil {
		// Temporarily remove protection to read
		UnprotectConfigFile()
		data, err := os.ReadFile(ConfigFile)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, config); err != nil {
			fmt.Println("Warning: Config file corrupted, creating new one")
			return SaveConfig()
		}
		// Restore protection
		ProtectConfigFile()
		return nil
	} else {
		fmt.Println("Creating keyphy configuration file...")
	}

	return SaveConfig()
}

func GetConfig() *Config {
	return config
}

func SaveConfig() error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	UnprotectConfigFile()
	// Use more restrictive permissions (root only)
	if err := os.WriteFile(ConfigFile, data, 0600); err != nil {
		return err
	}
	// Make file immutable to prevent tampering
	ProtectConfigFile()
	fmt.Println("Configuration saved successfully")
	return nil
}

func AddBlockedApp(app string) error {
	UnprotectConfigFile()
	// Check for duplicates
	for _, existing := range config.BlockedApps {
		if existing == app {
			fmt.Printf("'%s' is already in blocked applications list\n", app)
			return nil
		}
	}
	config.BlockedApps = append(config.BlockedApps, app)
	fmt.Printf("Added '%s' to blocked applications in config\n", app)
	return SaveConfig()
}

func AddBlockedAppWithPath(app, path string) error {
	UnprotectConfigFile()
	appEntry := fmt.Sprintf("%s:%s", app, path)
	// Check for duplicates
	for _, existing := range config.BlockedApps {
		if existing == appEntry || existing == path {
			fmt.Printf("'%s' is already in blocked applications list\n", appEntry)
			return nil
		}
	}
	config.BlockedApps = append(config.BlockedApps, appEntry)
	fmt.Printf("Added '%s' with path '%s' to blocked applications in config\n", app, path)
	return SaveConfig()
}

func AddBlockedWebsite(website string) error {
	UnprotectConfigFile()
	config.BlockedWebsites = append(config.BlockedWebsites, website)
	fmt.Printf("Added '%s' to blocked websites in config\n", website)
	return SaveConfig()
}

func AddBlockedPath(path string) error {
	UnprotectConfigFile()
	config.BlockedPaths = append(config.BlockedPaths, path)
	fmt.Printf("Added '%s' to blocked paths in config\n", path)
	return SaveConfig()
}

func RemoveBlocked(item string) error {
	UnprotectConfigFile()
	config.BlockedApps = removeFromSlice(config.BlockedApps, item)
	config.BlockedWebsites = removeFromSlice(config.BlockedWebsites, item)
	config.BlockedPaths = removeFromSlice(config.BlockedPaths, item)
	return SaveConfig()
}

func CleanDuplicates() error {
	UnprotectConfigFile()
	config.BlockedApps = removeDuplicates(config.BlockedApps)
	config.BlockedWebsites = removeDuplicates(config.BlockedWebsites)
	config.BlockedPaths = removeDuplicates(config.BlockedPaths)
	fmt.Println("Removed duplicate entries from config")
	return SaveConfig()
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func ProtectConfigFile() {
	// Make config file immutable using syscall
	file, err := os.Open(ConfigFile)
	if err != nil {
		return
	}
	defer file.Close()
	
	const FS_IOC_SETFLAGS = 0x40086602
	const FS_IMMUTABLE_FL = 0x00000010
	
	syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(FS_IOC_SETFLAGS),
		uintptr(FS_IMMUTABLE_FL))
}

func UnprotectConfigFile() {
	// Remove immutable flag using syscall
	file, err := os.Open(ConfigFile)
	if err != nil {
		return
	}
	defer file.Close()
	
	const FS_IOC_SETFLAGS = 0x40086602
	
	syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(FS_IOC_SETFLAGS),
		uintptr(0x00000000))
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}