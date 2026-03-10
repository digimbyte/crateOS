package tui

import (
	"fmt"
	"strings"
	"time"
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
