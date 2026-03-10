package tui

import "strings"

func (m model) executeUserBatchMutation(rawTarget string, actionLabel string, emptyMessage string, errorPrefix string, partialPrefix string, successPrefix string, mutate func(userRow) error) (tea.Model, tea.Cmd) {
	if !m.requireLiveControlPlane(actionLabel) {
		return m, nil
	}
	target := strings.TrimSpace(rawTarget)
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
		m.setCommandWarn(emptyMessage)
		return m, nil
	}
	applied := []string{}
	failed := []string{}
	for _, u := range targetUsers {
		if err := mutate(u); err != nil {
			failed = append(failed, u.Name)
			continue
		}
		applied = append(applied, u.Name)
	}
	m.refreshUsers()
	if len(applied) == 0 {
		m.setCommandError(errorPrefix + strings.Join(failed, ", "))
		return m, nil
	}
	if len(failed) > 0 {
		m.setCommandWarn(partialPrefix + strings.Join(applied, ",") + " failed=" + strings.Join(failed, ","))
		return m, nil
	}
	m.setCommandOK(successPrefix + strings.Join(applied, ","))
	return m, nil
}
