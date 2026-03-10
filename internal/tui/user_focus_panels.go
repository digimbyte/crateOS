package tui

import "strings"

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
