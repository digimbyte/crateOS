package tui

import "github.com/crateos/crateos/internal/sysinfo"

func userRoleCounts(users []userRow) (admins, operators int) {
	for _, u := range users {
		switch u.Role {
		case "admin":
			admins++
		case "operator":
			operators++
		}
	}
	return admins, operators
}

func compactRole(role string) string {
	switch role {
	case "admin":
		return "adm"
	case "operator":
		return "op"
	case "staff":
		return "stf"
	case "viewer":
		return "view"
	default:
		return role
	}
}

func roleBadge(role string) string {
	text := "role:" + role
	switch role {
	case "admin":
		return danger.Render(text)
	case "operator":
		return warn.Render(text)
	case "staff":
		return ok.Render(text)
	case "viewer":
		return dim.Render(text)
	default:
		return warn.Render(text)
	}
}

func (m *model) enterMenuView() {
	m.refreshOverview()
	m.currentView = viewMenu
	m.cursor = 0
	m.logSourceCursor = 0
}

func (m *model) enterStatusView() {
	m.refreshOverview()
	m.currentView = viewStatus
	m.cursor = 0
	m.logSourceCursor = 0
	m.statusSection = 0
	m.ownershipCursor = 0
}

func (m *model) enterDiagnosticsView() {
	m.refreshOverview()
	m.currentView = viewDiagnostics
	m.cursor = 0
	m.logSourceCursor = 0
	m.statusSection = 0
	m.ownershipCursor = 0
}

func (m *model) enterServicesView() {
	m.refreshServices()
	m.currentView = viewServices
	m.cursor = 0
	m.logSourceCursor = 0
}

func (m *model) enterUsersView() {
	m.refreshUsers()
	m.currentView = viewUsers
	m.cursor = 0
	m.logSourceCursor = 0
}

func (m *model) enterLogsView() {
	m.refreshOverview()
	m.currentView = viewLogs
	m.cursor = 0
	m.logSourceCursor = 0
}

func (m *model) enterNetworkView() {
	m.refreshOverview()
	m.interfaces = sysinfo.NetworkInterfaces()
	m.currentView = viewNetwork
	m.cursor = 0
	m.logSourceCursor = 0
}
