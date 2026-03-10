package tui

import (
	"fmt"
	"strings"
)

func menuItemTitle(cursor int) string {
	if cursor < 0 || cursor >= len(menuItems) {
		return "System Status"
	}
	return menuItems[cursor]
}

func menuSelectionSummary(cursor int) string {
	switch menuItemTitle(cursor) {
	case "System Status":
		return "Review machine state, crate readiness, and platform adapter health."
	case "Services":
		return "Inspect crate runtime state and execute service lifecycle actions."
	case "Diagnostics":
		return "Inspect verification posture, diagnostic ledgers, and operator-facing drift surfaces."
	case "Users":
		return "Manage local users, roles, and permission assignments."
	case "Logs":
		return "Browse crate journals and exported logs from one terminal surface."
	case "Network":
		return "Inspect interfaces and network-facing platform state."
	case "Exit":
		return "Leave the CrateOS control panel."
	default:
		return ""
	}
}

func menuRailGlyph(item string) string {
	switch item {
	case "System Status":
		return "◆"
	case "Services":
		return "■"
	case "Diagnostics":
		return "▣"
	case "Users":
		return "◈"
	case "Logs":
		return "◬"
	case "Network":
		return "◉"
	case "Exit":
		return "×"
	default:
		return "•"
	}
}

func menuSelectionAction(cursor int) string {
	switch menuItemTitle(cursor) {
	case "System Status":
		return "route: sys / svc / platform"
	case "Services":
		return "route: lifecycle control"
	case "Diagnostics":
		return "route: verification / ledgers / drift"
	case "Users":
		return "route: accounts and roles"
	case "Logs":
		return "route: source and preview"
	case "Network":
		return "route: iface and posture"
	case "Exit":
		return "route: terminate session"
	default:
		return ""
	}
}

func menuSelectorLabel(m model, cursor int) string {
	switch menuItemTitle(cursor) {
	case "System Status":
		readyAdapters, _ := menuPlatformCounts(m.platform.Adapters)
		return renderBadgeRow(
			"SYS",
			selectorStat("plat", fmt.Sprintf("%d/%d", readyAdapters, len(m.platform.Adapters))),
		)
	case "Services":
		failed, partial, staged, _ := menuServiceCounts(m.services)
		return renderBadgeRow(
			"SVC",
			selectorStat("f", failed),
			selectorStat("p", partial),
			selectorStat("s", staged),
		)
	case "Diagnostics":
		return renderBadgeRow(
			"DIA",
			selectorStat("vrf", strings.TrimSpace(m.diagnostics.Verification.Status)),
			selectorStat("cfg", m.diagnostics.Config.Tracked),
			selectorStat("u", m.diagnostics.Config.Unmonitored),
		)
	case "Users":
		return renderBadgeRow("USR", selectorStat("ops", len(m.users)))
	case "Logs":
		return renderBadgeRow("LOG", selectorStat("src", len(m.services)))
	case "Network":
		return renderBadgeRow("NET", selectorStat("if", len(m.interfaces)))
	case "Exit":
		return "EXIT CONTROL PANEL"
	default:
		return strings.ToUpper(menuItemTitle(cursor))
	}
}

func menuSelectionHint(cursor int) string {
	switch menuItemTitle(cursor) {
	case "System Status":
		return "Press Enter to open the full system and adapter diagnostics panel."
	case "Diagnostics":
		return "Press Enter to inspect MVP verification posture, ledgers, and config drift surfaces."
	case "Services":
		return "Press Enter to open crate control. Use enable/start/stop/disable actions there."
	case "Users":
		return "Press Enter to open user administration with add/edit/delete controls."
	case "Logs":
		return "Press Enter to inspect journals, crate logs, and source previews."
	case "Network":
		return "Press Enter to inspect interfaces and network-related state."
	case "Exit":
		return "Press Enter or Q to exit."
	default:
		return ""
	}
}

func menuServiceCounts(services []ServiceInfo) (failed, partial, staged, healthy int) {
	for _, s := range services {
		switch s.Status {
		case "failed":
			failed++
		case "partial":
			partial++
		case "staged":
			staged++
		}
		if s.Ready && s.Health == "ok" {
			healthy++
		}
	}
	return failed, partial, staged, healthy
}

func menuTopIssues(services []ServiceInfo, limit int) []string {
	issues := make([]string, 0, limit)
	for _, s := range services {
		if issue := crateIssueLine(s); issue != "" {
			issues = append(issues, issue)
			if len(issues) >= limit {
				break
			}
		}
	}
	return issues
}

func menuPlatformCounts(adapters []PlatformAdapter) (ready, failed int) {
	for _, adapter := range adapters {
		switch adapter.Status {
		case "ready":
			ready++
		case "failed":
			failed++
		}
	}
	return ready, failed
}

func menuTopPlatformIssues(adapters []PlatformAdapter, limit int) []string {
	issues := make([]string, 0, limit)
	for _, adapter := range adapters {
		if issue := platformIssueLine(adapter); issue != "" {
			issues = append(issues, issue)
			if len(issues) >= limit {
				break
			}
		}
	}
	return issues
}
