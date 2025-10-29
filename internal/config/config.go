package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	BlockedApps     []string `json:"blocked_apps"`
	BlockedWebsites []string `json:"blocked_websites"`
	BlockedPaths    []string `json:"blocked_paths"`
	AuthDevice      string   `json:"auth_device"`
	AuthKey         string   `json:"auth_key"`
}

var (
	ConfigDir  = "/etc/keyphy"
	ConfigFile = filepath.Join(ConfigDir, "config.json")
	config     *Config
)

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
		data, err := os.ReadFile(ConfigFile)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, config)
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
	if err := os.WriteFile(ConfigFile, data, 0644); err != nil {
		return err
	}
	fmt.Println("Configuration saved successfully")
	return nil
}

func AddBlockedApp(app string) error {
	config.BlockedApps = append(config.BlockedApps, app)
	fmt.Printf("Added '%s' to blocked applications in config\n", app)
	return SaveConfig()
}

func AddBlockedWebsite(website string) error {
	config.BlockedWebsites = append(config.BlockedWebsites, website)
	fmt.Printf("Added '%s' to blocked websites in config\n", website)
	return SaveConfig()
}

func AddBlockedPath(path string) error {
	config.BlockedPaths = append(config.BlockedPaths, path)
	fmt.Printf("Added '%s' to blocked paths in config\n", path)
	return SaveConfig()
}

func RemoveBlocked(item string) error {
	config.BlockedApps = removeFromSlice(config.BlockedApps, item)
	config.BlockedWebsites = removeFromSlice(config.BlockedWebsites, item)
	config.BlockedPaths = removeFromSlice(config.BlockedPaths, item)
	return SaveConfig()
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}