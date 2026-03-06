package state

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
)

// ActualState represents the current observed state of the system.
type ActualState struct {
	CrateRootExists bool              `json:"crate_root_exists"`
	InstalledMarker bool              `json:"installed_marker"`
	Directories     map[string]bool   `json:"directories"`
	Services        []ServiceState    `json:"services"`
	Network         []sysinfo.NetIface `json:"network"`
	Hostname        string            `json:"hostname"`
}

// ServiceState captures the runtime state of a single service.
type ServiceState struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"` // "active", "inactive", "failed", "unknown"
}

// Probe inspects the running system and returns ActualState.
func Probe(serviceNames []string) *ActualState {
	s := &ActualState{
		Directories: make(map[string]bool),
		Hostname:    sysinfo.Gather().Hostname,
		Network:     sysinfo.NetworkInterfaces(),
	}

	// Check crate root
	if info, err := os.Stat(platform.CrateRoot); err == nil && info.IsDir() {
		s.CrateRootExists = true
	}

	// Check installed marker
	marker := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(marker); err == nil {
		s.InstalledMarker = true
	}

	// Check required directories
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		_, err := os.Stat(p)
		s.Directories[d] = err == nil
	}

	// Check services
	for _, name := range serviceNames {
		s.Services = append(s.Services, probeService(name))
	}

	return s
}

func probeService(name string) ServiceState {
	ss := ServiceState{Name: name, Status: "unknown"}

	if runtime.GOOS != "linux" {
		return ss
	}

	// Check if active
	out, err := exec.Command("systemctl", "is-active", name).Output()
	if err == nil {
		ss.Status = strings.TrimSpace(string(out))
		ss.Active = ss.Status == "active"
	} else {
		ss.Status = "inactive"
	}

	// Check if enabled
	out, err = exec.Command("systemctl", "is-enabled", name).Output()
	if err == nil {
		ss.Enabled = strings.TrimSpace(string(out)) == "enabled"
	}

	return ss
}
