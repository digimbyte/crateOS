package tui

import (
	"fmt"
)

func (m model) viewServices() string {
	return renderSplitView(m, "Services", renderServiceSelectionPanel(m), renderServiceFocusPanel(m), "  ↑↓ select crate · [e] enable · [s] start · [x] stop · [d] disable · [esc] back · [:] command · user: "+m.currentUser+" · plane: "+m.controlPlaneMode())
}

func renderServiceSelectionPanel(m model) string {
	lines := []string{}
	if len(m.services) == 0 {
		lines = append(lines, dim.Render("  No services registered."))
		return renderSelectionPanel("CRATE CONTROL", 34, lines)
	}
	for i, s := range m.services {
		nameText := s.DisplayName
		if nameText == "" {
			nameText = s.Name
		}
		line := fmt.Sprintf(
			"%s %s  %s",
			statusIndicator(s.Status),
			compactLabel(nameText, 12),
			renderBadgeRow(selectorStat("s", s.Status), selectorStat("r", boolToRail(s.Ready))),
		)
		lines = append(lines, renderSelectorLineWithGlyph(i == m.cursor, serviceRailGlyph(s), line))
	}
	return renderSelectionPanelWithMeta("CRATE CONTROL", "CRATES", 34, lines)
}

func (m model) currentServiceIndex() int {
	if len(m.services) == 0 {
		return 0
	}
	cursor := m.cursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(m.services) {
		return len(m.services) - 1
	}
	return cursor
}
