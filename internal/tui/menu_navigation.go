package tui

import tea "github.com/charmbracelet/bubbletea"

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter":
			return m.selectMenuItem()
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.enterStatusView()
		case "2":
			m.enterServicesView()
		case "3":
			m.enterDiagnosticsView()
		case "4":
			m.enterUsersView()
		case "5":
			m.enterLogsView()
		case "6":
			m.enterNetworkView()
		}
	}
	return m, nil
}

func (m model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		m.enterStatusView()
	case 1:
		m.enterServicesView()
	case 2:
		m.enterDiagnosticsView()
	case 3:
		m.enterUsersView()
	case 4:
		m.enterLogsView()
	case 5:
		m.enterNetworkView()
	case 6:
		m.quitting = true
		return m, tea.Quit
	}
	m.cursor = 0
	return m, nil
}
