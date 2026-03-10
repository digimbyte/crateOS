package tui

func (m *model) refreshUsers() {
	if rows := fetchUsersViaAPI(m.currentUser); rows != nil {
		m.users = rows
		m.controlPlaneOnline = true
		return
	}
	m.users = fetchUsersFromConfig()
}
