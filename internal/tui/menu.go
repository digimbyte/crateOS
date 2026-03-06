package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
)

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
			m.currentView = viewStatus
			m.info = sysinfo.Gather()
			m.cursor = 0
		case "2":
			m.currentView = viewServices
			m.services = gatherServices()
			m.cursor = 0
		case "3":
			m.currentView = viewLogs
			m.cursor = 0
		case "4":
			m.currentView = viewNetwork
			m.interfaces = sysinfo.NetworkInterfaces()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		m.currentView = viewStatus
		m.info = sysinfo.Gather()
	case 1:
		m.currentView = viewServices
		m.services = gatherServices()
	case 2:
		m.currentView = viewLogs
	case 3:
		m.currentView = viewNetwork
		m.interfaces = sysinfo.NetworkInterfaces()
	case 4:
		m.quitting = true
		return m, tea.Quit
	}
	m.cursor = 0
	return m, nil
}

func (m model) viewMenu() string {
	var b strings.Builder

	// ── Header ──
	header := fmt.Sprintf(
		"C R A T E O S   %s\n%s · %s/%s",
		platform.Version, m.info.Hostname, m.info.OS, m.info.Arch,
	)
	b.WriteString(headerBox.Render(header))
	b.WriteString("\n\n")

	// ── Menu items ──
	for i, item := range menuItems {
		hotkey := fmt.Sprintf("[%d]", i+1)
		if i == len(menuItems)-1 {
			hotkey = "[Q]"
		}

		line := fmt.Sprintf("%s %s", hotkey, item)
		if i == m.cursor {
			b.WriteString(menuActive.Render("▸ " + line))
		} else {
			b.WriteString(menuItem.Render("  " + line))
		}
		b.WriteString("\n")
	}

	// ── Footer ──
	b.WriteString(footer.Render("  ↑↓ navigate · enter select · q quit"))

	return b.String()
}
