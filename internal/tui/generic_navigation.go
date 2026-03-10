package tui

import tea "github.com/charmbracelet/bubbletea"

// updateGeneric handles ESC/backspace to return to the menu from any sub-view.
func (m model) updateGeneric(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		}
	}
	return m, nil
}
