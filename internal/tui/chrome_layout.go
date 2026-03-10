package tui

import (
	"strings"

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

func renderShellStatusBar(parts ...string) string {
	return shellFrame.Render(renderBadgeRow(
		shellLabel.Render("CHASSIS"),
		renderBadgeRow(parts...),
	))
}

func renderShellFrame(body string) string {
	return shellFrame.Render(body)
}
