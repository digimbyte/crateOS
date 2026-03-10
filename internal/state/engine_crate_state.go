package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
)

func writeCrateState(desired config.ServiceEntry, actualByName map[string]ServiceState, mods map[string]modules.Module) {
	mod, hasMod := mods[desired.Name]
	crate := buildCrateState(desired, actualByName, mod, hasMod)
	payload := StoredCrateState{
		GeneratedAt: actualTimestamp(),
		Crate:       crate,
	}
	base := platform.CratePath("services", desired.Name)
	_ = os.MkdirAll(base, 0755)
	path := filepath.Join(base, "crate-state.json")
	if err := preservePreviousStableCrateState(path, base); err != nil {
		log.Printf("warning: preserve last-good crate state %s: %v", desired.Name, err)
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("warning: marshal crate state %s: %v", desired.Name, err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("warning: write crate state %s: %v", desired.Name, err)
	}
	if err := writeManagedActorProvisioningState(crate); err != nil {
		log.Printf("warning: write actor provisioning state %s: %v", desired.Name, err)
	}
	if err := writeManagedActorRegistry(crate); err != nil {
		log.Printf("warning: write actor registry %s: %v", desired.Name, err)
	}
	if err := writeManagedActorOwnership(crate); err != nil {
		log.Printf("warning: write actor ownership %s: %v", desired.Name, err)
	}
	writeExecutionPosture(desired, crate)
	writeManagedExecutionArtifacts(desired, crate, hasMod)
	applyManagedExecutionNativeState(desired, crate, hasMod)
}

func preservePreviousStableCrateState(currentPath string, crateBase string) error {
	b, err := os.ReadFile(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var stored StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return err
	}
	if !isStableCrateState(stored.Crate) {
		return nil
	}
	lastGoodDir := filepath.Join(crateBase, "runtime", "last-good")
	if err := os.MkdirAll(lastGoodDir, 0755); err != nil {
		return err
	}
	lastGoodPath := filepath.Join(lastGoodDir, "crate-state.json")
	return os.WriteFile(lastGoodPath, b, 0644)
}

func isStableCrateState(crate CrateState) bool {
	switch crate.Status {
	case "active", "running", "inactive", "disabled", "staged":
	default:
		return false
	}
	switch crate.Health {
	case "", "ok", "pending", "unknown":
	default:
		return false
	}
	return strings.TrimSpace(crate.LastError) == ""
}

func crateProbeNames(desired config.ServiceEntry, mods map[string]modules.Module) []string {
	mod, ok := mods[desired.Name]
	names := desiredManagedUnits(desired, mod, ok)
	names = appendUnique(names, "crateos-agent", "crateos-policy")
	return names
}

func aggregateCrateStatus(units []ServiceState) string {
	if len(units) == 0 {
		return "unknown"
	}
	activeCount := 0
	failedCount := 0
	unknownCount := 0
	for _, unit := range units {
		switch unit.Status {
		case "failed":
			failedCount++
		case "active", "running":
			activeCount++
		case "", "unknown":
			unknownCount++
		}
	}
	switch {
	case failedCount > 0:
		return "failed"
	case activeCount == len(units):
		return "active"
	case activeCount > 0:
		return "partial"
	case unknownCount == len(units):
		return "unknown"
	default:
		return "inactive"
	}
}

func aggregateCrateHealth(units []ServiceState) string {
	if len(units) == 0 {
		return "unknown"
	}
	allOK := true
	for _, unit := range units {
		if unit.Health == "degraded" {
			return "degraded"
		}
		if unit.Health != "ok" {
			allOK = false
		}
	}
	if allOK {
		return "ok"
	}
	return "unknown"
}

func summarizeCrateUnits(units []ServiceState) string {
	if len(units) == 0 {
		return ""
	}
	activeCount := 0
	enabledCount := 0
	failed := make([]string, 0)
	inactive := make([]string, 0)
	for _, unit := range units {
		if unit.Active {
			activeCount++
		}
		if unit.Enabled {
			enabledCount++
		}
		if unit.Status == "failed" {
			failed = append(failed, unit.Name)
			continue
		}
		if !unit.Active {
			inactive = append(inactive, unit.Name)
		}
	}
	switch {
	case len(failed) > 0:
		return "failed units: " + strings.Join(failed, ", ")
	case activeCount > 0 && activeCount < len(units):
		return fmt.Sprintf("partially active (%d/%d units running)", activeCount, len(units))
	case enabledCount > 0 && enabledCount < len(units):
		return fmt.Sprintf("partially enabled (%d/%d units enabled)", enabledCount, len(units))
	case len(inactive) == len(units):
		return "all units inactive"
	default:
		return ""
	}
}

func crateUnits(name string, mod modules.Module, hasMod bool) []string {
	if units := modules.ResolveUnits(name, mod, hasMod); len(units) > 0 {
		return units
	}
	return nil
}

func crateUnitStates(actual []ServiceState, units []string) []ServiceState {
	states := make([]ServiceState, 0, len(units))
	for _, unit := range units {
		found := false
		for _, svc := range actual {
			if svc.Name == unit {
				states = append(states, svc)
				found = true
				break
			}
		}
		if !found {
			states = append(states, ServiceState{Name: unit, Status: "unknown", Health: "unknown"})
		}
	}
	return states
}

func containsServiceState(states []ServiceState, name string) bool {
	for _, state := range states {
		if state.Name == name {
			return true
		}
	}
	return false
}
