package state

import (
	"fmt"
	"os/exec"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
)

func reconcileModuleUnits(desired config.ServiceEntry, mod modules.Module, hasMod bool, unitStates []ServiceState) []Action {
	if !hasMod || mod.Runtime() != "systemd" {
		return nil
	}

	var actions []Action
	for _, pkg := range mod.Spec.Packages {
		if !packageInstalled(pkg) {
			if err := installPackage(pkg); err == nil {
				actions = append(actions, Action{
					Description: fmt.Sprintf("installed package %s for %s", pkg, desired.Name),
					Component:   "package",
					Target:      pkg,
				})
			}
		}
	}
	for _, unit := range modules.ResolveUnits(desired.Name, mod, true) {
		if !containsServiceState(unitStates, unit) {
			_ = systemctl("enable", unit)
			if mod.InstallMode() != "staged" || desired.Autostart {
				_ = systemctl("start", unit)
			}
			actions = append(actions, Action{
				Description: fmt.Sprintf("ensured module unit %s for %s", unit, desired.Name),
				Component:   "service",
				Target:      unit,
			})
		}
	}
	for _, hc := range mod.Spec.HealthChecks {
		if hc.Type == "command" && hc.Command != "" {
			if err := exec.Command("sh", "-c", hc.Command).Run(); err != nil {
				actions = append(actions, Action{
					Description: fmt.Sprintf("health check failed for %s: %s", desired.Name, hc.Command),
					Component:   "health",
					Target:      desired.Name,
				})
			}
		}
	}
	return actions
}
