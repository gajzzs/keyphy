package system

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemMonitor struct {
	processes map[int32]*process.Process
}

func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		processes: make(map[int32]*process.Process),
	}
}

// GetSystemInfo returns comprehensive system information
func (sm *SystemMonitor) GetSystemInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Host info
	if hostInfo, err := host.Info(); err == nil {
		info["hostname"] = hostInfo.Hostname
		info["os"] = hostInfo.OS
		info["platform"] = hostInfo.Platform
		info["uptime"] = hostInfo.Uptime
	}

	// Memory info
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info["memory_total"] = memInfo.Total
		info["memory_used"] = memInfo.Used
		info["memory_percent"] = memInfo.UsedPercent
	}

	// CPU info
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		info["cpu_percent"] = cpuPercent[0]
	}

	// Disk info
	if diskInfo, err := disk.Usage("/"); err == nil {
		info["disk_total"] = diskInfo.Total
		info["disk_used"] = diskInfo.Used
		info["disk_percent"] = diskInfo.UsedPercent
	}

	return info, nil
}

// MonitorProcesses tracks running processes for blocked apps
func (sm *SystemMonitor) MonitorProcesses(blockedApps []string) ([]int32, error) {
	var blockedPids []int32
	
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		for _, blockedApp := range blockedApps {
			if name == blockedApp {
				blockedPids = append(blockedPids, proc.Pid)
			}
		}
	}

	return blockedPids, nil
}

// GetNetworkConnections returns active network connections
func (sm *SystemMonitor) GetNetworkConnections() ([]net.ConnectionStat, error) {
	return net.Connections("inet")
}

// DetectSuspiciousActivity monitors for bypass attempts
func (sm *SystemMonitor) DetectSuspiciousActivity() ([]string, error) {
	var alerts []string
	
	// Check for high CPU usage (potential bypass tools)
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		if cpuPercent[0] > 90 {
			alerts = append(alerts, fmt.Sprintf("High CPU usage detected: %.2f%%", cpuPercent[0]))
		}
	}

	// Check for suspicious processes
	processes, err := process.Processes()
	if err != nil {
		return alerts, err
	}

	suspiciousNames := []string{"chattr", "xattr", "lsattr", "iptables", "pfctl"}
	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		for _, suspicious := range suspiciousNames {
			if name == suspicious {
				alerts = append(alerts, fmt.Sprintf("Suspicious process detected: %s (PID: %d)", name, proc.Pid))
			}
		}
	}

	return alerts, nil
}