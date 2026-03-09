package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func renderSplitView(m model, title, left, right, commands string) string {
	var b strings.Builder
	b.WriteString(renderOperationalHeader(title))
	b.WriteString("\n")
	b.WriteString(renderShellStatusBar(
		selectorStat("left", "rail"),
		selectorStat("right", "focus"),
		selectorStat("mode", "cpanel"),
		selectorStat("plane", m.controlPlaneMode()),
	))
	b.WriteString("\n")
	b.WriteString(renderShellFrame(lipgloss.JoinHorizontal(lipgloss.Top, left, right)))
	b.WriteString("\n")
	b.WriteString(renderCommandStrip(m, commands))
	return b.String()
}

func renderOperationalHeader(title string) string {
	lines := strings.Split(title, "\n")
	main := ""
	if len(lines) > 0 {
		main = strings.TrimSpace(lines[0])
	}
	meta := ""
	if len(lines) > 1 {
		meta = strings.TrimSpace(lines[1])
	}
	var b strings.Builder
	if main != "" {
		b.WriteString(headerTitle.Render(main))
	}
	if meta != "" {
		b.WriteString("\n")
		parts := strings.Split(meta, "·")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if i > 0 {
				b.WriteString(headerSubmeta.Render("  ·  "))
			}
			b.WriteString(headerMeta.Render(part))
		}
	}
	return headerBox.Render(b.String())
}

func renderSelectionPanel(title string, width int, lines []string) string {
	return renderSelectionPanelWithMeta(title, "SELECT", width, lines)
}

func renderSelectionPanelWithMeta(title, meta string, width int, lines []string) string {
	var b strings.Builder
	b.WriteString(renderRailTitleBar(title, meta))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return panelBox.Width(width).Render(b.String())
}

func renderSelectorLine(selected bool, line string) string {
	return renderSelectorLineWithGlyph(selected, "•", line)
}

func renderSelectorLineWithGlyph(selected bool, glyph, line string) string {
	if selected {
		return selectorActive.Render("▌ " + glyph + " " + line)
	}
	return selectorIdle.Render("│ " + glyph + " " + line)
}

func renderActivePanel(title string, width int, body string) string {
	var b strings.Builder
	b.WriteString(renderPanelTitleBar(title, "ACTIVE"))
	b.WriteString("\n\n")
	b.WriteString(body)
	return panelActive.Width(width).Render(b.String())
}

func renderPanelSection(title string) string {
	return "\n" + subsectionTitle.Render(title) + "\n"
}

func renderPanelLines(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderPanelKV(labelText, valueText string) string {
	return label.Render(labelText) + value.Render(valueText)
}

func renderBulletLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		rendered = append(rendered, dim.Render("- "+line))
	}
	return renderPanelLines(rendered...)
}

func renderCountKV(labelText string, count int) string {
	return renderPanelKV(labelText, fmt.Sprintf("%d", count))
}

func renderSubsectionCard(title, body string) string {
	var b strings.Builder
	b.WriteString(subsectionTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	return subsectionCard.Render(b.String())
}

func renderSummaryCard(title, body string) string {
	var b strings.Builder
	b.WriteString(subsectionTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	return subsectionCardSummary.Render(b.String())
}

func renderWarningCard(title, body string) string {
	var b strings.Builder
	b.WriteString(danger.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	return subsectionCardWarning.Render(b.String())
}

func renderActionCard(title, body string) string {
	var b strings.Builder
	b.WriteString(warn.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	return subsectionCardAction.Render(b.String())
}

func renderInsetSelectorCard(title, meta, summary string, lines []string) string {
	var b strings.Builder
	b.WriteString(renderRailTitleBar(title, meta))
	if strings.TrimSpace(summary) != "" {
		b.WriteString("\n")
		b.WriteString(dim.Render(summary))
	}
	if len(lines) > 0 {
		b.WriteString("\n")
		for _, line := range lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return subsectionCardAction.Render(strings.TrimRight(b.String(), "\n"))
}

func renderBadgeRow(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "  ")
}

func binaryBadge(labelText string, enabled bool) string {
	if enabled {
		return ok.Render(labelText + ":on")
	}
	return dim.Render(labelText + ":off")
}

func compactLabel(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || text == "" {
		return ""
	}
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	if limit <= 1 {
		return "…"
	}
	runes := []rune(text)
	return string(runes[:limit-1]) + "…"
}

func selectorStat(label string, value any) string {
	return fmt.Sprintf("%s:%v", label, value)
}

func renderPanelTitleBar(title, meta string) string {
	left := panelTitleBar.Render(title)
	if strings.TrimSpace(meta) == "" {
		return left
	}
	right := panelTitleMeta.Render(meta)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func renderRailTitleBar(title, meta string) string {
	left := railTitleBar.Render(title)
	if strings.TrimSpace(meta) == "" {
		return left
	}
	right := railTitleMeta.Render(meta)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func renderStatStrip(parts ...string) string {
	return statStrip.Render(renderBadgeRow(parts...))
}

func renderShellStatusBar(parts ...string) string {
	return shellFrame.Render(renderBadgeRow(
		shellLabel.Render("CHASSIS"),
		renderBadgeRow(parts...),
	))
}

func renderShellFrame(body string) string {
	return shellFrame.Render(body)
}

func renderCommandStrip(m model, text string) string {
	lines := []string{renderCommandBus(text)}
	if !m.commandLaneEnabled() {
		return commandStrip.Render(strings.Join(lines, "\n"))
	}
	if m.commandMode {
		lines = append(lines, renderCommandPrompt(m.commandInput))
	} else {
		lines = append(lines, commandPromptHint.Render("[:] command mode · type help"))
	}
	if strings.TrimSpace(m.commandStatus) != "" {
		lines = append(lines, renderCommandStatusLine(m.commandStatusLevel, m.commandStatus))
	}
	return commandStrip.Render(strings.Join(lines, "\n"))
}

func renderCommandStatusLine(level, message string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "ok", "success":
		return commandStatusOK.Render(message)
	case "warn", "warning":
		return commandStatusWarn.Render(message)
	case "error", "err":
		return commandStatusError.Render(message)
	default:
		return commandStatusInfo.Render(message)
	}
}

func renderCommandPrompt(input string) string {
	cursor := commandPromptCursor.Render("█")
	if strings.TrimSpace(input) == "" {
		return commandPromptLabel.Render("INPUT") + commandPromptSep.Render(" :: ") + commandPromptInput.Render(cursor)
	}
	return commandPromptLabel.Render("INPUT") + commandPromptSep.Render(" :: ") + commandPromptInput.Render(input) + commandPromptInput.Render(cursor)
}

func renderCommandBus(text string) string {
	parts := strings.Split(text, "·")
	items := make([]string, 0, len(parts)+1)
	items = append(items, commandBusLabel.Render("COMMAND BUS"))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, formatCommandBusItem(part))
	}
	return strings.Join(items, commandBusSep.Render("  │  "))
}

func formatCommandBusItem(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	keyParts := make([]string, 0, len(fields))
	descStart := len(fields)
	for i, field := range fields {
		if looksLikeCommandToken(field) {
			keyParts = append(keyParts, field)
			continue
		}
		descStart = i
		break
	}
	if len(keyParts) == 0 {
		return dim.Render(text)
	}
	keyText := commandBusKey.Render(strings.Join(keyParts, " "))
	if descStart >= len(fields) {
		return keyText
	}
	return keyText + commandBusSep.Render(" ") + dim.Render(strings.Join(fields[descStart:], " "))
}

func looksLikeCommandToken(field string) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
		return true
	}
	switch field {
	case "↑↓", "←→", "tab", "shift+tab", "enter", "esc", "backspace", "q":
		return true
	}
	return false
}
