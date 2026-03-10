package state

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

func reconcileAccess(cfg config.CrateOSConfig) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	adapter := platformAdapterState("access", "Access", true)
	renderedPath := platform.CratePath("state", "rendered", "access.json")
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)

	validationIssues, summary := validateAccessConfig(cfg)
	if len(validationIssues) > 0 {
		adapter.Validation = "failed"
		adapter.ValidationErr = strings.Join(validationIssues, "; ")
		adapter.Apply = "blocked"
		issues = append(issues, adapter.ValidationErr)
	} else {
		adapter.Validation = "ok"
		adapter.Apply = "ok"
	}

	localGUIProvider := strings.TrimSpace(cfg.Access.LocalGUI.Provider)
	virtualDesktopProvider := strings.TrimSpace(cfg.Access.VirtualDesktop.Provider)
	if runtime.GOOS == "linux" {
		switch {
		case cfg.Access.LocalGUI.Enabled && strings.EqualFold(localGUIProvider, "lightdm") && !packageInstalled("lightdm"):
			issues = append(issues, "local gui enabled but lightdm package is not installed")
		case cfg.Access.VirtualDesktop.Enabled && virtualDesktopProvider != "" && !isSupportedVirtualDesktopProvider(virtualDesktopProvider):
			issues = append(issues, "virtual desktop provider is not yet supported by CrateOS session adapters")
		}
	}

	stateSummary := map[string]interface{}{
		"ssh": map[string]interface{}{
			"enabled": cfg.Access.SSH.Enabled,
			"landing": normalizeLanding(cfg.Access.SSH.Landing),
		},
		"local_gui": map[string]interface{}{
			"enabled":       cfg.Access.LocalGUI.Enabled,
			"provider":      normalizeLocalGUIProvider(cfg.Access.LocalGUI.Provider),
			"landing":       normalizeLanding(cfg.Access.LocalGUI.Landing),
			"default_shell": normalizeDefaultShell(cfg.Access.LocalGUI.DefaultShell),
		},
		"virtual_desktop": map[string]interface{}{
			"enabled":  cfg.Access.VirtualDesktop.Enabled,
			"provider": normalizeVirtualDesktopProvider(cfg.Access.VirtualDesktop.Provider),
			"landing":  normalizeLanding(cfg.Access.VirtualDesktop.Landing),
		},
		"break_glass": map[string]interface{}{
			"enabled":            cfg.Access.BreakGlass.Enabled,
			"require_permission": normalizeBreakGlassPermission(cfg.Access.BreakGlass.RequirePerm),
			"allowed_surfaces":   normalizeAllowedSurfaces(cfg.Access.BreakGlass.AllowedSurfaces),
		},
		"summary": summary,
		"validation": map[string]interface{}{
			"status": adapter.Validation,
			"error":  adapter.ValidationErr,
		},
	}
	if data, err := json.MarshalIndent(stateSummary, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"access/state.json",
			renderedPath,
			string(data)+"\n",
			"access",
			"rendered access and session state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}

	adapter.Summary = summary
	return actions, finalizePlatformAdapterState(adapter, issues)
}

func validateAccessConfig(cfg config.CrateOSConfig) ([]string, string) {
	issues := []string{}
	enabledSurfaces := 0
	if cfg.Access.SSH.Enabled {
		enabledSurfaces++
	}
	if cfg.Access.LocalGUI.Enabled {
		enabledSurfaces++
	}
	if cfg.Access.VirtualDesktop.Enabled {
		enabledSurfaces++
	}
	if enabledSurfaces == 0 {
		issues = append(issues, "at least one controlled entry surface must remain enabled")
	}

	sshLanding := normalizeLanding(cfg.Access.SSH.Landing)
	if cfg.Access.SSH.Enabled && sshLanding != "console" {
		issues = append(issues, "ssh landing must remain console")
	}

	localLanding := normalizeLanding(cfg.Access.LocalGUI.Landing)
	if cfg.Access.LocalGUI.Enabled {
		if normalizeLocalGUIProvider(cfg.Access.LocalGUI.Provider) != "lightdm" {
			issues = append(issues, "local_gui provider must be lightdm when enabled")
		}
		if localLanding == "shell" || localLanding == "desktop" {
			issues = append(issues, "local_gui landing must stay inside CrateOS-owned surfaces")
		}
		if normalizeDefaultShell(cfg.Access.LocalGUI.DefaultShell) != "crateos-session" {
			issues = append(issues, "local_gui default_shell must be crateos-session")
		}
	}

	virtualLanding := normalizeLanding(cfg.Access.VirtualDesktop.Landing)
	if cfg.Access.VirtualDesktop.Enabled && (virtualLanding == "shell" || virtualLanding == "desktop") {
		issues = append(issues, "virtual_desktop landing must stay inside CrateOS-owned surfaces")
	}

	breakGlassPerm := normalizeBreakGlassPermission(cfg.Access.BreakGlass.RequirePerm)
	if cfg.Access.BreakGlass.Enabled && breakGlassPerm == "" {
		issues = append(issues, "break_glass enabled requires require_permission")
	}

	allowedSurfaces := normalizeAllowedSurfaces(cfg.Access.BreakGlass.AllowedSurfaces)
	for _, surface := range allowedSurfaces {
		switch surface {
		case "ssh", "local_gui", "virtual_desktop":
		default:
			issues = append(issues, fmt.Sprintf("break_glass allowed surface %q is unsupported", surface))
		}
	}

	summaryParts := []string{}
	if cfg.Access.SSH.Enabled {
		summaryParts = append(summaryParts, "ssh→"+sshLanding)
	}
	if cfg.Access.LocalGUI.Enabled {
		summaryParts = append(summaryParts, "local_gui→"+localLanding)
	}
	if cfg.Access.VirtualDesktop.Enabled {
		summaryParts = append(summaryParts, "virtual_desktop→"+virtualLanding)
	}
	summary := "controlled entry surfaces: " + strings.Join(summaryParts, ", ")
	if cfg.Access.BreakGlass.Enabled {
		summary += "; break-glass gated by " + breakGlassPerm
	} else {
		summary += "; break-glass disabled"
	}
	return issues, summary
}

func normalizeLanding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "console":
		return "console"
	case "panel":
		return "panel"
	case "workspace":
		return "workspace"
	case "recovery":
		return "recovery"
	case "shell":
		return "shell"
	case "desktop":
		return "desktop"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeLocalGUIProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "lightdm":
		return "lightdm"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeVirtualDesktopProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDefaultShell(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "crateos-session":
		return "crateos-session"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeBreakGlassPermission(value string) string {
	return strings.TrimSpace(value)
}

func normalizeAllowedSurfaces(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isSupportedVirtualDesktopProvider(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return true
	default:
		return false
	}
}
