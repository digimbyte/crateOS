package tui

import (
	"fmt"
	"strings"
)

func (m model) viewUsers() string {
	if m.userFormOpen {
		return renderSplitView(m, "Users", renderUserSelectionPanel(m), renderUserFocusPanel(m), "  tab / ↑↓ cycle fields · [enter] save user form · [esc] cancel form")
	}
	return renderSplitView(m, "Users", renderUserSelectionPanel(m), renderUserFocusPanel(m), "  ↑↓ select user · [a] add · [enter] edit · [r] cycle role · [p] perms preset · [s] set current · [d] delete · [esc] back · [:] command")
}

func renderUserSelectionPanel(m model) string {
	lines := []string{}
	if len(m.users) == 0 {
		lines = append(lines, dim.Render("  No users found."))
		return renderSelectionPanel("USER DIRECTORY", 34, lines)
	}
	for i, u := range m.users {
		cursor := m.currentUserIndex()
		marker := ""
		if u.Name == m.currentUser {
			marker = "  " + ok.Render("cur")
		}
		line := fmt.Sprintf("%s %s  %s%s", userRoleIndicator(u.Role), compactLabel(u.Name, 12), selectorStat("r", compactRole(u.Role)), marker)
		lines = append(lines, renderSelectorLineWithGlyph(i == cursor && !m.userFormOpen, userRailGlyph(u), line))
	}
	return renderSelectionPanelWithMeta("USER DIRECTORY", "OPERATORS", 34, lines)
}

func renderUserFocusPanel(m model) string {
	title := "ACTIVE USER"
	if m.userFormOpen {
		title = "USER ADMIN PANEL"
	}
	if m.userFormOpen {
		return renderActivePanel(title, 68, renderUserFormPanel(m))
	}
	if len(m.users) == 0 {
		var b strings.Builder
		b.WriteString(dim.Render("No user selected.\n"))
		b.WriteString(renderActionCard("NEXT ADD", dim.Render("role: "+m.newUserRole)))
		return renderActivePanel(title, 68, b.String())
	}
	u := m.users[m.currentUserIndex()]
	var header strings.Builder
	header.WriteString(renderPanelTitleBar(strings.ToUpper(u.Name), "OPERATOR"))
	header.WriteString("\n")
	header.WriteString(renderStatStrip(roleBadge(u.Role)))
	header.WriteString("\n")
	if u.Name == m.currentUser {
		header.WriteString(ok.Render("session:current\n"))
	} else {
		header.WriteString(dim.Render("session:standby\n"))
	}
	var permissions strings.Builder
	if len(u.Perms) == 0 {
		permissions.WriteString(dim.Render("No explicit permissions; inherits from role.\n"))
	} else {
		permissions.WriteString(renderBulletLines(u.Perms))
	}
	operations := renderPanelLines(
		dim.Render("[enter] edit selected user"),
		dim.Render("[r] cycle role"),
		dim.Render("[p] apply permission preset"),
		dim.Render("[s] switch current operator"),
		dim.Render("[d] delete selected user"),
	)
	admins, operators := userRoleCounts(m.users)
	var posture strings.Builder
	posture.WriteString(renderBadgeRow(
		selectorStat("usr", len(m.users)),
		selectorStat("adm", admins),
		selectorStat("ops", operators),
		selectorStat("cur", m.currentUser),
	))
	posture.WriteString("\n")
	posture.WriteString(dim.Render("Operator state shapes who can drive the rest of the control plane from this terminal surface."))
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(renderSummaryCard("CONTROL POSTURE", posture.String()))
	b.WriteString(renderSummaryCard("PERMISSIONS", permissions.String()))
	b.WriteString(renderActionCard("OPERATIONS", operations))
	b.WriteString(renderActionCard("NEXT ADD", dim.Render("default role: "+m.newUserRole)))
	return renderActivePanel(title, 68, b.String())
}

func renderUserFormPanel(m model) string {
	var b strings.Builder
	mode := "CREATE USER"
	if m.userFormEdit {
		mode = "EDIT USER"
	}
	b.WriteString(value.Render(mode))
	b.WriteString("\n")
	if m.userFormEdit {
		b.WriteString(dim.Render("Username is fixed during edit; explicit perms override the selected role."))
	} else {
		b.WriteString(dim.Render("Comma-separated perms; leave blank to inherit from role."))
	}
	b.WriteString("\n\n")
	b.WriteString(renderUserFormRow("Username", m.userFormName, m.userFormField == 0))
	b.WriteString("\n")
	b.WriteString(renderUserFormRow("Role", m.userFormRole, m.userFormField == 1))
	b.WriteString("\n")
	b.WriteString(renderUserFormRow("Perms", m.userFormPerms, m.userFormField == 2))
	b.WriteString(renderActionCard("FORM ACTIONS", renderPanelLines(
		dim.Render("[tab]/[↑↓] move field"),
		dim.Render("[enter] save"),
		dim.Render("[esc] cancel"),
	)))
	return b.String()
}

func (m model) currentUserIndex() int {
	if len(m.users) == 0 {
		return 0
	}
	cursor := m.cursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(m.users) {
		return len(m.users) - 1
	}
	return cursor
}

func userRoleIndicator(role string) string {
	switch role {
	case "admin":
		return danger.Render("◆")
	case "operator":
		return warn.Render("◈")
	case "staff":
		return ok.Render("●")
	case "viewer":
		return dim.Render("○")
	default:
		return warn.Render("?")
	}
}

func userRailGlyph(u userRow) string {
	switch u.Role {
	case "admin":
		return "◆"
	case "operator":
		return "◈"
	case "staff":
		return "●"
	case "viewer":
		return "○"
	default:
		return "?"
	}
}

func nextRole(role string) string {
	order := []string{"viewer", "operator", "staff", "admin"}
	for i, r := range order {
		if r == role {
			return order[(i+1)%len(order)]
		}
	}
	return "staff"
}

func togglePermPreset(current []string) []string {
	joined := strings.Join(current, ",")
	switch joined {
	case "", "logs.view,svc.list,net.status", "svc.list,logs.view,net.status":
		return []string{"users.view", "logs.view", "svc.list", "net.status"}
	case "users.view,logs.view,svc.list,net.status":
		return []string{"svc.*", "net.*", "proxy.*", "logs.view"}
	default:
		return []string{"logs.view", "svc.list", "net.status"}
	}
}
