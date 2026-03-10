package tui

import (
	"fmt"
	"strings"
)

func renderServiceLifecycleSection(s ServiceInfo) string {
	var lifecycle strings.Builder
	lifecycle.WriteString(renderBadgeRow(
		lifecycleDesiredBadge(s),
		lifecycleAutostartBadge(s),
		lifecycleRuntimeBadge(s),
		readyBadge(s.Ready),
	))
	lifecycle.WriteString("\n")
	lifecycle.WriteString(dim.Render("intent: " + lifecycleIntentText(s)))
	lifecycle.WriteString("\n")
	lifecycle.WriteString(dim.Render("units: " + lifecycleUnitCounts(s)))
	lifecycle.WriteString("\n")
	if s.DisplayName != "" && s.DisplayName != s.Name {
		lifecycle.WriteString(dim.Render("id: " + s.Name))
		lifecycle.WriteString("\n")
	}
	if s.Module {
		lifecycle.WriteString(dim.Render("module category: " + s.Category))
		lifecycle.WriteString("\n")
	}
	if s.Summary != "" {
		lifecycle.WriteString(dim.Render("summary: " + s.Summary))
		lifecycle.WriteString("\n")
	}
	if s.LastAction != "" {
		lifecycle.WriteString(dim.Render("last action: " + s.LastAction))
		if s.LastActionAt != "" {
			lifecycle.WriteString(dim.Render(" @ " + s.LastActionAt))
		}
		lifecycle.WriteString("\n")
	}
	if s.LastError != "" {
		lifecycle.WriteString(danger.Render("issue: " + s.LastError))
		lifecycle.WriteString("\n")
	}
	if s.SuggestedRepair != "" {
		lifecycle.WriteString(warn.Render("repair: " + s.SuggestedRepair))
		lifecycle.WriteString("\n")
	}
	if (s.Status == "failed" || s.Status == "partial" || s.Health == "degraded") && (s.LastGoodStatus != "" || s.LastGoodSummary != "") {
		lastGood := []string{}
		if s.LastGoodStatus != "" {
			lastGood = append(lastGood, "state:"+s.LastGoodStatus)
		}
		if s.LastGoodHealth != "" {
			lastGood = append(lastGood, "health:"+s.LastGoodHealth)
		}
		if s.LastGoodAt != "" {
			lastGood = append(lastGood, "at:"+s.LastGoodAt)
		}
		lifecycle.WriteString(ok.Render("last-good: " + strings.Join(lastGood, "  ")))
		lifecycle.WriteString("\n")
		if s.LastGoodSummary != "" {
			lifecycle.WriteString(dim.Render("last-good summary: " + s.LastGoodSummary))
			lifecycle.WriteString("\n")
		}
	}
	if s.Module && !s.PackagesInstalled && len(s.MissingPackages) > 0 {
		lifecycle.WriteString(danger.Render("missing packages: " + strings.Join(s.MissingPackages, ", ")))
		lifecycle.WriteString("\n")
	}
	if s.Stateful {
		if strings.TrimSpace(s.StorageSummary) != "" {
			lifecycle.WriteString(dim.Render("storage: " + s.StorageSummary))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.DataPath) != "" {
			lifecycle.WriteString(dim.Render("data path: " + s.DataPath))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.NativeDataPath) != "" {
			lifecycle.WriteString(dim.Render("native data: " + s.NativeDataPath))
			lifecycle.WriteString("\n")
		}
	}
	if strings.TrimSpace(s.ActorName) != "" || strings.TrimSpace(s.ExecutionMode) != "" {
		lifecycle.WriteString(dim.Render("actor: " + valueOrFallback(strings.TrimSpace(s.ActorName), "unassigned")))
		if strings.TrimSpace(s.ActorType) != "" {
			lifecycle.WriteString(dim.Render(" · type: " + s.ActorType))
		}
		lifecycle.WriteString("\n")
		if strings.TrimSpace(s.ActorID) != "" {
			lifecycle.WriteString(dim.Render("actor id: " + s.ActorID))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorUser) != "" || strings.TrimSpace(s.ActorGroup) != "" {
			lifecycle.WriteString(dim.Render("actor runtime account: " + valueOrFallback(strings.TrimSpace(s.ActorUser), "pending")))
			lifecycle.WriteString(dim.Render(":" + valueOrFallback(strings.TrimSpace(s.ActorGroup), "pending")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorHome) != "" {
			lifecycle.WriteString(dim.Render("actor home: " + s.ActorHome))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorProvisioning) != "" {
			lifecycle.WriteString(dim.Render("actor provisioning: " + s.ActorProvisioning))
			if strings.TrimSpace(s.ActorProvisioningError) != "" {
				lifecycle.WriteString(danger.Render(" · " + s.ActorProvisioningError))
			}
			lifecycle.WriteString("\n")
			if strings.TrimSpace(s.ActorProvisioningUpdatedAt) != "" {
				lifecycle.WriteString(dim.Render("actor provisioning updated: " + s.ActorProvisioningUpdatedAt))
				lifecycle.WriteString("\n")
			}
			if strings.TrimSpace(s.ActorProvisioningStatePath) != "" {
				lifecycle.WriteString(dim.Render("actor provisioning state: " + s.ActorProvisioningStatePath))
				lifecycle.WriteString("\n")
			}
		}
		if strings.TrimSpace(s.ActorOwnershipStatus) != "" {
			lifecycle.WriteString(dim.Render("actor ownership: " + s.ActorOwnershipStatus))
			if strings.TrimSpace(s.ActorOwnershipUpdatedAt) != "" {
				lifecycle.WriteString(dim.Render(" @ " + s.ActorOwnershipUpdatedAt))
			}
			lifecycle.WriteString("\n")
			if strings.TrimSpace(s.ActorOwnershipRetiredAt) != "" {
				lifecycle.WriteString(dim.Render("actor ownership retired: " + s.ActorOwnershipRetiredAt))
				lifecycle.WriteString("\n")
			}
		}
		if strings.TrimSpace(s.ActorRuntimeDir) != "" || strings.TrimSpace(s.ActorStateDir) != "" {
			lifecycle.WriteString(dim.Render("actor dirs: run " + valueOrFallback(strings.TrimSpace(s.ActorRuntimeDir), "pending")))
			lifecycle.WriteString(dim.Render(" · state " + valueOrFallback(strings.TrimSpace(s.ActorStateDir), "pending")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionSummary) != "" {
			lifecycle.WriteString(dim.Render("execution: " + s.ExecutionSummary))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.DeploySource) != "" {
			lifecycle.WriteString(dim.Render("deploy: " + s.DeploySource))
			if strings.TrimSpace(s.UploadPath) != "" {
				lifecycle.WriteString(dim.Render(" · intake: " + s.UploadPath))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.WorkingDir) != "" {
			lifecycle.WriteString(dim.Render("working dir: " + s.WorkingDir))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.InstallCommand) != "" {
			lifecycle.WriteString(dim.Render("install: " + s.InstallCommand))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionAdapter) != "" || strings.TrimSpace(s.ExecutionStatus) != "" {
			lifecycle.WriteString(dim.Render("runtime object: " + valueOrFallback(strings.TrimSpace(s.ExecutionAdapter), "systemd")))
			if strings.TrimSpace(s.ExecutionStatus) != "" {
				lifecycle.WriteString(dim.Render(" · " + s.ExecutionStatus))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.PrimaryUnit) != "" {
			lifecycle.WriteString(dim.Render("primary unit: " + s.PrimaryUnit))
			if strings.TrimSpace(s.CompanionUnit) != "" {
				lifecycle.WriteString(dim.Render(" · companion: " + s.CompanionUnit))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.PrimaryUnitPath) != "" {
			lifecycle.WriteString(dim.Render("unit file: " + s.PrimaryUnitPath))
			if strings.TrimSpace(s.CompanionUnitPath) != "" {
				lifecycle.WriteString(dim.Render(" · timer file: " + s.CompanionUnitPath))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionStatePath) != "" {
			lifecycle.WriteString(dim.Render("execution state: " + s.ExecutionStatePath))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.Schedule) != "" {
			lifecycle.WriteString(dim.Render("schedule: " + s.Schedule))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.Timeout) != "" || strings.TrimSpace(s.StopTimeout) != "" {
			lifecycle.WriteString(dim.Render("timeouts: run " + valueOrFallback(strings.TrimSpace(s.Timeout), "0") + " · stop " + valueOrFallback(strings.TrimSpace(s.StopTimeout), "30s")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.OnTimeout) != "" {
			lifecycle.WriteString(dim.Render("timeout action: " + s.OnTimeout + " via " + valueOrFallback(strings.TrimSpace(s.KillSignal), "SIGTERM")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ConcurrencyPolicy) != "" {
			lifecycle.WriteString(dim.Render("overlap: " + s.ConcurrencyPolicy))
			lifecycle.WriteString("\n")
		}
	}
	return lifecycle.String()
}

func renderServiceUnitGrid(s ServiceInfo) string {
	var unitGrid strings.Builder
	if len(s.Units) == 0 {
		unitGrid.WriteString(dim.Render("No unit detail available."))
		unitGrid.WriteString("\n")
		return unitGrid.String()
	}
	for _, unit := range s.Units {
		line := fmt.Sprintf("%s %s  %s  %s  %s  %s",
			statusIndicator(unit.Status),
			unit.Name,
			statusBadge(unit.Status),
			healthBadge(unit.Health),
			binaryBadge("enabled", unit.Enabled),
			binaryBadge("active", unit.Active),
		)
		unitGrid.WriteString(dim.Render(line))
		unitGrid.WriteString("\n")
	}
	return unitGrid.String()
}

func renderServiceLegend() string {
	return fmt.Sprintf("%s active   %s inactive   %s failed   %s unknown",
		ok.Render("●"),
		dim.Render("○"),
		danger.Render("✖"),
		warn.Render("?"),
	)
}

func renderServicePostureSection(services []ServiceInfo) string {
	failed, partial, staged, healthy := menuServiceCounts(services)
	var posture strings.Builder
	posture.WriteString(renderBadgeRow(
		selectorStat("trk", len(services)),
		selectorStat("ok", healthy),
		selectorStat("bad", failed+partial),
		selectorStat("stg", staged),
	))
	posture.WriteString("\n")
	posture.WriteString(dim.Render("This crate sits inside the wider desired-state fleet; use Status for aggregate diagnostics when local symptoms stack."))
	return posture.String()
}
