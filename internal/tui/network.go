package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"

	"github.com/crateos/crateos/internal/sysinfo"
)

func (m model) updateNetwork(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.interfaces)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) executeNetCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewNetwork {
		m.enterNetworkView()
	}
	if len(m.interfaces) == 0 {
		m.setCommandWarn("no interfaces available")
		return m, nil
	}
	switch mod {
	case "list":
		names := make([]string, 0, len(m.interfaces))
		for _, iface := range m.interfaces {
			if strings.TrimSpace(iface.Name) != "" {
				names = append(names, iface.Name)
			}
		}
		if len(names) == 0 {
			m.setCommandWarn("interfaces: none")
			return m, nil
		}
		m.setCommandInfo("interfaces: " + strings.Join(names, ", "))
		return m, nil
	case "":
		m.setCommandOK("route: network")
	case "next":
		if m.cursor < len(m.interfaces)-1 {
			m.cursor++
		}
		m.setCommandInfo("network selector advanced")
	case "prev":
		if m.cursor > 0 {
			m.cursor--
		}
		m.setCommandInfo("network selector reversed")
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: net select <interface|iface1,iface2>")
			return m, nil
		}
		targets := parseCSVTargets(strings.Join(params, " "))
		if len(targets) == 0 {
			m.setCommandWarn("usage: net select <interface|iface1,iface2>")
			return m, nil
		}
		selected := []string{}
		missing := []string{}
		for _, target := range targets {
			found := false
			for i, iface := range m.interfaces {
				if strings.EqualFold(iface.Name, target) {
					m.cursor = i
					selected = append(selected, iface.Name)
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, target)
			}
		}
		if len(selected) == 0 {
			m.setCommandError("interface not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(missing) > 0 {
			m.setCommandWarn("net select partial: ok=" + strings.Join(selected, ",") + " missing=" + strings.Join(missing, ","))
			return m, nil
		}
		m.setCommandOK("interfaces selected: " + strings.Join(selected, ","))
	default:
		m.setCommandWarn("usage: net <list|next|prev|select> [interface|iface1,iface2]")
	}
	return m, nil
}

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
