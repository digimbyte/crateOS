package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func (m model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.logSourceCursor = 0
			}
		case "down", "j":
			if m.cursor < len(m.services)-1 {
				m.cursor++
				m.logSourceCursor = 0
			}
		case "right", "l", "tab":
			sources := logSourcesForService(m.currentLogService())
			if m.logSourceCursor < len(sources)-1 {
				m.logSourceCursor++
			}
		case "left", "h", "shift+tab":
			if m.logSourceCursor > 0 {
				m.logSourceCursor--
			}
		}
	}
	return m, nil
}

func (m model) currentLogService() ServiceInfo {
	if len(m.services) == 0 {
		return ServiceInfo{}
	}
	cursor := m.cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.services) {
		cursor = len(m.services) - 1
	}
	return m.services[cursor]
}

func (m model) currentLogSourceIndex() int {
	sources := logSourcesForService(m.currentLogService())
	if len(sources) == 0 {
		return 0
	}
	cursor := m.logSourceCursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(sources) {
		return len(sources) - 1
	}
	return cursor
}

type logSource struct {
	Kind    string
	Scope   string
	Label   string
	Path    string
	Order   int
	Content string
	Tail    bool
}

type logPreview struct {
	Content string
	Tail    bool
}

func (m model) viewLogs() string {
	return renderSplitView(m, "Logs", renderLogServiceSelectionPanel(m), renderLogFocusPanel(m), "  ↑↓ select crate · ←→ / tab select source · [esc] back · [:] command")
}

func renderLogServiceSelectionPanel(m model) string {
	lines := []string{}
	if len(m.services) == 0 {
		lines = append(lines, dim.Render("  No crates available for log inspection."))
		return renderSelectionPanel("LOG TARGETS", 34, lines)
	}
	for i, svc := range m.services {
		name := svc.DisplayName
		if strings.TrimSpace(name) == "" {
			name = svc.Name
		}
		line := fmt.Sprintf(
			"%s %s  %s",
			statusIndicator(svc.Status),
			compactLabel(name, 12),
			renderBadgeRow(selectorStat("s", svc.Status), selectorStat("r", boolToRail(svc.Ready))),
		)
		lines = append(lines, renderSelectorLineWithGlyph(i == m.currentServiceIndex(), serviceRailGlyph(svc), line))
	}
	return renderSelectionPanelWithMeta("LOG TARGETS", "BUS", 34, lines)
}

func renderLogFocusPanel(m model) string {
	if len(m.services) == 0 {
		return renderActivePanel("ACTIVE LOG PANEL", 68, dim.Render("No crate selected."))
	}
	selected := m.currentLogService()
	sources := logSourcesForService(selected)
	sourceCursor := m.currentLogSourceIndex()
	header := renderLogHeader(selected)
	if len(sources) == 0 {
		var b strings.Builder
		b.WriteString(header)
		b.WriteString(renderInsetSelectorCard("SOURCE SELECTOR", "FEEDS", "No log sources available.", nil))
		return renderActivePanel("ACTIVE LOG PANEL", 68, b.String())
	}
	selectedSource := sources[sourceCursor]
	var b strings.Builder
	b.WriteString(header)
	b.WriteString(renderLogSourceSelectorSection(selected, sources, sourceCursor))
	b.WriteString(renderSummaryCard("PREVIEW", renderLogPreviewSection(selected, selectedSource, sourceCursor, len(sources))))
	return renderActivePanel("ACTIVE LOG PANEL", 68, b.String())
}
