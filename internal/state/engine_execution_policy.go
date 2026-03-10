package state

import (
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
)

func normalizeExecutionMode(mode, runtimeName string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "service", "job":
		return mode
	}
	if strings.EqualFold(strings.TrimSpace(runtimeName), "task") {
		return "job"
	}
	return "service"
}

func normalizeDurationField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeTimeoutBehavior(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "kill":
		return "kill"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeKillSignal(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "SIGTERM":
		return "SIGTERM"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func normalizeConcurrencyPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "replace":
		return "replace"
	case "forbid":
		return "forbid"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func summarizeExecutionPolicy(crate CrateState) string {
	parts := []string{}
	if crate.ActorName != "" {
		parts = append(parts, "actor "+crate.ActorName)
	}
	if crate.ExecutionMode != "" {
		parts = append(parts, "mode "+crate.ExecutionMode)
	}
	if crate.Schedule != "" {
		parts = append(parts, "schedule "+crate.Schedule)
	}
	if crate.Timeout != "" {
		parts = append(parts, "timeout "+crate.Timeout)
	}
	if crate.ConcurrencyPolicy != "" {
		parts = append(parts, "overlap "+crate.ConcurrencyPolicy)
	}
	if crate.ExecutionStatus != "" {
		parts = append(parts, "runtime "+crate.ExecutionStatus)
	}
	return strings.Join(parts, " · ")
}

func executionAdapterForRuntime(runtimeName string) string {
	switch strings.ToLower(strings.TrimSpace(runtimeName)) {
	case "docker":
		return "docker"
	default:
		return "systemd"
	}
}

func inferExecutionStatus(desired config.ServiceEntry, hasMod bool) string {
	if hasMod {
		return "module-owned"
	}
	if normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) == "job" && translateExecutionSchedule(strings.TrimSpace(desired.Execution.Schedule)) == "" {
		return "schedule-invalid"
	}
	switch normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) {
	case "job":
		return "native-timer"
	default:
		return "native-service"
	}
}

func inferExecutionUnits(desired config.ServiceEntry, mod modules.Module, hasMod bool) (string, string) {
	if hasMod {
		units := modules.ResolveUnits(desired.Name, mod, true)
		switch len(units) {
		case 0:
			return "", ""
		case 1:
			return units[0], ""
		default:
			return units[0], units[1]
		}
	}
	name := strings.TrimSpace(desired.Name)
	if name == "" {
		return "", ""
	}
	switch normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) {
	case "job":
		base := managedExecutionUnitBase(name)
		return base + ".service", base + ".timer"
	default:
		base := managedExecutionUnitBase(name)
		return base + ".service", ""
	}
}
