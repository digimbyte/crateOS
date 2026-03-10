package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
		return m.executeUserBatchMutation(strings.Join(params, " "), "user role cycle", "no users resolved for role action", "role cycle failed for: ", "role partial: ok=", "role cycled for users: ", func(u userRow) error {
			return updateUserDirect(u.Name, "", nextRole(u.Role), nil)
		})
	case "perms":
		return m.executeUserBatchMutation(strings.Join(params, " "), "user permissions update", "no users resolved for perms action", "perms toggle failed for: ", "perms partial: ok=", "perms preset toggled for users: ", func(u userRow) error {
			return updateUserDirect(u.Name, "", "", togglePermPreset(u.Perms))
		})
	case "delete":
		return m.executeUserBatchMutation(strings.Join(params, " "), "user delete", "no users resolved for delete action", "delete failed for: ", "delete partial: ok=", "users deleted: ", func(u userRow) error {
			return deleteUserDirect(u.Name)
		})
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
