package api

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
)

type diagnosticsView struct {
	Config       configDiagnosticsView       `json:"config"`
	Verification verificationDiagnosticsView `json:"verification"`
	Ownership    ownershipDiagnosticsView    `json:"ownership"`
}

type configDiagnosticsView struct {
	GeneratedAt   string                     `json:"generated_at,omitempty"`
	Tracked       int                        `json:"tracked"`
	Monitored     int                        `json:"monitored"`
	Unmonitored   int                        `json:"unmonitored"`
	ExternalEdits int                        `json:"external_edits"`
	Files         []configDiagnosticFileView `json:"files,omitempty"`
}

type configDiagnosticFileView struct {
	File          string `json:"file"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Monitoring    string `json:"monitoring"`
	LastWriter    string `json:"last_writer,omitempty"`
	LastSeenAt    string `json:"last_seen_at,omitempty"`
	LastChangedAt string `json:"last_changed_at,omitempty"`
}

type verificationDiagnosticsView struct {
	Status         string   `json:"status"`
	Summary        string   `json:"summary,omitempty"`
	Missing        []string `json:"missing,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	PlatformState  string   `json:"platform_state,omitempty"`
	Readiness      string   `json:"readiness,omitempty"`
	StorageState   string   `json:"storage_state,omitempty"`
	OwnershipState string   `json:"ownership_state,omitempty"`
	AgentSocket    bool     `json:"agent_socket"`
	AdminPresent   bool     `json:"admin_present"`
}

type ownershipDiagnosticsView struct {
	GeneratedAt string                         `json:"generated_at,omitempty"`
	Managed     int                            `json:"managed"`
	Provisioned int                            `json:"provisioned"`
	Pending     int                            `json:"pending"`
	Blocked     int                            `json:"blocked"`
	Active      int                            `json:"active"`
	Retired     int                            `json:"retired"`
	Claims      []ownershipDiagnosticClaimView `json:"claims,omitempty"`
	Workloads   []actorLifecycleDiagnosticView `json:"workloads,omitempty"`
}

type ownershipDiagnosticClaimView struct {
	Crate     string `json:"crate"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	ID        string `json:"id,omitempty"`
	User      string `json:"user,omitempty"`
	Group     string `json:"group,omitempty"`
	Home      string `json:"home,omitempty"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at,omitempty"`
	RetiredAt string `json:"retired_at,omitempty"`
}

type actorLifecycleDiagnosticView struct {
	Crate                 string                              `json:"crate"`
	ActorName             string                              `json:"actor_name,omitempty"`
	ActorType             string                              `json:"actor_type,omitempty"`
	ActorID               string                              `json:"actor_id,omitempty"`
	ActorUser             string                              `json:"actor_user,omitempty"`
	ActorGroup            string                              `json:"actor_group,omitempty"`
	ActorHome             string                              `json:"actor_home,omitempty"`
	Provisioning          string                              `json:"provisioning,omitempty"`
	ProvisioningError     string                              `json:"provisioning_error,omitempty"`
	ProvisioningUpdatedAt string                              `json:"provisioning_updated_at,omitempty"`
	LastSuccessAt         string                              `json:"last_success_at,omitempty"`
	LastFailureAt         string                              `json:"last_failure_at,omitempty"`
	ProvisioningStatePath string                              `json:"provisioning_state_path,omitempty"`
	OwnershipStatus       string                              `json:"ownership_status,omitempty"`
	OwnershipUpdatedAt    string                              `json:"ownership_updated_at,omitempty"`
	OwnershipRetiredAt    string                              `json:"ownership_retired_at,omitempty"`
	RecentEvents          []actorLifecycleEventDiagnosticView `json:"recent_events,omitempty"`
}

type actorLifecycleEventDiagnosticView struct {
	At           string `json:"at,omitempty"`
	Provisioning string `json:"provisioning,omitempty"`
	Error        string `json:"error,omitempty"`
}

const maxCrateStateAge = 20 * time.Minute

func buildDiagnosticsView() diagnosticsView {
	return diagnosticsView{
		Config:       loadConfigDiagnostics(),
		Verification: loadVerificationDiagnostics(),
		Ownership:    loadOwnershipDiagnostics(),
	}
}

func loadVerificationDiagnostics() verificationDiagnosticsView {
	view := verificationDiagnosticsView{
		Status:   "ready",
		Missing:  []string{},
		Warnings: []string{},
	}
	requiredFiles := []struct {
		path  string
		label string
	}{
		{platform.CratePath("state", "installed.json"), "installed marker"},
		{platform.CratePath("state", "platform-state.json"), "platform state"},
		{platform.CratePath("state", "readiness-report.json"), "readiness report"},
		{platform.CratePath("state", "storage-state.json"), "storage state"},
		{platform.CratePath("state", "actor-ownership-state.json"), "actor ownership state"},
	}
	for _, item := range requiredFiles {
		if _, err := os.Stat(item.path); err != nil {
			view.Missing = append(view.Missing, item.label)
		}
	}
	if _, err := os.Stat(platform.AgentSocket); err == nil {
		view.AgentSocket = true
	}
	if cfg, err := config.Load(); err == nil {
		for _, user := range cfg.Users.Users {
			if strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
				view.AdminPresent = true
				break
			}
		}
	}
	platformState := state.LoadPlatformState()
	view.PlatformState = strings.TrimSpace(platformState.GeneratedAt)
	if view.PlatformState == "" {
		view.Warnings = append(view.Warnings, "platform state not rendered yet")
	}
	if readiness, ok := loadReadinessStatusSummary(); ok {
		view.Readiness = readiness
		if readiness != "ready" {
			view.Warnings = append(view.Warnings, "readiness report is not ready")
		}
	} else {
		view.Warnings = append(view.Warnings, "readiness report unreadable")
	}
	if storage := state.LoadStorageState(); strings.TrimSpace(storage.GeneratedAt) != "" {
		view.StorageState = storage.GeneratedAt
	} else {
		view.Warnings = append(view.Warnings, "storage posture not rendered yet")
	}
	if ownership := state.LoadActorOwnershipState(); strings.TrimSpace(ownership.GeneratedAt) != "" {
		view.OwnershipState = ownership.GeneratedAt
	} else {
		view.Warnings = append(view.Warnings, "actor ownership state not rendered yet")
	}
	if !view.AgentSocket {
		view.Warnings = append(view.Warnings, "agent socket unavailable")
	}
	if !view.AdminPresent {
		view.Missing = append(view.Missing, "admin operator")
	}
	switch {
	case len(view.Missing) > 0:
		view.Status = "failed"
		view.Summary = "verification prerequisites missing"
	case len(view.Warnings) > 0:
		view.Status = "degraded"
		view.Summary = "verification surfaces present with warnings"
	default:
		view.Summary = "verification surfaces present"
	}
	return view
}

func loadOwnershipDiagnostics() ownershipDiagnosticsView {
	snapshot := state.LoadActorOwnershipState()
	view := ownershipDiagnosticsView{
		GeneratedAt: strings.TrimSpace(snapshot.GeneratedAt),
		Active:      snapshot.Active,
		Retired:     snapshot.Retired,
		Claims:      make([]ownershipDiagnosticClaimView, 0, len(snapshot.Claims)),
		Workloads:   []actorLifecycleDiagnosticView{},
	}
	claimsByCrate := map[string]state.ActorOwnershipStateItem{}
	for _, claim := range snapshot.Claims {
		claimsByCrate[strings.TrimSpace(claim.Crate)] = claim
		view.Claims = append(view.Claims, ownershipDiagnosticClaimView{
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
			view.Managed++
			provisioningState := state.LoadActorProvisioningState(svc.Name)
			workload := actorLifecycleDiagnosticView{
				Crate:                 svc.Name,
				ActorName:             strings.TrimSpace(provisioningState.Actor.Name),
				ActorType:             strings.TrimSpace(provisioningState.Actor.Type),
				ActorID:               strings.TrimSpace(provisioningState.Actor.ID),
				ActorUser:             strings.TrimSpace(provisioningState.Actor.User),
				ActorGroup:            strings.TrimSpace(provisioningState.Actor.Group),
				ActorHome:             strings.TrimSpace(provisioningState.Actor.Home),
				Provisioning:          strings.TrimSpace(provisioningState.Provisioning),
				ProvisioningError:     strings.TrimSpace(provisioningState.Error),
				ProvisioningUpdatedAt: strings.TrimSpace(provisioningState.GeneratedAt),
				LastSuccessAt:         strings.TrimSpace(provisioningState.LastSuccessAt),
				LastFailureAt:         strings.TrimSpace(provisioningState.LastFailureAt),
				RecentEvents:          make([]actorLifecycleEventDiagnosticView, 0, len(provisioningState.Events)),
			}
			for _, event := range provisioningState.Events {
				workload.RecentEvents = append(workload.RecentEvents, actorLifecycleEventDiagnosticView{
					At:           strings.TrimSpace(event.At),
					Provisioning: strings.TrimSpace(event.Provisioning),
					Error:        strings.TrimSpace(event.Error),
				})
			}
			if workload.ActorName == "" {
				workload.ActorName = strings.TrimSpace(svc.Actor.Name)
			}
			workload.ProvisioningStatePath = platform.CratePath("services", svc.Name, "runtime", "actor-provisioning.json")
			if claim, ok := claimsByCrate[strings.TrimSpace(svc.Name)]; ok {
				workload.OwnershipStatus = strings.TrimSpace(claim.Status)
				workload.OwnershipUpdatedAt = strings.TrimSpace(claim.UpdatedAt)
				workload.OwnershipRetiredAt = strings.TrimSpace(claim.RetiredAt)
				if workload.ActorName == "" {
					workload.ActorName = strings.TrimSpace(claim.Name)
				}
				if workload.ActorType == "" {
					workload.ActorType = strings.TrimSpace(claim.Type)
				}
				if workload.ActorID == "" {
					workload.ActorID = strings.TrimSpace(claim.ID)
				}
				if workload.ActorUser == "" {
					workload.ActorUser = strings.TrimSpace(claim.User)
				}
				if workload.ActorGroup == "" {
					workload.ActorGroup = strings.TrimSpace(claim.Group)
				}
				if workload.ActorHome == "" {
					workload.ActorHome = strings.TrimSpace(claim.Home)
				}
			}
			switch workload.Provisioning {
			case "provisioned":
				view.Provisioned++
			case "blocked":
				view.Blocked++
			default:
				view.Pending++
			}
			view.Workloads = append(view.Workloads, workload)
		}
	}
	return view
}

func loadReadinessStatusSummary() (string, bool) {
	data, err := os.ReadFile(platform.CratePath("state", "readiness-report.json"))
	if err != nil {
		return "", false
	}
	var report struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return "", false
	}
	status := strings.TrimSpace(report.Status)
	if status == "" {
		status = "unknown"
	}
	return status, true
}

func loadConfigDiagnostics() configDiagnosticsView {
	ledger, err := config.LoadConfigChangeLedger()
	if err != nil {
		return configDiagnosticsView{}
	}
	view := configDiagnosticsView{
		GeneratedAt: ledger.GeneratedAt,
		Files:       make([]configDiagnosticFileView, 0, len(ledger.Files)),
	}
	for _, record := range ledger.Files {
		view.Tracked++
		switch strings.TrimSpace(record.Monitoring) {
		case "unmonitored":
			view.Unmonitored++
		default:
			view.Monitored++
		}
		if strings.TrimSpace(record.LastWriter) == "external" {
			view.ExternalEdits++
		}
		view.Files = append(view.Files, configDiagnosticFileView{
			File:          record.File,
			Path:          record.Path,
			Exists:        record.Exists,
			Monitoring:    record.Monitoring,
			LastWriter:    record.LastWriter,
			LastSeenAt:    record.LastSeenAt,
			LastChangedAt: record.LastChangedAt,
		})
	}
	return view
}
