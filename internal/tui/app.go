package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/sysinfo"
)

func (m model) controlPlaneMode() string {
	if m.controlPlaneOnline {
		return "agent-live"
	}
	return "fallback-local"
}

func normalizeServiceAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "install":
		return "enable"
	case "uninstall":
		return "disable"
	case "restart":
		return "restart"
	case "enable", "start", "stop", "disable":
		return strings.ToLower(strings.TrimSpace(action))
	default:
		return ""
	}
}

func (m *model) bootstrapAdmin(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	// Load config and add bootstrap admin
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	// Check if user already exists
	for _, u := range cfg.Users.Users {
		if u.Name == name {
			return false
		}
	}
	// Add admin user
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name:        name,
		Role:        "admin",
		Permissions: []string{},
	})
	if err := config.SaveUsers(cfg); err != nil {
		return false
	}
	m.currentUser = name
	m.currentView = viewMenu
	m.cursor = 0
	m.refreshUsers()
	m.refreshServices()
	m.info = sysinfo.Gather()
	return true
}

// NewModel creates the initial TUI model with fresh system info (API first, fallback local).
func NewModel() model {
	startUser := selectInitialUser()
	if info, svcs, platformInfo, diagnostics, usr := fetchStatusViaAPI(startUser); info != nil {
		users := fetchUsersViaAPI(startUser)
		startView := viewMenu
		if len(users) == 0 {
			startView = viewSetup
		}
		return model{
			currentView:        startView,
			info:               *info,
			interfaces:         sysinfo.NetworkInterfaces(),
			services:           svcs,
			platform:           platformInfo,
			diagnostics:        diagnostics,
			users:              users,
			currentUser:        usr,
			newUserRole:        "staff",
			setupAdmin:         "admin",
			controlPlaneOnline: true,
		}
	}
	users := fetchUsersFromConfig()
	startView := viewMenu
	if len(users) == 0 {
		startView = viewSetup
	}
	return model{
		currentView:        startView,
		info:               sysinfo.Gather(),
		interfaces:         sysinfo.NetworkInterfaces(),
		services:           gatherServices(),
		platform:           readFallbackPlatformState(),
		diagnostics:        readFallbackDiagnostics(),
		users:              users,
		currentUser:        startUser,
		newUserRole:        "staff",
		setupAdmin:         "admin",
		controlPlaneOnline: false,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if m.commandLaneEnabled() {
			if m.commandMode {
				return m.updateCommandInput(msg)
			}
			if msg.String() == ":" {
				m.commandMode = true
				m.commandInput = ""
				m.setCommandInfo("command mode active")
				return m, nil
			}
		}
	}

	switch m.currentView {
	case viewMenu:
		return m.updateMenu(msg)
	case viewSetup:
		return m.updateSetup(msg)
	case viewStatus:
		return m.updateStatus(msg)
	case viewDiagnostics:
		return m.updateDiagnostics(msg)
	case viewServices:
		return m.updateServices(msg)
	case viewUsers:
		return m.updateUsers(msg)
	case viewLogs:
		return m.updateLogs(msg)
	case viewNetwork:
		return m.updateNetwork(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.currentView {
	case viewMenu:
		return m.viewMenu()
	case viewSetup:
		return m.viewSetup()
	case viewStatus:
		return m.viewStatus()
	case viewDiagnostics:
		return m.viewDiagnostics()
	case viewServices:
		return m.viewServices()
	case viewUsers:
		return m.viewUsers()
	case viewLogs:
		return m.viewLogs()
	case viewNetwork:
		return m.viewNetwork()
	}
	return ""
}

// Run starts the TUI in alt-screen mode.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *model) clampOwnershipCursor() {
	if len(m.diagnostics.Ownership.Workloads) == 0 {
		m.ownershipCursor = 0
		return
	}
	if m.ownershipCursor < 0 {
		m.ownershipCursor = 0
		return
	}
	if m.ownershipCursor >= len(m.diagnostics.Ownership.Workloads) {
		m.ownershipCursor = len(m.diagnostics.Ownership.Workloads) - 1
	}
}

func (m *model) selectOwnershipWorkload(target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	if target == "" {
		return false
	}
	for i, workload := range m.diagnostics.Ownership.Workloads {
		if strings.EqualFold(strings.TrimSpace(workload.Crate), target) ||
			strings.EqualFold(strings.TrimSpace(workload.ActorName), target) ||
			strings.EqualFold(strings.TrimSpace(workload.ActorUser), target) ||
			strings.EqualFold(strings.TrimSpace(workload.ActorID), target) {
			m.ownershipCursor = i
			return true
		}
	}
	return false
}
