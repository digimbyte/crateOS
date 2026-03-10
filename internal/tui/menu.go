package tui

import (
	"fmt"

	"github.com/crateos/crateos/internal/platform"
)

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
