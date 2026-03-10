package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func (m model) viewStatus() string {
	return renderSplitView(m, "System Status", renderStatusSelectionPanel(m), renderStatusFocusPanel(m), "  ↑↓ select section · [1-3] jump · [esc] back · [:] command")
}

func renderStatusSelectionPanel(m model) string {
	lines := make([]string, 0, len(statusPanelItems()))
	for i, item := range statusPanelItems() {
		line := fmt.Sprintf("[%d] %s", i+1, statusSelectorLabel(m, item))
		lines = append(lines, renderSelectorLineWithGlyph(i == m.statusSection, statusRailGlyph(item), line))
	}
	return renderSelectionPanelWithMeta("STATUS MODULES", "SCAN", 30, lines)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func statusPanelItems() []string {
	return []string{"System", "Services", "Platform"}
}

func statusPanelTitle(cursor int) string {
	items := statusPanelItems()
	if cursor < 0 || cursor >= len(items) {
		return items[0]
	}
	return items[cursor]
}

func statusCountText(count int) string {
	if count == 0 {
		return ok.Render("0")
	}
	return warn.Render(fmt.Sprintf("%d", count))
}
