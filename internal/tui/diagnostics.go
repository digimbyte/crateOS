package tui

import (
	"fmt"
	"strings"
)

func diagnosticsPanelItems() []string {
	return []string{"Summary", "Verification", "Ownership", "Config"}
}

func compactOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func diagnosticsPanelTitle(cursor int) string {
	items := diagnosticsPanelItems()
	if cursor < 0 || cursor >= len(items) {
		return items[0]
	}
	return items[cursor]
}

func (m model) viewDiagnostics() string {
	hint := "  ↑↓ select section · [1-4] jump · [esc] back · [:] command"
	if m.statusSection == 2 {
		hint = "  ↑↓ browse workloads/details · [h/l] move workload · [1-4] jump section · [esc] back · [:] command"
	}
	return renderSplitView(m, "Diagnostics", renderDiagnosticsSelectionPanel(m), renderDiagnosticsFocusPanel(m), hint)
}

func renderDiagnosticsSelectionPanel(m model) string {
	lines := make([]string, 0, len(diagnosticsPanelItems()))
	for i, item := range diagnosticsPanelItems() {
		line := fmt.Sprintf("[%d] %s", i+1, diagnosticsSelectorLabel(m, item))
		lines = append(lines, renderSelectorLineWithGlyph(i == m.statusSection, diagnosticsRailGlyph(item), line))
	}
	return renderSelectionPanelWithMeta("DIAGNOSTICS", "SURFACES", 30, lines)
}

func renderDiagnosticsFocusPanel(m model) string {
	switch diagnosticsPanelTitle(m.statusSection) {
	case "Verification":
		return renderDiagnosticsVerificationPanel(m)
	case "Ownership":
		return renderDiagnosticsOwnershipPanel(m)
	case "Config":
		return renderDiagnosticsConfigPanel(m)
	default:
		return renderDiagnosticsSummaryPanel(m)
	}
}

func currentOwnershipSelection(m model) int {
	if len(m.diagnostics.Ownership.Workloads) == 0 {
		return 1
	}
	return m.ownershipCursor + 1
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
