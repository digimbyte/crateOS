package tui

import "strings"

func renderStatusFocusPanel(m model) string {
	switch statusPanelTitle(m.statusSection) {
	case "System":
		return renderSystemStatusPanel(m)
	case "Services":
		return renderServiceStatusPanel(m)
	case "Platform":
		return renderPlatformStatusPanel(m)
	default:
		return renderSystemStatusPanel(m)
	}
}

func statusSelectorLabel(m model, item string) string {
	switch item {
	case "System":
		return renderBadgeRow("SYSTEM", selectorStat("host", compactLabel(m.info.Hostname, 8)))
	case "Services":
		failed, partial, staged, _ := menuServiceCounts(m.services)
		return renderBadgeRow(
			"SERVICES",
			selectorStat("f", failed),
			selectorStat("p", partial),
			selectorStat("s", staged),
		)
	case "Platform":
		ready, failed := menuPlatformCounts(m.platform.Adapters)
		return renderBadgeRow(
			"PLATFORM",
			selectorStat("r", ready),
			selectorStat("f", failed),
		)
	default:
		return strings.ToUpper(item)
	}
}

func statusRailGlyph(item string) string {
	switch item {
	case "System":
		return "◆"
	case "Services":
		return "■"
	case "Platform":
		return "◉"
	default:
		return "•"
	}
}
