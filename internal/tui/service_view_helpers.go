package tui

import (
	"fmt"
	"strings"
)

func statusIndicator(status string) string {
	switch status {
	case "active", "running":
		return ok.Render("●")
	case "inactive", "dead":
		return dim.Render("○")
	case "failed":
		return danger.Render("✖")
	case "partial", "activating":
		return warn.Render("◐")
	case "staged":
		return dim.Render("◌")
	default:
		return warn.Render("?")
	}
}

func valueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func serviceRailGlyph(s ServiceInfo) string {
	switch s.Status {
	case "failed":
		return "✖"
	case "partial":
		return "◐"
	case "active", "running":
		return "●"
	case "staged":
		return "◌"
	default:
		return "○"
	}
}

func lifecycleDesiredBadge(s ServiceInfo) string {
	if s.Desired {
		return ok.Render("desired:on")
	}
	return dim.Render("desired:off")
}

func lifecycleAutostartBadge(s ServiceInfo) string {
	if !s.Desired {
		return dim.Render("autostart:n/a")
	}
	if s.Autostart {
		return ok.Render("autostart:on")
	}
	return warn.Render("autostart:off")
}

func lifecycleRuntimeBadge(s ServiceInfo) string {
	switch {
	case !s.Desired:
		return dim.Render("runtime:disabled")
	case s.Status == "failed":
		return danger.Render("runtime:failed")
	case s.Status == "partial":
		return warn.Render("runtime:partial")
	case s.Active:
		return ok.Render("runtime:running")
	case s.Status == "staged":
		return dim.Render("runtime:waiting")
	case s.Status == "unknown":
		return warn.Render("runtime:unknown")
	default:
		return warn.Render("runtime:stopped")
	}
}

func lifecycleIntentText(s ServiceInfo) string {
	switch {
	case s.ExecutionMode == "job" && s.Schedule != "":
		return "crate is modeled as a scheduled job and should run on the declared cadence"
	case !s.Desired:
		return "crate is disabled and should not be running"
	case s.Status == "failed":
		return "crate is enabled but one or more units have failed"
	case s.Status == "partial":
		return "crate is enabled but only part of the unit set is running cleanly"
	case s.Status == "staged":
		return "crate is enabled but waiting for an explicit start"
	case s.Autostart && s.Active:
		return "crate is enabled and should be kept running automatically"
	case s.Autostart && !s.Active:
		return "crate is enabled for automatic runtime but is not currently running"
	case !s.Autostart && s.Active:
		return "crate is enabled and currently running without automatic restart intent"
	default:
		return "crate is enabled but intentionally stopped until started again"
	}
}

func statusBadge(status string) string {
	text := "state:" + status
	switch status {
	case "active", "running", "ready":
		return ok.Render(text)
	case "inactive", "dead", "disabled", "staged":
		return dim.Render(text)
	case "failed":
		return danger.Render(text)
	case "partial", "activating", "unknown":
		return warn.Render(text)
	default:
		return warn.Render(text)
	}
}

func healthBadge(health string) string {
	if strings.TrimSpace(health) == "" {
		return dim.Render("health:n/a")
	}
	text := "health:" + health
	switch health {
	case "ok", "healthy", "ready":
		return ok.Render(text)
	case "warn", "warning", "degraded":
		return warn.Render(text)
	case "fail", "failed", "error", "critical":
		return danger.Render(text)
	default:
		return dim.Render(text)
	}
}

func typeBadge(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return dim.Render("type:unknown")
	}
	return dim.Render("type:" + kind)
}

func lifecycleUnitCounts(s ServiceInfo) string {
	if len(s.Units) == 0 {
		return "no unit detail available"
	}
	running := 0
	enabled := 0
	failed := 0
	for _, unit := range s.Units {
		if unit.Active {
			running++
		}
		if unit.Enabled {
			enabled++
		}
		if unit.Status == "failed" {
			failed++
		}
	}
	return fmt.Sprintf("%d/%d running · %d/%d enabled · %d failed", running, len(s.Units), enabled, len(s.Units), failed)
}

func readyBadge(ready bool) string {
	if ready {
		return ok.Render("ready:on")
	}
	return danger.Render("ready:off")
}

func boolToRail(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
