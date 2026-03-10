package tui

import "strings"

func diagnosticsSelectorLabel(m model, item string) string {
	switch item {
	case "Summary":
		return renderBadgeRow("DIAG", selectorStat("cfg", m.diagnostics.Config.Tracked), selectorStat("plane", m.controlPlaneMode()))
	case "Verification":
		return renderBadgeRow("VERIFY", selectorStat("st", strings.TrimSpace(m.diagnostics.Verification.Status)), selectorStat("miss", len(m.diagnostics.Verification.Missing)))
	case "Ownership":
		return renderBadgeRow("ACTORS", selectorStat("wrk", m.diagnostics.Ownership.Managed), selectorStat("blk", m.diagnostics.Ownership.Blocked), selectorStat("act", m.diagnostics.Ownership.Active))
	case "Config":
		return renderBadgeRow("CONFIG", selectorStat("u", m.diagnostics.Config.Unmonitored), selectorStat("x", m.diagnostics.Config.ExternalEdits))
	default:
		return strings.ToUpper(item)
	}
}

func diagnosticsRailGlyph(item string) string {
	switch item {
	case "Summary":
		return "◆"
	case "Verification":
		return "◈"
	case "Ownership":
		return "◎"
	case "Config":
		return "▣"
	default:
		return "•"
	}
}
