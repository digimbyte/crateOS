package state

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
)

func buildCrateState(desired config.ServiceEntry, actualByName map[string]ServiceState, mod modules.Module, hasMod bool) CrateState {
	crate := CrateState{
		Name:        desired.Name,
		DisplayName: desired.Name,
		Runtime:     modules.ResolveRuntime(desired.Runtime, mod, hasMod),
		Desired:     desired.Enabled,
		Autostart:   desired.Autostart,
		Status:      "unknown",
		Health:      "unknown",
		Ready:       !desired.Enabled,
	}
	applyManagedExecutionMetadata(&crate, desired, mod, hasMod)
	applyManagedActorProvisioningPosture(&crate)

	if hasMod {
		crate.Module = true
		crate.DisplayName = mod.DisplayName()
		crate.Category = mod.Metadata.Category
		crate.Description = mod.Metadata.Description
		crate.InstallMode = modules.ResolveInstallMode(mod, true)
		crate.UnitNames = modules.ResolveUnits(desired.Name, mod, true)
		crate.DataPath = strings.TrimSpace(mod.Spec.Paths.Data.Canonical)
		crate.NativeDataPath = strings.TrimSpace(mod.Spec.Paths.Data.Native)
		crate.Stateful = crate.DataPath != "" || crate.NativeDataPath != ""
		if crate.Stateful {
			switch {
			case crate.DataPath != "" && crate.NativeDataPath != "":
				crate.StorageSummary = fmt.Sprintf("canonical data path %s maps to native %s", crate.DataPath, crate.NativeDataPath)
			case crate.DataPath != "":
				crate.StorageSummary = "canonical data path " + crate.DataPath
			case crate.NativeDataPath != "":
				crate.StorageSummary = "native data path " + crate.NativeDataPath
			}
		}
		for _, unit := range crate.UnitNames {
			if unitState, ok := actualByName[unit]; ok {
				crate.Units = append(crate.Units, unitState)
			} else {
				crate.Units = append(crate.Units, ServiceState{Name: unit, Status: "unknown", Health: "unknown"})
			}
		}
		crate.PackagesInstalled = true
		for _, pkg := range mod.Spec.Packages {
			if !packageInstalled(pkg) {
				crate.PackagesInstalled = false
				crate.MissingPackages = append(crate.MissingPackages, pkg)
			}
		}
		for _, hc := range mod.Spec.HealthChecks {
			if hc.Command != "" {
				crate.HealthChecks = append(crate.HealthChecks, hc.Command)
			}
		}
	} else {
		for _, unit := range desiredManagedUnits(desired, mod, hasMod) {
			if svc, ok := actualByName[unit]; ok {
				crate.Units = append(crate.Units, svc)
			} else {
				crate.Units = append(crate.Units, ServiceState{Name: unit, Status: "unknown", Health: "unknown"})
			}
		}
		crate.PackagesInstalled = true
		if strings.TrimSpace(desired.Runtime) == "docker" {
			crate.StorageSummary = "runtime-managed data path depends on the selected service implementation"
		}
	}

	waitingForExplicitStart := hasMod && mod.InstallMode() == "staged" && desired.Enabled && !desired.Autostart
	if waitingForExplicitStart {
		crate.Status = "staged"
		crate.Health = "pending"
		crate.Summary = "waiting for explicit start"
	} else {
		crate.Status = aggregateCrateStatus(crate.Units)
		crate.Health = aggregateCrateHealth(crate.Units)
		crate.Summary = summarizeCrateUnits(crate.Units)
	}

	ready := desired.Enabled
	if !desired.Enabled {
		ready = true
	}
	if desired.Enabled {
		if hasMod {
			if !crate.PackagesInstalled {
				ready = false
			}
			if !waitingForExplicitStart {
				for _, unit := range crate.Units {
					if !unit.Active {
						ready = false
					}
					if unit.Health != "ok" {
						ready = false
					}
				}
			}
		} else {
			primary := actualByName[desired.Name]
			if len(crate.Units) > 0 {
				primary = crate.Units[0]
			}
			ready = primary.Active
		}
	}
	crate.Ready = ready
	if desired.Enabled && !ready && crate.LastError == "" {
		if len(crate.MissingPackages) > 0 {
			crate.LastError = "missing packages: " + strings.Join(crate.MissingPackages, ", ")
		} else if crate.Status == "partial" {
			crate.LastError = "crate units are not in a consistent running state"
		} else if crate.Health != "ok" && crate.Health != "pending" {
			crate.LastError = "service health is " + crate.Health
		}
	}
	validationIssues := validateManagedExecutionPolicy(crate)
	if len(validationIssues) > 0 {
		crate.Health = "degraded"
		if crate.Status == "" || crate.Status == "unknown" {
			crate.Status = "partial"
		}
		crate.Ready = false
		if crate.LastError == "" {
			crate.LastError = validationIssues[0]
		}
		if crate.Summary == "" || crate.Summary == "rendered desired state successfully" {
			crate.Summary = validationIssues[0]
		}
	}
	if crate.DisplayName == "" {
		crate.DisplayName = desired.Name
	}
	if crate.ActorProvisioningError != "" && crate.LastError == "" {
		crate.LastError = crate.ActorProvisioningError
	}
	crate.LastAction = inferCrateLastAction(desired, crate, waitingForExplicitStart)
	crate.LastActionAt = actualTimestamp()
	crate.SuggestedRepair = inferCrateSuggestedRepair(crate, waitingForExplicitStart)
	return crate
}

func inferCrateLastAction(desired config.ServiceEntry, crate CrateState, waitingForExplicitStart bool) string {
	switch {
	case !desired.Enabled:
		return "disable"
	case len(crate.MissingPackages) > 0:
		return "install"
	case waitingForExplicitStart:
		return "enable"
	case crate.Status == "failed" || crate.Status == "partial":
		return "start"
	case desired.Autostart:
		return "reconcile"
	default:
		return "configure"
	}
}

func inferCrateSuggestedRepair(crate CrateState, waitingForExplicitStart bool) string {
	switch {
	case crate.ActorProvisioning == "blocked":
		return "repair managed actor identity policy before reconcile can continue"
	case crate.ActorProvisioning == "pending" && crate.ExecutionMode != "":
		return "run reconcile to provision the managed actor runtime identity"
	case crate.ActorName == "" && crate.ExecutionMode != "":
		return "assign a managed actor for this workload"
	case crate.ExecutionMode == "job" && crate.Schedule == "":
		return "set a schedule before enabling recurring job execution"
	case crate.ExecutionMode != "" && crate.StartCommand == "" && !crate.Module:
		return "set a start command or bind the workload to a module-defined runtime"
	case len(crate.MissingPackages) > 0:
		return "repair dependencies and rerun reconcile"
	case waitingForExplicitStart:
		return "start crate explicitly after staged install"
	case crate.Status == "failed":
		return "inspect unit failures and rerun reconcile"
	case crate.Status == "partial":
		return "repair inconsistent units and rerun reconcile"
	case crate.Health == "degraded":
		return "run health checks and repair dependent resources"
	default:
		return ""
	}
}

func applyManagedExecutionMetadata(crate *CrateState, desired config.ServiceEntry, mod modules.Module, hasMod bool) {
	crate.ActorName = strings.TrimSpace(desired.Actor.Name)
	crate.ActorType = strings.TrimSpace(desired.Actor.Type)
	// TODO: Split desired actor identity posture from provisioned native account state once reconcile can detect collisions and lifecycle failures.
	crate.ActorID, crate.ActorUser, crate.ActorGroup = resolveManagedActorIdentity(desired.Name, crate.ActorName)
	if crate.ActorUser != "" {
		crate.ActorHome = filepath.Join(platform.CratePath("services", desired.Name), "runtime", "actors", crate.ActorUser)
		crate.ActorRuntimeDir = filepath.Join(crate.ActorHome, "run")
		crate.ActorStateDir = filepath.Join(crate.ActorHome, "state")
	}
	crate.DeploySource = strings.TrimSpace(desired.Deploy.Source)
	crate.UploadPath = strings.TrimSpace(desired.Deploy.UploadPath)
	crate.WorkingDir = strings.TrimSpace(desired.Deploy.WorkingDir)
	crate.Entry = strings.TrimSpace(desired.Deploy.Entry)
	crate.InstallCommand = strings.TrimSpace(desired.Deploy.InstallCmd)
	crate.EnvironmentFile = strings.TrimSpace(desired.Deploy.EnvFile)
	crate.ExecutionMode = normalizeExecutionMode(desired.Execution.Mode, desired.Runtime)
	crate.ExecutionAdapter = executionAdapterForRuntime(desired.Runtime)
	crate.ExecutionStatus = inferExecutionStatus(desired, hasMod)
	crate.PrimaryUnit, crate.CompanionUnit = inferExecutionUnits(desired, mod, hasMod)
	crate.ExecutionStatePath = filepath.Join(platform.CratePath("services", desired.Name), "runtime", "execution-posture.json")
	crate.StartCommand = strings.TrimSpace(desired.Execution.StartCmd)
	crate.Schedule = strings.TrimSpace(desired.Execution.Schedule)
	crate.Timeout = normalizeDurationField(desired.Execution.Timeout, "0")
	crate.StopTimeout = normalizeDurationField(desired.Execution.StopTimeout, "30s")
	crate.OnTimeout = normalizeTimeoutBehavior(desired.Execution.OnTimeout)
	crate.KillSignal = normalizeKillSignal(desired.Execution.KillSignal)
	crate.ConcurrencyPolicy = normalizeConcurrencyPolicy(desired.Execution.Concurrency)
	crate.PrimaryUnitPath, crate.CompanionUnitPath = inferExecutionArtifactPaths(desired, *crate, hasMod)
	crate.ExecutionSummary = summarizeExecutionPolicy(*crate)
}

func validateManagedExecutionPolicy(crate CrateState) []string {
	issues := []string{}
	if crate.ActorName == "" {
		issues = append(issues, "managed workload requires actor.name")
	}
	if crate.ExecutionMode == "job" && crate.Schedule == "" {
		issues = append(issues, "job execution requires execution.schedule")
	}
	if crate.ExecutionMode == "job" && crate.StartCommand == "" {
		issues = append(issues, "job execution requires execution.start_cmd")
	}
	if crate.DeploySource == "upload" && crate.UploadPath == "" {
		issues = append(issues, "upload deploy source requires deploy.upload_path")
	}
	if crate.DeploySource == "upload" && crate.WorkingDir == "" {
		issues = append(issues, "upload deploy source requires deploy.working_dir")
	}
	if crate.ExecutionMode == "service" && crate.StartCommand == "" && !crate.Module {
		issues = append(issues, "service execution without a module runtime requires execution.start_cmd")
	}
	return issues
}
