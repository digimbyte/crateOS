package sysinfo

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Info holds basic system information.
type Info struct {
	Hostname  string
	OS        string
	Arch      string
	CPUs      int
	GoVersion string
	Time      time.Time
}

// Gather collects system information using stdlib (cross-platform).
func Gather() Info {
	hostname, _ := os.Hostname()
	return Info{
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		GoVersion: runtime.Version(),
		Time:      time.Now(),
	}
}

// NetIface represents a network interface with its addresses.
type NetIface struct {
	Name  string
	MAC   string
	Flags net.Flags
	Addrs []string
	Up    bool
}

// NetworkInterfaces returns all network interfaces and their addresses.
func NetworkInterfaces() []NetIface {
	var result []NetIface
	ifaces, err := net.Interfaces()
	if err != nil {
		return result
	}
	for _, iface := range ifaces {
		ni := NetIface{
			Name:  iface.Name,
			MAC:   iface.HardwareAddr.String(),
			Flags: iface.Flags,
			Up:    iface.Flags&net.FlagUp != 0,
		}
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ni.Addrs = append(ni.Addrs, addr.String())
			}
		}
		result = append(result, ni)
	}
	return result
}

type StorageDevice struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Mountpoint string `json:"mountpoint"`
	FSType     string `json:"fs_type"`
	Size       string `json:"size"`
	Removable  bool   `json:"removable"`
	Rotational bool   `json:"rotational"`
}

func StorageDevices() []StorageDevice {
	if runtime.GOOS != "linux" {
		return nil
	}
	out, err := exec.Command("lsblk", "-J", "-o", "NAME,PATH,TYPE,MOUNTPOINT,FSTYPE,SIZE,RM,ROTA").Output()
	if err != nil {
		return nil
	}
	var payload struct {
		Blockdevices []struct {
			Name       string `json:"name"`
			Path       string `json:"path"`
			Type       string `json:"type"`
			Mountpoint string `json:"mountpoint"`
			FSType     string `json:"fstype"`
			Size       string `json:"size"`
			RM         bool   `json:"rm"`
			ROTA       bool   `json:"rota"`
			Children   []struct {
				Name       string `json:"name"`
				Path       string `json:"path"`
				Type       string `json:"type"`
				Mountpoint string `json:"mountpoint"`
				FSType     string `json:"fstype"`
				Size       string `json:"size"`
				RM         bool   `json:"rm"`
				ROTA       bool   `json:"rota"`
			} `json:"children"`
		} `json:"blockdevices"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil
	}
	devices := make([]StorageDevice, 0)
	for _, dev := range payload.Blockdevices {
		devices = append(devices, StorageDevice{
			Name:       strings.TrimSpace(dev.Name),
			Path:       strings.TrimSpace(dev.Path),
			Type:       strings.TrimSpace(dev.Type),
			Mountpoint: strings.TrimSpace(dev.Mountpoint),
			FSType:     strings.TrimSpace(dev.FSType),
			Size:       strings.TrimSpace(dev.Size),
			Removable:  dev.RM,
			Rotational: dev.ROTA,
		})
		for _, child := range dev.Children {
			devices = append(devices, StorageDevice{
				Name:       strings.TrimSpace(child.Name),
				Path:       strings.TrimSpace(child.Path),
				Type:       strings.TrimSpace(child.Type),
				Mountpoint: strings.TrimSpace(child.Mountpoint),
				FSType:     strings.TrimSpace(child.FSType),
				Size:       strings.TrimSpace(child.Size),
				Removable:  child.RM,
				Rotational: child.ROTA,
			})
		}
	}
	return devices
}
