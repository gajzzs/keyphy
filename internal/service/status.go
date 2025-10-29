package service

import (
	"fmt"
	"os"
	"os/exec"
)

func GetDaemonStatus() (bool, error) {
	// Check if daemon process is running
	cmd := exec.Command("pgrep", "-f", "keyphy service start")
	err := cmd.Run()
	return err == nil, nil
}

func GetServiceStatus() string {
	// Check systemd service status
	cmd := exec.Command("systemctl", "is-active", "keyphy")
	output, err := cmd.Output()
	if err != nil {
		return "inactive"
	}
	return string(output)
}

func CreatePidFile() error {
	pidFile := "/var/run/keyphy.pid"
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

func RemovePidFile() error {
	return os.Remove("/var/run/keyphy.pid")
}