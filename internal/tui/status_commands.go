package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.statusSection > 0 {
				m.statusSection--
			}
		case "down", "j":
			if m.statusSection < len(statusPanelItems())-1 {
				m.statusSection++
			}
		case "1":
			m.statusSection = 0
		case "2":
			m.statusSection = 1
		case "3":
			m.statusSection = 2
		}
	}
	return m, nil
}

func (m model) executeStatusCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewStatus {
		m.enterStatusView()
	}
	switch mod {
	case "list":
		m.setCommandInfo("status sections: system, services, platform")
		return m, nil
	case "":
		m.setCommandOK("route: status")
	case "system", "sys", "1":
		m.statusSection = 0
		m.setCommandOK("status section: system")
	case "services", "svc", "2":
		m.statusSection = 1
		m.setCommandOK("status section: services")
	case "platform", "plat", "3":
		m.statusSection = 2
		m.setCommandOK("status section: platform")
	case "next":
		if m.statusSection < len(statusPanelItems())-1 {
			m.statusSection++
		}
		m.setCommandInfo("status section advanced")
	case "prev":
		if m.statusSection > 0 {
			m.statusSection--
		}
		m.setCommandInfo("status section reversed")
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: status select <system|services|platform|1|2|3>")
			return m, nil
		}
		return m.executeStatusCommand(strings.ToLower(params[0]), params[1:])
	default:
		m.setCommandWarn("usage: status <system|services|platform|next|prev>")
	}
	return m, nil
}
