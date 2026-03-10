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

func (m model) executeUserCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewUsers {
		m.enterUsersView()
	}
	switch mod {
	case "list":
		if len(m.users) == 0 {
			m.setCommandWarn("users: none")
			return m, nil
		}
		names := make([]string, 0, len(m.users))
		for _, u := range m.users {
			if strings.TrimSpace(u.Name) != "" {
				names = append(names, u.Name)
			}
		}
		m.setCommandInfo("users: " + strings.Join(names, ", "))
		return m, nil
	case "add":
		if !m.requireLiveControlPlane("user add") {
			return m, nil
		}
		if len(params) < 2 {
			m.setCommandWarn("usage: user add <name> <role> [perm1,perm2]")
			return m, nil
		}
		name := strings.TrimSpace(params[0])
		role := strings.TrimSpace(params[1])
		if name == "" || role == "" {
			m.setCommandWarn("usage: user add <name> <role> [perm1,perm2]")
			return m, nil
		}
		perms := []string{}
		if len(params) > 2 {
			perms = parsePermList(strings.Join(params[2:], " "))
		}
		if err := addUserDirect(name, role, perms); err != nil {
			m.setCommandError("user add failed: " + err.Error())
			return m, nil
		}
		m.refreshUsers()
		m.setCommandOK("user added: " + name)
		return m, nil
	case "rename":
		if !m.requireLiveControlPlane("user rename") {
			return m, nil
		}
		if len(params) < 2 {
			m.setCommandWarn("usage: user rename <old> <new>")
			return m, nil
		}
		target := strings.TrimSpace(params[0])
		next := strings.TrimSpace(params[1])
		if target == "" || next == "" {
			m.setCommandWarn("usage: user rename <old> <new>")
			return m, nil
		}
		if err := updateUserDirect(target, next, "", nil); err != nil {
			m.setCommandError("user rename failed: " + err.Error())
			return m, nil
		}
		m.refreshUsers()
		m.setCommandOK("user renamed: " + target + " -> " + next)
		return m, nil
	case "set", "use":
		if len(params) == 0 {
			m.setCommandWarn("usage: user set <name>")
			return m, nil
		}
		return m.executeSetCurrentUserCommand(strings.Join(params, " "))
	case "role":
		if !m.requireLiveControlPlane("user role cycle") {
			return m, nil
		}
		target := strings.TrimSpace(strings.Join(params, " "))
		defaultUser := userRow{}
		if idx := m.currentUserIndex(); idx >= 0 && idx < len(m.users) {
			defaultUser = m.users[idx]
		}
		targetUsers, missing := resolveUserTargets(m.users, defaultUser, target)
		if len(missing) > 0 {
			m.setCommandError("user not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(targetUsers) == 0 {
			m.setCommandWarn("no users resolved for role action")
			return m, nil
		}
		applied := []string{}
		failed := []string{}
		for _, u := range targetUsers {
			if err := updateUserDirect(u.Name, "", nextRole(u.Role), nil); err != nil {
				failed = append(failed, u.Name)
				continue
			}
			applied = append(applied, u.Name)
		}
		m.refreshUsers()
		if len(applied) == 0 {
			m.setCommandError("role cycle failed for: " + strings.Join(failed, ", "))
			return m, nil
		}
		if len(failed) > 0 {
			m.setCommandWarn("role partial: ok=" + strings.Join(applied, ",") + " failed=" + strings.Join(failed, ","))
			return m, nil
		}
		m.setCommandOK("role cycled for users: " + strings.Join(applied, ","))
		return m, nil
	case "perms":
		if !m.requireLiveControlPlane("user permissions update") {
			return m, nil
		}
		target := strings.TrimSpace(strings.Join(params, " "))
		defaultUser := userRow{}
		if idx := m.currentUserIndex(); idx >= 0 && idx < len(m.users) {
			defaultUser = m.users[idx]
		}
		targetUsers, missing := resolveUserTargets(m.users, defaultUser, target)
		if len(missing) > 0 {
			m.setCommandError("user not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(targetUsers) == 0 {
			m.setCommandWarn("no users resolved for perms action")
			return m, nil
		}
		applied := []string{}
		failed := []string{}
		for _, u := range targetUsers {
			if err := updateUserDirect(u.Name, "", "", togglePermPreset(u.Perms)); err != nil {
				failed = append(failed, u.Name)
				continue
			}
			applied = append(applied, u.Name)
		}
		m.refreshUsers()
		if len(applied) == 0 {
			m.setCommandError("perms toggle failed for: " + strings.Join(failed, ", "))
			return m, nil
		}
		if len(failed) > 0 {
			m.setCommandWarn("perms partial: ok=" + strings.Join(applied, ",") + " failed=" + strings.Join(failed, ","))
			return m, nil
		}
		m.setCommandOK("perms preset toggled for users: " + strings.Join(applied, ","))
		return m, nil
	case "delete":
		if !m.requireLiveControlPlane("user delete") {
			return m, nil
		}
		target := strings.TrimSpace(strings.Join(params, " "))
		targetUsers, missing := resolveUserTargets(m.users, userRow{}, target)
		if len(missing) > 0 {
			m.setCommandError("user not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(targetUsers) == 0 {
			m.setCommandWarn("no users resolved for delete action")
			return m, nil
		}
		applied := []string{}
		failed := []string{}
		for _, u := range targetUsers {
			if err := deleteUserDirect(u.Name); err != nil {
				failed = append(failed, u.Name)
				continue
			}
			applied = append(applied, u.Name)
		}
		m.refreshUsers()
		if len(applied) == 0 {
			m.setCommandError("delete failed for: " + strings.Join(failed, ", "))
			return m, nil
		}
		if len(failed) > 0 {
			m.setCommandWarn("delete partial: ok=" + strings.Join(applied, ",") + " failed=" + strings.Join(failed, ","))
			return m, nil
		}
		m.setCommandOK("users deleted: " + strings.Join(applied, ","))
		return m, nil
	case "next":
		if len(m.users) == 0 {
			m.setCommandWarn("no users available")
			return m, nil
		}
		if m.cursor < len(m.users)-1 {
			m.cursor++
		}
		m.setCommandInfo("user selector advanced")
		return m, nil
	case "prev":
		if len(m.users) == 0 {
			m.setCommandWarn("no users available")
			return m, nil
		}
		if m.cursor > 0 {
			m.cursor--
		}
		m.setCommandInfo("user selector reversed")
		return m, nil
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: user select <name|name1,name2>")
			return m, nil
		}
		target := strings.TrimSpace(strings.Join(params, " "))
		targetUsers, missing := resolveUserTargets(m.users, userRow{}, target)
		if len(missing) > 0 {
			m.setCommandError("user not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(targetUsers) == 0 {
			m.setCommandWarn("usage: user select <name|name1,name2>")
			return m, nil
		}
		last := targetUsers[len(targetUsers)-1]
		for i, u := range m.users {
			if strings.EqualFold(u.Name, last.Name) {
				m.cursor = i
				break
			}
		}
		m.setCommandOK("users selected: " + strings.Join(userNames(targetUsers), ","))
		return m, nil
	case "":
		m.setCommandWarn("usage: user <list|add|rename|set|role|perms|delete|next|prev|select> [name|name1,name2]")
		return m, nil
	default:
		m.setCommandWarn("usage: user <list|add|rename|set|role|perms|delete|next|prev|select> [name|name1,name2]")
		return m, nil
	}
}

func (m model) executeSetCurrentUserCommand(rawName string) (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		m.setCommandWarn("usage: user set <name>")
		return m, nil
	}
	m.refreshUsers()
	for _, u := range m.users {
		if strings.EqualFold(u.Name, name) {
			m.currentUser = u.Name
			m.setCommandOK("current user: " + u.Name)
			return m, nil
		}
	}
	m.setCommandError("user not found: " + name)
	return m, nil
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
