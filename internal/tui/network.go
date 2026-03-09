package tui

import (
	"fmt"
	"strings"

	"github.com/crateos/crateos/internal/sysinfo"
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

func renderNetworkFocusPanel(m model) string {
	if len(m.interfaces) == 0 {
		return renderActivePanel("ACTIVE INTERFACE", 68, dim.Render("No interface selected."))
	}
	iface := m.interfaces[m.currentInterfaceIndex()]
	var header strings.Builder
	header.WriteString(renderPanelTitleBar(strings.ToUpper(iface.Name), "IFACE"))
	header.WriteString("\n")
	header.WriteString(renderStatStrip(
		linkBadge(iface.Up),
		binaryBadge("addr", len(iface.Addrs) > 0),
	))
	header.WriteString("\n")
	var identity strings.Builder
	if strings.TrimSpace(iface.MAC) != "" {
		identity.WriteString(dim.Render("mac: " + iface.MAC))
	} else {
		identity.WriteString(dim.Render("mac: unavailable"))
	}
	var addresses strings.Builder
	if len(iface.Addrs) == 0 {
		addresses.WriteString(dim.Render("No addresses assigned."))
		addresses.WriteString("\n")
	} else {
		for _, addr := range iface.Addrs {
			addresses.WriteString(value.Render(addr))
			addresses.WriteString("\n")
		}
	}
	upCount, addressedCount := networkPostureCounts(m.interfaces)
	var postureSummary strings.Builder
	postureSummary.WriteString(renderBadgeRow(
		selectorStat("if", len(m.interfaces)),
		selectorStat("up", upCount),
		selectorStat("addr", addressedCount),
	))
	postureSummary.WriteString("\n")
	postureSummary.WriteString(dim.Render("Interface state here is native posture; use Platform for rendered adapter outputs and target mapping."))
	posture := dim.Render("Use System Status → Platform to inspect rendered network adapter state and native targets.")
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(renderSummaryCard("SYSTEM NETWORK", postureSummary.String()))
	b.WriteString(renderSummaryCard("IDENTITY", identity.String()))
	b.WriteString(renderSubsectionCard("ADDRESSES", addresses.String()))
	b.WriteString(renderActionCard("NETWORK POSTURE", posture))
	return renderActivePanel("ACTIVE INTERFACE", 68, b.String())
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

func linkBadge(up bool) string {
	if up {
		return ok.Render("link:up")
	}
	return dim.Render("link:down")
}
func networkRailGlyph(iface sysinfo.NetIface) string {
	switch {
	case iface.Up && len(iface.Addrs) > 0:
		return "◉"
	case iface.Up:
		return "◌"
	default:
		return "○"
	}
}

func networkPostureCounts(interfaces []sysinfo.NetIface) (upCount, addressedCount int) {
	for _, iface := range interfaces {
		if iface.Up {
			upCount++
		}
		if len(iface.Addrs) > 0 {
			addressedCount++
		}
	}
	return upCount, addressedCount
}
