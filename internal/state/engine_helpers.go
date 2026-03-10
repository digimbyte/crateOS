package state

import (
	"os"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
)

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func desiredManagedUnits(desired config.ServiceEntry, mod modules.Module, hasMod bool) []string {
	if hasMod {
		return crateUnits(desired.Name, mod, hasMod)
	}
	if executionAdapterForRuntime(desired.Runtime) != "systemd" {
		if strings.TrimSpace(desired.Name) == "" {
			return nil
		}
		return []string{desired.Name}
	}
	primary, companion := inferExecutionUnits(desired, mod, hasMod)
	units := []string{}
	if primary != "" {
		units = append(units, primary)
	}
	if companion != "" {
		units = append(units, companion)
	}
	if len(units) == 0 && strings.TrimSpace(desired.Name) != "" {
		units = append(units, desired.Name)
	}
	return units
}

func hasModule(mods map[string]modules.Module, name string) bool {
	_, ok := mods[name]
	return ok
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}
