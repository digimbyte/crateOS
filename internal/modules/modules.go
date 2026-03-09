package modules

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Module is a minimal representation of a packaged module.
type Module struct {
	Metadata struct {
		ID          string `yaml:"id"`
		Name        string `yaml:"name"`
		Category    string `yaml:"category"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"metadata"`
	Spec struct {
		RuntimeType string   `yaml:"runtimeType"`
		InstallMode string   `yaml:"installMode"`
		Units       []string `yaml:"units"`
		Packages    []string `yaml:"packages"`
		Paths       struct {
			Config struct {
				Canonical string `yaml:"canonical"`
				Native    string `yaml:"native"`
			} `yaml:"config"`
			Data struct {
				Canonical string `yaml:"canonical"`
				Native    string `yaml:"native"`
			} `yaml:"data"`
			Logs struct {
				Canonical string `yaml:"canonical"`
				Native    string `yaml:"native"`
			} `yaml:"logs"`
		} `yaml:"paths"`
		HealthChecks []struct {
			Type    string `yaml:"type"`
			Command string `yaml:"command"`
		} `yaml:"healthChecks"`
	} `yaml:"spec"`
}

func (m Module) DisplayName() string {
	if m.Metadata.Name != "" {
		return m.Metadata.Name
	}
	return m.Metadata.ID
}

func (m Module) Runtime() string {
	switch strings.TrimSpace(m.Spec.RuntimeType) {
	case "":
		return "systemd"
	default:
		return m.Spec.RuntimeType
	}
}

func (m Module) InstallMode() string {
	switch strings.TrimSpace(m.Spec.InstallMode) {
	case "":
		return "immediate"
	default:
		return m.Spec.InstallMode
	}
}

func (m Module) UnitNames() []string {
	if len(m.Spec.Units) == 0 {
		return nil
	}
	return append([]string(nil), m.Spec.Units...)
}

func ResolveRuntime(fallback string, mod Module, hasMod bool) string {
	if hasMod {
		return mod.Runtime()
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "systemd"
}

func ResolveInstallMode(mod Module, hasMod bool) string {
	if hasMod {
		return mod.InstallMode()
	}
	return ""
}

func ResolveUnits(name string, mod Module, hasMod bool) []string {
	if hasMod {
		if units := mod.UnitNames(); len(units) > 0 {
			return units
		}
		return nil
	}
	return []string{name}
}

// LoadAll loads module definitions from packaging/modules/*.module.yaml.
func LoadAll(root string) map[string]Module {
	out := make(map[string]Module)
	glob := filepath.Join(root, "packaging", "modules", "*.module.yaml")
	matches, _ := filepath.Glob(glob)
	for _, path := range matches {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m Module
		if err := yaml.Unmarshal(b, &m); err != nil {
			continue
		}
		if m.Metadata.ID == "" {
			continue
		}
		m.Spec.RuntimeType = m.Runtime()
		m.Spec.InstallMode = m.InstallMode()
		out[m.Metadata.ID] = m
	}
	return out
}
