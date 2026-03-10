package tui

import (
	"fmt"
	"strings"
)

func renderMenuFocusPanel(m model) string {
	var b strings.Builder
	selected := menuItemTitle(m.cursor)
	b.WriteString(renderPanelTitleBar(strings.ToUpper(selected), "ROUTE"))
	b.WriteString("\n")
	if summary := menuSelectionSummary(m.cursor); summary != "" {
		b.WriteString(dim.Render(summary + "\n"))
	}
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("SYSTEM SNAPSHOT", menuSystemSnapshotCard(m)))
	b.WriteString(renderSummaryCard("CONTROL POSTURE", menuControlPostureCard(m)))
	b.WriteString(renderWarningCard("ATTENTION", menuAttentionCard(m)))
	b.WriteString(renderActionCard("QUICK ACCESS", renderPanelLines(
		dim.Render(menuSelectionHint(m.cursor)),
		dim.Render(menuSelectionAction(m.cursor)),
	)))
	return renderActivePanel("ACTIVE PANEL", 64, b.String())
}

func menuSystemSnapshotCard(m model) string {
	failed, partial, staged, healthy := menuServiceCounts(m.services)
	readyAdapters, failedAdapters := menuPlatformCounts(m.platform.Adapters)
	var snapshot strings.Builder
	snapshot.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("crates:%d", len(m.services))),
		danger.Render(fmt.Sprintf("failed:%d", failed)),
		warn.Render(fmt.Sprintf("partial:%d", partial)),
		dim.Render(fmt.Sprintf("staged:%d", staged)),
		ok.Render(fmt.Sprintf("healthy:%d", healthy)),
	))
	snapshot.WriteString("\n")
	snapshot.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("adapters:%d", len(m.platform.Adapters))),
		ok.Render(fmt.Sprintf("ready:%d", readyAdapters)),
		danger.Render(fmt.Sprintf("failed:%d", failedAdapters)),
	))
	snapshot.WriteString("\n")
	snapshot.WriteString(renderStatStrip(
		selectorStat("svc", len(m.services)),
		selectorStat("usr", len(m.users)),
		selectorStat("net", len(m.interfaces)),
		selectorStat("adp", len(m.platform.Adapters)),
	))
	return snapshot.String()
}

func menuControlPostureCard(m model) string {
	failed, _, _, healthy := menuServiceCounts(m.services)
	_, failedAdapters := menuPlatformCounts(m.platform.Adapters)
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
	return posture.String()
}

func menuAttentionCard(m model) string {
	issues := menuTopIssues(m.services, 2)
	issues = append(issues, menuTopPlatformIssues(m.platform.Adapters, 2)...)
	if len(issues) > 2 {
		issues = issues[:2]
	}
	var attention strings.Builder
	if report, readinessOK := readReadinessReport(); readinessOK && strings.TrimSpace(report.Status) == "degraded" {
		attention.WriteString(danger.Render("readiness:degraded"))
		attention.WriteString("\n")
		if len(report.Failures) > 0 {
			attention.WriteString(renderBulletLines(report.Failures[:minInt(len(report.Failures), 3)]))
		} else if strings.TrimSpace(report.Summary) != "" {
			attention.WriteString(renderBulletLines([]string{report.Summary}))
		}
		return attention.String()
	}
	if len(issues) == 0 {
		attention.WriteString(ok.Render("issues:none"))
		return attention.String()
	}
	attention.WriteString(danger.Render(fmt.Sprintf("issues:%d", len(issues))))
	attention.WriteString("\n")
	attention.WriteString(renderBulletLines(issues))
	return attention.String()
}
