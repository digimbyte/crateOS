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
	if report, readinessOK := readReadinessReport(); readinessOK {
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
	if report, readinessOK := readReadinessReport(); readinessOK && strings.TrimSpace(report.Status) == "degraded" {
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
