package tui

import (
	"fmt"
)

func (m model) viewNetwork() string {
	return renderSplitView(m, "Network Interfaces", renderNetworkSelectionPanel(m), renderNetworkFocusPanel(m), "  ↑↓ select interface · [esc] back · [:] command")
}

func renderNetworkSelectionPanel(m model) string {
	lines := []string{}
	if len(m.interfaces) == 0 {
		lines = append(lines, dim.Render("  No network interfaces detected."))
		return renderSelectionPanel("INTERFACE SELECTOR", 34, lines)
	}
	for i, iface := range m.interfaces {
		line := fmt.Sprintf(
			"%s %s  %s",
			compactLabel(iface.Name, 10),
			selectorStat("lnk", boolToRail(iface.Up)),
			selectorStat("adr", boolToRail(len(iface.Addrs) > 0)),
		)
		lines = append(lines, renderSelectorLineWithGlyph(i == m.currentInterfaceIndex(), networkRailGlyph(iface), line))
	}
	return renderSelectionPanelWithMeta("INTERFACE SELECTOR", "LINKS", 34, lines)
}

func (m model) currentInterfaceIndex() int {
	if len(m.interfaces) == 0 {
		return 0
	}
	cursor := m.cursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(m.interfaces) {
		return len(m.interfaces) - 1
	}
	return cursor
}
