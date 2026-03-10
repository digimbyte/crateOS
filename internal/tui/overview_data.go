package tui

import (
	"os"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
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

func readFallbackPlatformState() PlatformInfo {
	snapshot := state.LoadPlatformState()
	info := PlatformInfo{
		GeneratedAt: snapshot.GeneratedAt,
		Adapters:    make([]PlatformAdapter, 0, len(snapshot.Adapters)),
	}
	for _, adapter := range snapshot.Adapters {
		info.Adapters = append(info.Adapters, PlatformAdapter{
			Name:          adapter.Name,
			DisplayName:   adapter.DisplayName,
			Enabled:       adapter.Enabled,
			Status:        adapter.Status,
			Health:        adapter.Health,
			Summary:       adapter.Summary,
			LastError:     adapter.LastError,
			Validation:    adapter.Validation,
			ValidationErr: adapter.ValidationErr,
			Apply:         adapter.Apply,
			ApplyErr:      adapter.ApplyErr,
			RenderedPaths: append([]string(nil), adapter.RenderedPaths...),
			NativeTargets: append([]string(nil), adapter.NativeTargets...),
		})
	}
	return info
}

func readFallbackOwnershipDiagnostics() OwnershipDiagnosticsInfo {
	snapshot := state.LoadActorOwnershipState()
	info := OwnershipDiagnosticsInfo{
		GeneratedAt: strings.TrimSpace(snapshot.GeneratedAt),
		Active:      snapshot.Active,
		Retired:     snapshot.Retired,
		Claims:      make([]OwnershipDiagnosticClaim, 0, len(snapshot.Claims)),
		Workloads:   []ActorLifecycleDiagnostic{},
	}
	claimsByCrate := map[string]state.ActorOwnershipStateItem{}
	for _, claim := range snapshot.Claims {
		claimsByCrate[strings.TrimSpace(claim.Crate)] = claim
		info.Claims = append(info.Claims, OwnershipDiagnosticClaim{
			Crate:     claim.Crate,
			Name:      claim.Name,
			Type:      claim.Type,
			ID:        claim.ID,
			User:      claim.User,
			Group:     claim.Group,
			Home:      claim.Home,
			Status:    claim.Status,
			UpdatedAt: claim.UpdatedAt,
			RetiredAt: claim.RetiredAt,
		})
	}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		for _, svc := range cfg.Services.Services {
			if strings.TrimSpace(svc.Actor.Name) == "" && strings.TrimSpace(svc.Execution.Mode) == "" {
				continue
			}
			info.Managed++
			provisioning := state.LoadActorProvisioningState(svc.Name)
			item := ActorLifecycleDiagnostic{
				Crate:                 svc.Name,
				ActorName:             strings.TrimSpace(provisioning.Actor.Name),
				ActorType:             strings.TrimSpace(provisioning.Actor.Type),
				ActorID:               strings.TrimSpace(provisioning.Actor.ID),
				ActorUser:             strings.TrimSpace(provisioning.Actor.User),
				ActorGroup:            strings.TrimSpace(provisioning.Actor.Group),
				ActorHome:             strings.TrimSpace(provisioning.Actor.Home),
				Provisioning:          strings.TrimSpace(provisioning.Provisioning),
				ProvisioningError:     strings.TrimSpace(provisioning.Error),
				ProvisioningUpdatedAt: strings.TrimSpace(provisioning.GeneratedAt),
				LastSuccessAt:         strings.TrimSpace(provisioning.LastSuccessAt),
				LastFailureAt:         strings.TrimSpace(provisioning.LastFailureAt),
				ProvisioningStatePath: platform.CratePath("services", svc.Name, "runtime", "actor-provisioning.json"),
				RecentEvents:          make([]ActorLifecycleEventDiagnostic, 0, len(provisioning.Events)),
			}
			for _, event := range provisioning.Events {
				item.RecentEvents = append(item.RecentEvents, ActorLifecycleEventDiagnostic{
					At:           strings.TrimSpace(event.At),
					Provisioning: strings.TrimSpace(event.Provisioning),
					Error:        strings.TrimSpace(event.Error),
				})
			}
			if item.ActorName == "" {
				item.ActorName = strings.TrimSpace(svc.Actor.Name)
			}
			if claim, ok := claimsByCrate[strings.TrimSpace(svc.Name)]; ok {
				item.OwnershipStatus = strings.TrimSpace(claim.Status)
				item.OwnershipUpdatedAt = strings.TrimSpace(claim.UpdatedAt)
				item.OwnershipRetiredAt = strings.TrimSpace(claim.RetiredAt)
				if item.ActorName == "" {
					item.ActorName = strings.TrimSpace(claim.Name)
				}
				if item.ActorType == "" {
					item.ActorType = strings.TrimSpace(claim.Type)
				}
				if item.ActorID == "" {
					item.ActorID = strings.TrimSpace(claim.ID)
				}
				if item.ActorUser == "" {
					item.ActorUser = strings.TrimSpace(claim.User)
				}
				if item.ActorGroup == "" {
					item.ActorGroup = strings.TrimSpace(claim.Group)
				}
				if item.ActorHome == "" {
					item.ActorHome = strings.TrimSpace(claim.Home)
				}
			}
			switch item.Provisioning {
			case "provisioned":
				info.Provisioned++
			case "blocked":
				info.Blocked++
			default:
				info.Pending++
			}
			info.Workloads = append(info.Workloads, item)
		}
	}
	return info
}

func readFallbackVerificationDiagnostics() VerificationDiagnosticsInfo {
	info := VerificationDiagnosticsInfo{
		Status:   "ready",
		Missing:  []string{},
		Warnings: []string{},
	}
	requiredPaths := []struct {
		path  string
		label string
	}{
		{platform.CratePath("state", "installed.json"), "installed marker"},
		{platform.CratePath("state", "platform-state.json"), "platform state"},
		{platform.CratePath("state", "readiness-report.json"), "readiness report"},
		{platform.CratePath("state", "storage-state.json"), "storage state"},
		{platform.CratePath("state", "actor-ownership-state.json"), "actor ownership state"},
	}
	for _, item := range requiredPaths {
		if _, err := os.Stat(item.path); err != nil {
			info.Missing = append(info.Missing, item.label)
		}
	}
	if _, err := os.Stat(platform.AgentSocket); err == nil {
		info.AgentSocket = true
	}
	if rows := fetchUsersFromConfig(); len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(strings.TrimSpace(row.Role), "admin") {
				info.AdminPresent = true
				break
			}
		}
	}
	info.PlatformState = strings.TrimSpace(readFallbackPlatformState().GeneratedAt)
	if info.PlatformState == "" {
		info.Warnings = append(info.Warnings, "platform state not rendered yet")
	}
	if report, ok := readReadinessReport(); ok {
		info.Readiness = strings.TrimSpace(report.Status)
		if info.Readiness == "" {
			info.Readiness = "unknown"
		}
		if info.Readiness != "ready" {
			info.Warnings = append(info.Warnings, "readiness report is not ready")
		}
	} else {
		info.Warnings = append(info.Warnings, "readiness report unreadable")
	}
	if storage := state.LoadStorageState(); strings.TrimSpace(storage.GeneratedAt) != "" {
		info.StorageState = strings.TrimSpace(storage.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "storage posture not rendered yet")
	}
	if ownership := state.LoadActorOwnershipState(); strings.TrimSpace(ownership.GeneratedAt) != "" {
		info.OwnershipState = strings.TrimSpace(ownership.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "actor ownership state not rendered yet")
	}
	if !info.AgentSocket {
		info.Warnings = append(info.Warnings, "agent socket unavailable")
	}
	if !info.AdminPresent {
		info.Missing = append(info.Missing, "admin operator")
	}
	switch {
	case len(info.Missing) > 0:
		info.Status = "failed"
		info.Summary = "verification prerequisites missing"
	case len(info.Warnings) > 0:
		info.Status = "degraded"
		info.Summary = "verification surfaces present with warnings"
	default:
		info.Summary = "verification surfaces present"
	}
	return info
}

func readFallbackDiagnostics() DiagnosticsInfo {
	ledger, err := config.LoadConfigChangeLedger()
	if err != nil {
		return DiagnosticsInfo{
			Verification: readFallbackVerificationDiagnostics(),
			Ownership:    readFallbackOwnershipDiagnostics(),
		}
	}
	info := DiagnosticsInfo{
		Config: ConfigDiagnosticsInfo{
			GeneratedAt: ledger.GeneratedAt,
			Files:       make([]ConfigDiagnosticFile, 0, len(ledger.Files)),
		},
		Verification: readFallbackVerificationDiagnostics(),
		Ownership:    readFallbackOwnershipDiagnostics(),
	}
	for _, record := range ledger.Files {
		info.Config.Tracked++
		switch strings.TrimSpace(record.Monitoring) {
		case "unmonitored":
			info.Config.Unmonitored++
		default:
			info.Config.Monitored++
		}
		if strings.TrimSpace(record.LastWriter) == "external" {
			info.Config.ExternalEdits++
		}
		info.Config.Files = append(info.Config.Files, ConfigDiagnosticFile{
			File:          record.File,
			Path:          record.Path,
			Exists:        record.Exists,
			Monitoring:    record.Monitoring,
			LastWriter:    record.LastWriter,
			LastSeenAt:    record.LastSeenAt,
			LastChangedAt: record.LastChangedAt,
		})
	}
	return info
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
