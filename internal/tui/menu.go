package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/platform"
)

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter":
			return m.selectMenuItem()
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.enterStatusView()
		case "2":
			m.enterServicesView()
		case "3":
			m.enterDiagnosticsView()
		case "4":
			m.enterUsersView()
		case "5":
			m.enterLogsView()
		case "6":
			m.enterNetworkView()
		}
	}
	return m, nil
}

func (m model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		m.enterStatusView()
	case 1:
		m.enterServicesView()
	case 2:
		m.enterDiagnosticsView()
	case 3:
		m.enterUsersView()
	case 4:
		m.enterLogsView()
	case 5:
		m.enterNetworkView()
	case 6:
		m.quitting = true
		return m, tea.Quit
	}
	m.cursor = 0
	return m, nil
}

func (m model) viewMenu() string {
	header := fmt.Sprintf(
		"C R A T E O S   %s\n%s · %s/%s",
		platform.Version, m.info.Hostname, m.info.OS, m.info.Arch,
	)
	return renderSplitView(m, header, renderMenuSelectionPanel(m), renderMenuFocusPanel(m), "  ↑↓ navigate · enter select · [1-6] jump · q quit · [:] command")
}

func renderMenuSelectionPanel(m model) string {
	lines := make([]string, 0, len(menuItems))
	for i, item := range menuItems {
		hotkey := fmt.Sprintf("[%d]", i+1)
		if i == len(menuItems)-1 {
			hotkey = "[Q]"
		}
		line := fmt.Sprintf("%s %s", hotkey, menuSelectorLabel(m, i))
		lines = append(lines, renderSelectorLineWithGlyph(i == m.cursor, menuRailGlyph(item), line))
	}
	return renderSelectionPanelWithMeta("CONTROL MENU", "ROUTES", 30, lines)
}

func renderMenuFocusPanel(m model) string {
	var b strings.Builder
	selected := menuItemTitle(m.cursor)
	b.WriteString(renderPanelTitleBar(strings.ToUpper(selected), "ROUTE"))
	b.WriteString("\n")
	if summary := menuSelectionSummary(m.cursor); summary != "" {
		b.WriteString(dim.Render(summary + "\n"))
	}
	b.WriteString("\n")
	failed, partial, staged, healthy := menuServiceCounts(m.services)
	var snapshot strings.Builder
	snapshot.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("crates:%d", len(m.services))),
		danger.Render(fmt.Sprintf("failed:%d", failed)),
		warn.Render(fmt.Sprintf("partial:%d", partial)),
		dim.Render(fmt.Sprintf("staged:%d", staged)),
		ok.Render(fmt.Sprintf("healthy:%d", healthy)),
	))
	snapshot.WriteString("\n")
	readyAdapters, failedAdapters := menuPlatformCounts(m.platform.Adapters)
	snapshot.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("adapters:%d", len(m.platform.Adapters))),
		ok.Render(fmt.Sprintf("ready:%d", readyAdapters)),
		danger.Render(fmt.Sprintf("failed:%d", failedAdapters)),
	))
	snapshot.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("svc", len(m.services)),
		selectorStat("usr", len(m.users)),
		selectorStat("net", len(m.interfaces)),
		selectorStat("adp", len(m.platform.Adapters)),
	))
	b.WriteString(renderSummaryCard("SYSTEM SNAPSHOT", snapshot.String()))

	issues := menuTopIssues(m.services, 2)
	issues = append(issues, menuTopPlatformIssues(m.platform.Adapters, 2)...)
	if len(issues) > 2 {
		issues = issues[:2]
	}
	var posture strings.Builder
	posture.WriteString(renderBadgeRow(
		ok.Render(fmt.Sprintf("healthy:%d", healthy)),
		danger.Render(fmt.Sprintf("faults:%d", failed+failedAdapters)),
		warn.Render(fmt.Sprintf("operators:%d", len(m.users))),
		dim.Render(fmt.Sprintf("links:%d", len(m.interfaces))),
		selectorStat("plane", m.controlPlaneMode()),
	))
	posture.WriteString("\n")
	if report, ok := readReadinessReport(); ok {
		posture.WriteString(renderBadgeRow(
			selectorStat("ready", strings.TrimSpace(report.Status)),
		))
		posture.WriteString("\n")
		if strings.TrimSpace(report.CheckedAt) != "" {
			posture.WriteString(dim.Render("checked: " + strings.TrimSpace(report.CheckedAt)))
			posture.WriteString("\n")
		}
		if strings.TrimSpace(report.Summary) != "" {
			posture.WriteString(dim.Render("readiness: " + report.Summary))
			posture.WriteString("\n")
		}
	}
	posture.WriteString(dim.Render("Route from this panel into diagnostics, crate control, operator control, and network posture."))
	posture.WriteString("\n")
	b.WriteString(renderSummaryCard("CONTROL POSTURE", posture.String()))
	var attention strings.Builder
	if report, ok := readReadinessReport(); ok && strings.TrimSpace(report.Status) == "degraded" {
		attention.WriteString(danger.Render("readiness:degraded"))
		attention.WriteString("\n")
		if len(report.Failures) > 0 {
			attention.WriteString(renderBulletLines(report.Failures[:minInt(len(report.Failures), 3)]))
		} else if strings.TrimSpace(report.Summary) != "" {
			attention.WriteString(renderBulletLines([]string{report.Summary}))
		}
	} else if len(issues) == 0 {
		attention.WriteString(ok.Render("issues:none"))
	} else {
		attention.WriteString(danger.Render(fmt.Sprintf("issues:%d", len(issues))))
		attention.WriteString("\n")
		attention.WriteString(renderBulletLines(issues))
	}
	b.WriteString(renderWarningCard("ATTENTION", attention.String()))
	b.WriteString(renderActionCard("QUICK ACCESS", renderPanelLines(
		dim.Render(menuSelectionHint(m.cursor)),
		dim.Render(menuSelectionAction(m.cursor)),
	)))
	return renderActivePanel("ACTIVE PANEL", 64, b.String())
}

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
