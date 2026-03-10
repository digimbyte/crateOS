package api

import "github.com/crateos/crateos/internal/modules"

type serviceAction string

const (
	serviceActionEnableOnly serviceAction = "enable-only"
	serviceActionDisable    serviceAction = "disable"
	serviceActionStart      serviceAction = "start"
	serviceActionStop       serviceAction = "stop"
)

func applyServiceAction(name string, action serviceAction, mods map[string]modules.Module) {
	targets := []string{name}
	if mod, ok := mods[name]; ok {
		if units := modules.ResolveUnits(name, mod, true); len(units) > 0 {
			targets = units
		}
	}
	for _, target := range targets {
		switch action {
		case serviceActionEnableOnly:
			systemctlNoError("enable", target)
		case serviceActionDisable:
			systemctlNoError("stop", target)
			systemctlNoError("disable", target)
		case serviceActionStart:
			systemctlNoError("enable", target)
			systemctlNoError("start", target)
		case serviceActionStop:
			systemctlNoError("stop", target)
		}
	}
}

func shouldAutostartOnEnable(name string, mods map[string]modules.Module) bool {
	if mod, ok := mods[name]; ok {
		return mod.InstallMode() != "staged"
	}
	return true
}
