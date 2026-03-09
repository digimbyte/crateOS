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
	CrateRootExists bool               `json:"crate_root_exists"`
	InstalledMarker bool               `json:"installed_marker"`
	Directories     map[string]bool    `json:"directories"`
	Services        []ServiceState     `json:"services"`
	Network         []sysinfo.NetIface `json:"network"`
	Hostname        string             `json:"hostname"`
}

// ServiceState captures the runtime state of a single service.
type ServiceState struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"` // "active", "inactive", "failed", "unknown"
	Health  string `json:"health"` // "ok", "degraded", "unknown"
}

type CrateState struct {
	Name               string         `json:"name"`
	DisplayName        string         `json:"display_name"`
	Category           string         `json:"category,omitempty"`
	Description        string         `json:"description,omitempty"`
	Runtime            string         `json:"runtime"`
	InstallMode        string         `json:"install_mode,omitempty"`
	ActorName          string         `json:"actor_name,omitempty"`
	ActorType          string         `json:"actor_type,omitempty"`
	ActorID            string         `json:"actor_id,omitempty"`
	ActorUser          string         `json:"actor_user,omitempty"`
	ActorGroup         string         `json:"actor_group,omitempty"`
	ActorHome          string         `json:"actor_home,omitempty"`
	ActorRuntimeDir    string         `json:"actor_runtime_dir,omitempty"`
	ActorStateDir      string         `json:"actor_state_dir,omitempty"`
	ActorProvisioning  string         `json:"actor_provisioning,omitempty"`
	ActorProvisioningError string     `json:"actor_provisioning_error,omitempty"`
	ActorProvisioningUpdatedAt string `json:"actor_provisioning_updated_at,omitempty"`
	ActorProvisioningStatePath string `json:"actor_provisioning_state_path,omitempty"`
	ActorOwnershipStatus string       `json:"actor_ownership_status,omitempty"`
	ActorOwnershipUpdatedAt string    `json:"actor_ownership_updated_at,omitempty"`
	ActorOwnershipRetiredAt string    `json:"actor_ownership_retired_at,omitempty"`
	DeploySource       string         `json:"deploy_source,omitempty"`
	UploadPath         string         `json:"upload_path,omitempty"`
	WorkingDir         string         `json:"working_dir,omitempty"`
	Entry              string         `json:"entry,omitempty"`
	InstallCommand     string         `json:"install_command,omitempty"`
	EnvironmentFile    string         `json:"environment_file,omitempty"`
	ExecutionMode      string         `json:"execution_mode,omitempty"`
	ExecutionAdapter   string         `json:"execution_adapter,omitempty"`
	ExecutionStatus    string         `json:"execution_status,omitempty"`
	PrimaryUnit        string         `json:"primary_unit,omitempty"`
	CompanionUnit      string         `json:"companion_unit,omitempty"`
	PrimaryUnitPath    string         `json:"primary_unit_path,omitempty"`
	CompanionUnitPath  string         `json:"companion_unit_path,omitempty"`
	ExecutionStatePath string         `json:"execution_state_path,omitempty"`
	StartCommand       string         `json:"start_command,omitempty"`
	Schedule           string         `json:"schedule,omitempty"`
	Timeout            string         `json:"timeout,omitempty"`
	StopTimeout        string         `json:"stop_timeout,omitempty"`
	OnTimeout          string         `json:"on_timeout,omitempty"`
	KillSignal         string         `json:"kill_signal,omitempty"`
	ConcurrencyPolicy  string         `json:"concurrency_policy,omitempty"`
	ExecutionSummary   string         `json:"execution_summary,omitempty"`
	Stateful           bool           `json:"stateful,omitempty"`
	DataPath           string         `json:"data_path,omitempty"`
	NativeDataPath     string         `json:"native_data_path,omitempty"`
	StorageSummary     string         `json:"storage_summary,omitempty"`
	Desired            bool           `json:"desired"`
	Autostart          bool           `json:"autostart"`
	Ready              bool           `json:"ready"`
	Module             bool           `json:"module"`
	Health             string         `json:"health"`
	Status             string         `json:"status"`
	PackagesInstalled  bool           `json:"packages_installed"`
	MissingPackages    []string       `json:"missing_packages,omitempty"`
	Units              []ServiceState `json:"units,omitempty"`
	UnitNames          []string       `json:"unit_names,omitempty"`
	HealthChecks       []string       `json:"health_checks,omitempty"`
	Summary            string         `json:"summary,omitempty"`
	LastError          string         `json:"last_error,omitempty"`
	LastAction         string         `json:"last_action,omitempty"`
	LastActionAt       string         `json:"last_action_at,omitempty"`
	SuggestedRepair    string         `json:"suggested_repair,omitempty"`
}

type StoredCrateState struct {
	GeneratedAt string     `json:"generated_at"`
	Crate       CrateState `json:"crate"`
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
	ss := ServiceState{Name: name, Status: "unknown", Health: "unknown"}

	if runtime.GOOS != "linux" {
		return ss
	}

	// Check if active
	out, err := exec.Command("systemctl", "is-active", name).Output()
	if err == nil {
		ss.Status = strings.TrimSpace(string(out))
		ss.Active = ss.Status == "active"
		if ss.Active {
			ss.Health = "ok"
		}
	} else {
		ss.Status = "inactive"
		ss.Health = "degraded"
	}

	// Check if enabled
	out, err = exec.Command("systemctl", "is-enabled", name).Output()
	if err == nil {
		ss.Enabled = strings.TrimSpace(string(out)) == "enabled"
	}

	return ss
}
