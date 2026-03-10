package tui

import "strings"

func renderServiceFocusPanel(m model) string {
	if len(m.services) == 0 {
		return renderActivePanel("ACTIVE CRATE", 68, dim.Render("No crate selected."))
	}
	s := m.services[m.currentServiceIndex()]
	nameText := s.DisplayName
	if nameText == "" {
		nameText = s.Name
	}
	var header strings.Builder
	header.WriteString(renderPanelTitleBar(strings.ToUpper(nameText), "CRATE"))
	header.WriteString("\n")
	header.WriteString(renderStatStrip(
		statusBadge(s.Status),
		healthBadge(s.Health),
		typeBadge(s.Type),
	))
	header.WriteString("\n")
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(renderSummaryCard("SYSTEM POSTURE", renderServicePostureSection(m.services)))
	b.WriteString(renderSummaryCard("LIFECYCLE", renderServiceLifecycleSection(s)))
	b.WriteString(renderSubsectionCard("UNIT GRID", renderServiceUnitGrid(s)))
	b.WriteString(renderActionCard("LEGEND", renderServiceLegend()))
	return renderActivePanel("ACTIVE CRATE", 68, b.String())
}
