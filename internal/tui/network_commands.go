package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
