package tui

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
	"github.com/crateos/crateos/internal/sysinfo"
)

func gatherServices() []ServiceInfo {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	actual := state.Probe(state.CollectServiceNames(cfg))
	actualByName := make(map[string]state.ServiceState, len(actual.Services))
	for _, svc := range actual.Services {
		actualByName[svc.Name] = svc
	}

	mods := modules.LoadAll(".")
	svcs := make([]ServiceInfo, 0, len(cfg.Services.Services))
	for _, desired := range cfg.Services.Services {
		service := ServiceInfo{
			Name:      desired.Name,
			Type:      modules.ResolveRuntime(desired.Runtime, mods[desired.Name], false),
			Status:    "unknown",
			Health:    "unknown",
			Desired:   desired.Enabled,
			Autostart: desired.Autostart,
			Enabled:   desired.Enabled,
			Ready:     !desired.Enabled,
		}
		if mod, ok := mods[desired.Name]; ok {
			service.Module = true
			service.DisplayName = mod.DisplayName()
			service.Category = mod.Metadata.Category
			service.Type = modules.ResolveRuntime(desired.Runtime, mod, true)
			for _, unit := range modules.ResolveUnits(desired.Name, mod, true) {
				unitState, ok := actualByName[unit]
				if !ok {
					unitState = state.ServiceState{Name: unit, Status: "unknown", Health: "unknown"}
				}
				service.Units = append(service.Units, ServiceUnit{
					Name:    unitState.Name,
					Active:  unitState.Active,
					Enabled: unitState.Enabled,
					Status:  unitState.Status,
					Health:  unitState.Health,
				})
			}
		}
		if service.DisplayName == "" {
			service.DisplayName = desired.Name
		}
		if len(service.Units) == 0 {
			if unitState, ok := actualByName[desired.Name]; ok {
				service.Units = append(service.Units, ServiceUnit{
					Name:    unitState.Name,
					Active:  unitState.Active,
					Enabled: unitState.Enabled,
					Status:  unitState.Status,
					Health:  unitState.Health,
				})
			}
		}
		service.Status, service.Health, service.Enabled, service.Ready = summarizeServiceState(desired.Enabled, service.Units)
		service.Active = service.Status == "active" || service.Status == "partial"
		service.LastError = readFallbackCrateLastError(desired.Name)
		svcs = append(svcs, service)
	}
	return svcs
}

func (m *model) refreshServices() {
	if info, svcs, platformInfo, _, _ := fetchStatusViaAPI(m.currentUser); info != nil {
		m.info = *info
		m.services = svcs
		m.platform = platformInfo
		m.controlPlaneOnline = true
		return
	}
	m.services = gatherServices()
	m.platform = readFallbackPlatformState()
	m.info = sysinfo.Gather()
	m.controlPlaneOnline = false
}

func summarizeServiceState(desired bool, units []ServiceUnit) (status string, health string, enabled bool, ready bool) {
	if len(units) == 0 {
		if desired {
			return "unknown", "unknown", true, false
		}
		return "inactive", "unknown", false, true
	}

	activeCount := 0
	enabledCount := 0
	failedCount := 0
	healthyCount := 0
	for _, unit := range units {
		if unit.Active {
			activeCount++
		}
		if unit.Enabled {
			enabledCount++
		}
		if unit.Status == "failed" {
			failedCount++
		}
		if unit.Health == "ok" {
			healthyCount++
		}
	}

	switch {
	case failedCount > 0:
		status = "failed"
	case activeCount == len(units):
		status = "active"
	case activeCount > 0:
		status = "partial"
	case desired:
		status = "inactive"
	default:
		status = "inactive"
	}

	switch {
	case healthyCount == len(units):
		health = "ok"
	case activeCount > 0:
		health = "degraded"
	default:
		health = "unknown"
	}

	enabled = enabledCount == len(units)
	ready = !desired || (activeCount == len(units) && healthyCount == len(units))
	return status, health, enabled, ready
}

func readFallbackCrateLastError(name string) string {
	path := platform.CratePath("services", name, "crate-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var stored struct {
		Crate struct {
			LastError string `json:"last_error"`
			Summary   string `json:"summary"`
		} `json:"crate"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return ""
	}
	if strings.TrimSpace(stored.Crate.LastError) != "" {
		return stored.Crate.LastError
	}
	return stored.Crate.Summary
}
