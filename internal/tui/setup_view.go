package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "backspace":
			if len(m.setupAdmin) > 0 {
				m.setupAdmin = m.setupAdmin[:len(m.setupAdmin)-1]
			}
		case "enter":
			name := strings.TrimSpace(m.setupAdmin)
			if name == "" {
				return m, nil
			}
			if !m.requireLiveControlPlane("bootstrap") {
				return m, nil
			}
			if m.bootstrapAdmin(name) {
				m.setCommandOK("bootstrap complete: " + name)
			}
		default:
			if len(msg.String()) == 1 {
				m.setupAdmin += msg.String()
			}
		}
	}
	return m, nil
}

func (m model) viewSetup() string {
	var b strings.Builder
	b.WriteString(headerBox.Render("First Boot Setup"))
	b.WriteString("\n\n")
	b.WriteString("Create the initial root admin for this CrateOS install.\n\n")
	b.WriteString(label.Render("Admin username: "))
	b.WriteString(value.Render(m.setupAdmin))
	b.WriteString("\n\n")
	b.WriteString(dim.Render("Press enter to bootstrap the platform."))
	b.WriteString("\n")
	b.WriteString(footer.Render("  type username · enter create admin"))
	return b.String()
}
