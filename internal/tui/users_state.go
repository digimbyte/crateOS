package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) openAddUserForm() {
	m.userFormOpen = true
	m.userFormEdit = false
	m.userFormField = 0
	m.userFormTarget = ""
	m.userFormName = fmt.Sprintf("user-%d", time.Now().Unix())
	m.userFormRole = m.newUserRole
	m.userFormPerms = ""
}

func (m *model) openEditUserForm(u userRow) {
	m.userFormOpen = true
	m.userFormEdit = true
	m.userFormField = 1
	m.userFormTarget = u.Name
	m.userFormName = u.Name
	m.userFormRole = u.Role
	m.userFormPerms = strings.Join(u.Perms, ",")
}

func (m *model) closeUserForm() {
	m.userFormOpen = false
	m.userFormEdit = false
	m.userFormField = 0
	m.userFormTarget = ""
	m.userFormName = ""
	m.userFormRole = ""
	m.userFormPerms = ""
}

func (m *model) appendUserFormField(s string) {
	switch m.userFormField {
	case 0:
		if m.userFormEdit {
			return
		}
		m.userFormName += s
	case 1:
		m.userFormRole += s
	case 2:
		m.userFormPerms += s
	}
}

func (m *model) backspaceUserFormField() {
	switch m.userFormField {
	case 0:
		if m.userFormEdit {
			return
		}
		if len(m.userFormName) > 0 {
			m.userFormName = m.userFormName[:len(m.userFormName)-1]
		}
	case 1:
		if len(m.userFormRole) > 0 {
			m.userFormRole = m.userFormRole[:len(m.userFormRole)-1]
		}
	case 2:
		if len(m.userFormPerms) > 0 {
			m.userFormPerms = m.userFormPerms[:len(m.userFormPerms)-1]
		}
	}
}

func parsePermList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func renderUserFormRow(labelText, current string, active bool) string {
	row := label.Render(labelText+": ") + value.Render(current)
	if active {
		return selectorActive.Render("▌ " + row)
	}
	return selectorIdle.Render("│ " + row)
}

func (m model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.userFormOpen {
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
