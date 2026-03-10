package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.userFormOpen {
			return m.updateUsersFormInput(msg)
		}
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "a":
			m.openAddUserForm()
		case "enter":
			if m.cursor < len(m.users) {
				m.openEditUserForm(m.users[m.cursor])
			}
		case "r":
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user role cycle") {
					return m, nil
				}
				u := m.users[m.cursor]
				_ = updateUserDirect(u.Name, "", nextRole(u.Role), nil)
				m.refreshUsers()
			} else {
				m.newUserRole = nextRole(m.newUserRole)
			}
		case "p":
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user permissions update") {
					return m, nil
				}
				u := m.users[m.cursor]
				_ = updateUserDirect(u.Name, "", "", togglePermPreset(u.Perms))
				m.refreshUsers()
			}
		case "s":
			if m.cursor < len(m.users) {
				m.currentUser = m.users[m.cursor].Name
			}
		case "d":
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user delete") {
					return m, nil
				}
				name := m.users[m.cursor].Name
				if name != "" {
					_ = deleteUserDirect(name)
					m.refreshUsers()
				}
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.users)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) updateUsersFormInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeUserForm()
	case "tab", "down":
		m.userFormField = (m.userFormField + 1) % 3
	case "shift+tab", "up":
		m.userFormField = (m.userFormField + 2) % 3
	case "backspace":
		m.backspaceUserFormField()
	case "enter":
		name := strings.TrimSpace(m.userFormName)
		role := strings.TrimSpace(m.userFormRole)
		perms := parsePermList(m.userFormPerms)
		if name == "" || role == "" {
			return m, nil
		}
		if !m.requireLiveControlPlane("user update") {
			return m, nil
		}
		if m.userFormEdit {
			_ = updateUserDirect(m.userFormTarget, name, role, perms)
		} else {
			_ = addUserDirect(name, role, perms)
		}
		m.refreshUsers()
		m.closeUserForm()
	default:
		if len(msg.String()) == 1 {
			m.appendUserFormField(msg.String())
		}
	}
	return m, nil
}
