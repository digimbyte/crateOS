package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/sysinfo"
)

// ── View state ──────────────────────────────────────────────────────

type viewID int

const (
	viewMenu viewID = iota
	viewSetup
	viewStatus
	viewDiagnostics
	viewServices
	viewUsers
	viewLogs
	viewNetwork
)

// ── Service info ────────────────────────────────────────────────────

// ServiceInfo describes a managed service and its runtime status.
type ServiceInfo struct {
	Name                       string        `json:"name"`
	DisplayName                string        `json:"display_name"`
	Status                     string        `json:"status"`  // "active", "inactive", "failed", "unknown"
	Type                       string        `json:"runtime"` // "systemd" or "docker"
	ActorName                  string        `json:"actor_name"`
	ActorType                  string        `json:"actor_type"`
	ActorID                    string        `json:"actor_id"`
	ActorUser                  string        `json:"actor_user"`
	ActorGroup                 string        `json:"actor_group"`
	ActorHome                  string        `json:"actor_home"`
	ActorRuntimeDir            string        `json:"actor_runtime_dir"`
	ActorStateDir              string        `json:"actor_state_dir"`
	ActorProvisioning          string        `json:"actor_provisioning"`
	ActorProvisioningError     string        `json:"actor_provisioning_error"`
	ActorProvisioningUpdatedAt string        `json:"actor_provisioning_updated_at"`
	ActorProvisioningStatePath string        `json:"actor_provisioning_state_path"`
	ActorOwnershipStatus       string        `json:"actor_ownership_status"`
	ActorOwnershipUpdatedAt    string        `json:"actor_ownership_updated_at"`
	ActorOwnershipRetiredAt    string        `json:"actor_ownership_retired_at"`
	DeploySource               string        `json:"deploy_source"`
	UploadPath                 string        `json:"upload_path"`
	WorkingDir                 string        `json:"working_dir"`
	Entry                      string        `json:"entry"`
	InstallCommand             string        `json:"install_command"`
	EnvironmentFile            string        `json:"environment_file"`
	ExecutionMode              string        `json:"execution_mode"`
	ExecutionAdapter           string        `json:"execution_adapter"`
	ExecutionStatus            string        `json:"execution_status"`
	PrimaryUnit                string        `json:"primary_unit"`
	CompanionUnit              string        `json:"companion_unit"`
	PrimaryUnitPath            string        `json:"primary_unit_path"`
	CompanionUnitPath          string        `json:"companion_unit_path"`
	ExecutionStatePath         string        `json:"execution_state_path"`
	StartCommand               string        `json:"start_command"`
	Schedule                   string        `json:"schedule"`
	Timeout                    string        `json:"timeout"`
	StopTimeout                string        `json:"stop_timeout"`
	OnTimeout                  string        `json:"on_timeout"`
	KillSignal                 string        `json:"kill_signal"`
	ConcurrencyPolicy          string        `json:"concurrency_policy"`
	ExecutionSummary           string        `json:"execution_summary"`
	Stateful                   bool          `json:"stateful"`
	DataPath                   string        `json:"data_path"`
	NativeDataPath             string        `json:"native_data_path"`
	StorageSummary             string        `json:"storage_summary"`
	Health                     string        `json:"health"`
	Desired                    bool          `json:"desired"`
	Autostart                  bool          `json:"autostart"`
	Active                     bool          `json:"active"`
	Enabled                    bool          `json:"enabled"`
	Module                     bool          `json:"module"`
	Ready                      bool          `json:"ready"`
	PackagesInstalled          bool          `json:"packages_installed"`
	MissingPackages            []string      `json:"missing_packages"`
	Summary                    string        `json:"summary"`
	LastError                  string        `json:"last_error"`
	LastAction                 string        `json:"last_action"`
	LastActionAt               string        `json:"last_action_at"`
	SuggestedRepair            string        `json:"suggested_repair"`
	LastGoodStatus             string        `json:"last_good_status"`
	LastGoodHealth             string        `json:"last_good_health"`
	LastGoodAt                 string        `json:"last_good_at"`
	LastGoodSummary            string        `json:"last_good_summary"`
	Category                   string        `json:"category"`
	Units                      []ServiceUnit `json:"units"`
}

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

type ServiceUnit struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Health  string `json:"health"`
}

type userRow struct {
	Name  string
	Role  string
	Perms []string
}

// ── Model ───────────────────────────────────────────────────────────

type model struct {
	currentView        viewID
	cursor             int
	logSourceCursor    int
	statusSection      int
	ownershipCursor    int
	width              int
	height             int
	info               sysinfo.Info
	interfaces         []sysinfo.NetIface
	services           []ServiceInfo
	platform           PlatformInfo
	diagnostics        DiagnosticsInfo
	users              []userRow
	currentUser        string
	newUserRole        string
	setupAdmin         string
	userFormOpen       bool
	userFormEdit       bool
	userFormField      int
	userFormTarget     string
	userFormName       string
	userFormRole       string
	userFormPerms      string
	commandMode        bool
	commandInput       string
	commandStatus      string
	commandStatusLevel string
	controlPlaneOnline bool
	quitting           bool
}

var menuItems = []string{
	"System Status",
	"Services",
	"Diagnostics",
	"Users",
	"Logs",
	"Network",
	"Exit",
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
