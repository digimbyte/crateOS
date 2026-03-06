package sysinfo

import (
	"net"
	"os"
	"runtime"
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
