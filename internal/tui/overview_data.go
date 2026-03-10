package tui

import (
	"github.com/crateos/crateos/internal/sysinfo"
)

type PlatformInfo struct {
	GeneratedAt string            `json:"generated_at"`
	Adapters    []PlatformAdapter `json:"adapters"`
}

type DiagnosticsInfo struct {
	Config       ConfigDiagnosticsInfo       `json:"config"`
	Verification VerificationDiagnosticsInfo `json:"verification"`
	Ownership    OwnershipDiagnosticsInfo    `json:"ownership"`
}

type ConfigDiagnosticsInfo struct {
	GeneratedAt   string                 `json:"generated_at"`
	Tracked       int                    `json:"tracked"`
	Monitored     int                    `json:"monitored"`
	Unmonitored   int                    `json:"unmonitored"`
	ExternalEdits int                    `json:"external_edits"`
	Files         []ConfigDiagnosticFile `json:"files"`
}

type ConfigDiagnosticFile struct {
	File          string `json:"file"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Monitoring    string `json:"monitoring"`
	LastWriter    string `json:"last_writer"`
	LastSeenAt    string `json:"last_seen_at"`
	LastChangedAt string `json:"last_changed_at"`
}

type VerificationDiagnosticsInfo struct {
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	Missing        []string `json:"missing"`
	Warnings       []string `json:"warnings"`
	PlatformState  string   `json:"platform_state"`
	Readiness      string   `json:"readiness"`
	StorageState   string   `json:"storage_state"`
	OwnershipState string   `json:"ownership_state"`
	AgentSocket    bool     `json:"agent_socket"`
	AdminPresent   bool     `json:"admin_present"`
}

type OwnershipDiagnosticsInfo struct {
	GeneratedAt string                     `json:"generated_at"`
	Managed     int                        `json:"managed"`
	Provisioned int                        `json:"provisioned"`
	Pending     int                        `json:"pending"`
	Blocked     int                        `json:"blocked"`
	Active      int                        `json:"active"`
	Retired     int                        `json:"retired"`
	Claims      []OwnershipDiagnosticClaim `json:"claims"`
	Workloads   []ActorLifecycleDiagnostic `json:"workloads"`
}

type OwnershipDiagnosticClaim struct {
	Crate     string `json:"crate"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	User      string `json:"user"`
	Group     string `json:"group"`
	Home      string `json:"home"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
	RetiredAt string `json:"retired_at"`
}

type ActorLifecycleDiagnostic struct {
	Crate                 string                          `json:"crate"`
	ActorName             string                          `json:"actor_name"`
	ActorType             string                          `json:"actor_type"`
	ActorID               string                          `json:"actor_id"`
	ActorUser             string                          `json:"actor_user"`
	ActorGroup            string                          `json:"actor_group"`
	ActorHome             string                          `json:"actor_home"`
	Provisioning          string                          `json:"provisioning"`
	ProvisioningError     string                          `json:"provisioning_error"`
	ProvisioningUpdatedAt string                          `json:"provisioning_updated_at"`
	LastSuccessAt         string                          `json:"last_success_at"`
	LastFailureAt         string                          `json:"last_failure_at"`
	ProvisioningStatePath string                          `json:"provisioning_state_path"`
	OwnershipStatus       string                          `json:"ownership_status"`
	OwnershipUpdatedAt    string                          `json:"ownership_updated_at"`
	OwnershipRetiredAt    string                          `json:"ownership_retired_at"`
	RecentEvents          []ActorLifecycleEventDiagnostic `json:"recent_events"`
}

type ActorLifecycleEventDiagnostic struct {
	At           string `json:"at"`
	Provisioning string `json:"provisioning"`
	Error        string `json:"error"`
}

type PlatformAdapter struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Enabled       bool     `json:"enabled"`
	Status        string   `json:"status"`
	Health        string   `json:"health"`
	Summary       string   `json:"summary"`
	LastError     string   `json:"last_error"`
	Validation    string   `json:"validation"`
	ValidationErr string   `json:"validation_error"`
	Apply         string   `json:"apply"`
	ApplyErr      string   `json:"apply_error"`
	RenderedPaths []string `json:"rendered_paths"`
	NativeTargets []string `json:"native_targets"`
}

func (m *model) refreshOverview() {
	if info, svcs, platformInfo, diagnostics, _ := fetchStatusViaAPI(m.currentUser); info != nil {
		m.info = *info
		m.services = svcs
		m.platform = platformInfo
		m.diagnostics = diagnostics
		m.controlPlaneOnline = true
		return
	}
	m.refreshServices()
	m.platform = readFallbackPlatformState()
	m.diagnostics = readFallbackDiagnostics()
	m.info = sysinfo.Gather()
	m.controlPlaneOnline = false
}
