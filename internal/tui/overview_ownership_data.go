package tui

import (
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
)

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
