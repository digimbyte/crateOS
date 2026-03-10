package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/state"
)

func enableServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = shouldAutostartOnEnable(name, mods)
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			applyServiceAction(name, serviceActionEnableOnly, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func disableServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = false
			cfg.Services.Services[i].Autostart = false
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			applyServiceAction(name, serviceActionDisable, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func startServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = true
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			applyServiceAction(name, serviceActionStart, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func stopServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = false
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			applyServiceAction(name, serviceActionStop, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

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

func systemctlNoError(action, unit string) {
	if runtime.GOOS != "linux" {
		return
	}
	_ = exec.Command("systemctl", action, unit).Run()
}
