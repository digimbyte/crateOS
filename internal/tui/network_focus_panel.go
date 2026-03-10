package tui

import "strings"

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
