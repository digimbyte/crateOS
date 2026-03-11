package api

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/crateos/crateos/internal/auth"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/state"
)

// ---- Shared helpers and views --------------------------------------

func loadAuth(r *http.Request) (*config.Config, *auth.Authz, string) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, ""
	}
	authz := auth.Load(cfg)
	user := r.Header.Get("X-CrateOS-User")
	if user == "" {
		// default to the first user in config if none provided
		if len(cfg.Users.Users) > 0 {
			user = cfg.Users.Users[0].Name
		}
	}
	return cfg, authz, strings.TrimSpace(user)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func systemctlNoError(action, unit string) {
	if runtime.GOOS != "linux" {
		return
	}
	_ = exec.Command("systemctl", action, unit).Run()
}

func buildServiceView(cfg *config.Config, actual *state.ActualState, mods map[string]modules.Module) []serviceView {

	views := make([]serviceView, 0, len(cfg.Services.Services))
	for _, desired := range cfg.Services.Services {
		crate := loadCrateState(desired.Name)
		view := serviceView{
			Name:                       desired.Name,
			DisplayName:                desired.Name,
			Runtime:                    modules.ResolveRuntime(desired.Runtime, mods[desired.Name], false),
			ActorName:                  crate.ActorName,
			ActorType:                  crate.ActorType,
			ActorID:                    crate.ActorID,
			ActorUser:                  crate.ActorUser,
			ActorGroup:                 crate.ActorGroup,
			ActorHome:                  crate.ActorHome,
			ActorRuntimeDir:            crate.ActorRuntimeDir,
			ActorStateDir:              crate.ActorStateDir,
			ActorProvisioning:          crate.ActorProvisioning,
			ActorProvisioningError:     crate.ActorProvisioningError,
			ActorProvisioningUpdatedAt: crate.ActorProvisioningUpdatedAt,
			ActorProvisioningStatePath: crate.ActorProvisioningStatePath,
			ActorOwnershipStatus:       crate.ActorOwnershipStatus,
			ActorOwnershipUpdatedAt:    crate.ActorOwnershipUpdatedAt,
			ActorOwnershipRetiredAt:    crate.ActorOwnershipRetiredAt,
			DeploySource:               crate.DeploySource,
			UploadPath:                 crate.UploadPath,
			WorkingDir:                 crate.WorkingDir,
			Entry:                      crate.Entry,
			InstallCommand:             crate.InstallCommand,
			EnvironmentFile:            crate.EnvironmentFile,
			ExecutionMode:              crate.ExecutionMode,
			ExecutionAdapter:           crate.ExecutionAdapter,
			ExecutionStatus:            crate.ExecutionStatus,
			PrimaryUnit:                crate.PrimaryUnit,
			CompanionUnit:              crate.CompanionUnit,
			PrimaryUnitPath:            crate.PrimaryUnitPath,
			CompanionUnitPath:          crate.CompanionUnitPath,
			ExecutionStatePath:         crate.ExecutionStatePath,
			StartCommand:               crate.StartCommand,
			Schedule:                   crate.Schedule,
			Timeout:                    crate.Timeout,
			StopTimeout:                crate.StopTimeout,
			OnTimeout:                  crate.OnTimeout,
			KillSignal:                 crate.KillSignal,
			ConcurrencyPolicy:          crate.ConcurrencyPolicy,
			ExecutionSummary:           crate.ExecutionSummary,
			Stateful:                   crate.Stateful,
			DataPath:                   crate.DataPath,
			NativeDataPath:             crate.NativeDataPath,
			StorageSummary:             crate.StorageSummary,
			Desired:                    desired.Enabled,
			Autostart:                  desired.Autostart,
			Ready:                      crate.Ready,
			Status:                     crate.Status,
			Health:                     crate.Health,
			PackagesInstalled:          crate.PackagesInstalled,
			MissingPackages:            append([]string(nil), crate.MissingPackages...),
			Summary:                    crate.Summary,
			LastError:                  crate.LastError,
			LastAction:                 crate.LastAction,
			LastActionAt:               crate.LastActionAt,
			SuggestedRepair:            crate.SuggestedRepair,
		}
		if lastGood, ok := loadLastGoodCrateState(desired.Name); ok {
			view.LastGoodStatus = lastGood.Crate.Status
			view.LastGoodHealth = lastGood.Crate.Health
			view.LastGoodAt = lastGood.GeneratedAt
			view.LastGoodSummary = lastGood.Crate.Summary
		}
		if mod, ok := mods[desired.Name]; ok {
			view.Module = true
			view.DisplayName = mod.DisplayName()
			view.Category = mod.Metadata.Category
			view.Description = mod.Metadata.Description
			view.InstallMode = modules.ResolveInstallMode(mod, true)
			view.Packages = append([]string(nil), mod.Spec.Packages...)
			view.Runtime = modules.ResolveRuntime(desired.Runtime, mod, true)
		}
		if len(crate.Units) > 0 {
			view.Active = anyActive(crate.Units)
			view.Enabled = allEnabled(crate.Units)
			for _, unit := range crate.Units {
				view.Units = append(view.Units, unitView{
					Name:    unit.Name,
					Active:  unit.Active,
					Enabled: unit.Enabled,
					Status:  unit.Status,
					Health:  unit.Health,
				})
			}
		}
		if view.DisplayName == "" {
			view.DisplayName = desired.Name
		}
		if view.Status == "" {
			view.Status = "unknown"
		}
		if view.Health == "" {
			view.Health = "unknown"
		}
		views = append(views, view)
	}
	return views
}

func anyActive(units []state.ServiceState) bool {
	for _, unit := range units {
		if unit.Active {
			return true
		}
	}
	return false
}

func allEnabled(units []state.ServiceState) bool {
	if len(units) == 0 {
		return false
	}
	for _, unit := range units {
		if !unit.Enabled {
			return false
		}
	}
	return true
}
