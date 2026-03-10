package tui

import "github.com/crateos/crateos/internal/sysinfo"

// ── View state ──────────────────────────────────────────────────────

type viewID int

const (
	viewMenu viewID = iota
	viewSetup
	viewStatus
	viewDiagnostics
	viewServices
	viewUsers
	viewLogs
	viewNetwork
)

// ── Service info ────────────────────────────────────────────────────

// ServiceInfo describes a managed service and its runtime status.
type ServiceInfo struct {
	Name                       string        `json:"name"`
	DisplayName                string        `json:"display_name"`
	Status                     string        `json:"status"`  // "active", "inactive", "failed", "unknown"
	Type                       string        `json:"runtime"` // "systemd" or "docker"
	ActorName                  string        `json:"actor_name"`
	ActorType                  string        `json:"actor_type"`
	ActorID                    string        `json:"actor_id"`
	ActorUser                  string        `json:"actor_user"`
	ActorGroup                 string        `json:"actor_group"`
	ActorHome                  string        `json:"actor_home"`
	ActorRuntimeDir            string        `json:"actor_runtime_dir"`
	ActorStateDir              string        `json:"actor_state_dir"`
	ActorProvisioning          string        `json:"actor_provisioning"`
	ActorProvisioningError     string        `json:"actor_provisioning_error"`
	ActorProvisioningUpdatedAt string        `json:"actor_provisioning_updated_at"`
	ActorProvisioningStatePath string        `json:"actor_provisioning_state_path"`
	ActorOwnershipStatus       string        `json:"actor_ownership_status"`
	ActorOwnershipUpdatedAt    string        `json:"actor_ownership_updated_at"`
	ActorOwnershipRetiredAt    string        `json:"actor_ownership_retired_at"`
	DeploySource               string        `json:"deploy_source"`
	UploadPath                 string        `json:"upload_path"`
	WorkingDir                 string        `json:"working_dir"`
	Entry                      string        `json:"entry"`
	InstallCommand             string        `json:"install_command"`
	EnvironmentFile            string        `json:"environment_file"`
	ExecutionMode              string        `json:"execution_mode"`
	ExecutionAdapter           string        `json:"execution_adapter"`
	ExecutionStatus            string        `json:"execution_status"`
	PrimaryUnit                string        `json:"primary_unit"`
	CompanionUnit              string        `json:"companion_unit"`
	PrimaryUnitPath            string        `json:"primary_unit_path"`
	CompanionUnitPath          string        `json:"companion_unit_path"`
	ExecutionStatePath         string        `json:"execution_state_path"`
	StartCommand               string        `json:"start_command"`
	Schedule                   string        `json:"schedule"`
	Timeout                    string        `json:"timeout"`
	StopTimeout                string        `json:"stop_timeout"`
	OnTimeout                  string        `json:"on_timeout"`
	KillSignal                 string        `json:"kill_signal"`
	ConcurrencyPolicy          string        `json:"concurrency_policy"`
	ExecutionSummary           string        `json:"execution_summary"`
	Stateful                   bool          `json:"stateful"`
	DataPath                   string        `json:"data_path"`
	NativeDataPath             string        `json:"native_data_path"`
	StorageSummary             string        `json:"storage_summary"`
	Health                     string        `json:"health"`
	Desired                    bool          `json:"desired"`
	Autostart                  bool          `json:"autostart"`
	Active                     bool          `json:"active"`
	Enabled                    bool          `json:"enabled"`
	Module                     bool          `json:"module"`
	Ready                      bool          `json:"ready"`
	PackagesInstalled          bool          `json:"packages_installed"`
	MissingPackages            []string      `json:"missing_packages"`
	Summary                    string        `json:"summary"`
	LastError                  string        `json:"last_error"`
	LastAction                 string        `json:"last_action"`
	LastActionAt               string        `json:"last_action_at"`
	SuggestedRepair            string        `json:"suggested_repair"`
	LastGoodStatus             string        `json:"last_good_status"`
	LastGoodHealth             string        `json:"last_good_health"`
	LastGoodAt                 string        `json:"last_good_at"`
	LastGoodSummary            string        `json:"last_good_summary"`
	Category                   string        `json:"category"`
	Units                      []ServiceUnit `json:"units"`
}

type ServiceUnit struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Health  string `json:"health"`
}

type userRow struct {
	Name  string
	Role  string
	Perms []string
}

// ── Model ───────────────────────────────────────────────────────────

type model struct {
	currentView        viewID
	cursor             int
	logSourceCursor    int
	statusSection      int
	ownershipCursor    int
	width              int
	height             int
	info               sysinfo.Info
	interfaces         []sysinfo.NetIface
	services           []ServiceInfo
	platform           PlatformInfo
	diagnostics        DiagnosticsInfo
	users              []userRow
	currentUser        string
	newUserRole        string
	setupAdmin         string
	userFormOpen       bool
	userFormEdit       bool
	userFormField      int
	userFormTarget     string
	userFormName       string
	userFormRole       string
	userFormPerms      string
	commandMode        bool
	commandInput       string
	commandStatus      string
	commandStatusLevel string
	controlPlaneOnline bool
	quitting           bool
}

var menuItems = []string{
	"System Status",
	"Services",
	"Diagnostics",
	"Users",
	"Logs",
	"Network",
	"Exit",
}
