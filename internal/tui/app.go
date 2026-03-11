package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
	"github.com/crateos/crateos/internal/takeover"
	"github.com/crateos/crateos/internal/users"
)

func (m model) controlPlaneMode() string {
	if m.controlPlaneOnline {
		return "agent-live"
	}
	return "fallback-local"
}

func readInitialPrimerHostname() string {
	cfg, err := config.Load()
	if err == nil {
		if hostname := strings.TrimSpace(cfg.CrateOS.Platform.Hostname); hostname != "" {
			return hostname
		}
	}
	info := sysinfo.Gather()
	if hostname := strings.TrimSpace(info.Hostname); hostname != "" {
		return hostname
	}
	return "crateos"
}

func platformInfoHostname(platformInfo PlatformInfo, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return readInitialPrimerHostname()
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
	if len(cfg.Users.Users) > 0 {
		return false
	}
	if cfg.Users.Roles == nil {
		cfg.Users.Roles = map[string]config.Role{}
	}
	if _, ok := cfg.Users.Roles["admin"]; !ok {
		cfg.Users.Roles["admin"] = config.Role{
			Description: "Full platform access including break-glass shell",
			Permissions: []string{"*"},
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
	m.refreshPrimerState()
	if m.primerRequired {
		m.currentView = viewSetup
	}
	return true
}

func (m *model) refreshPrimerStatusMessage() {
	m.refreshPrimerState()
	if m.primerRequired {
		m.currentView = viewSetup
		m.setCommandWarn("primer checks refreshed; blocking setup remains")
		return
	}
	m.setCommandOK("primer complete: console unlocked")
}

func (m *model) savePrimerIdentity() bool {
	hostname := strings.TrimSpace(m.setupHostname)
	if hostname == "" {
		return false
	}
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	cfg.CrateOS.Platform.Hostname = hostname
	if err := config.SaveCrateOS(cfg); err != nil {
		return false
	}
	m.setupHostname = hostname
	m.refreshPrimerState()
	return true
}

func (m *model) applyPrimerTakeover() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	username := ""
	if len(cfg.Users.Users) > 0 {
		username = strings.TrimSpace(cfg.Users.Users[0].Name)
	}
	return takeover.RepairLocalInstallContract(username) == nil
}

func (m *model) provisionPrimerUsers() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	if _, _, err := users.ProvisionUsers(cfg); err != nil {
		return false
	}
	return true
}

// NewModel creates the initial TUI model with fresh system info (API first, fallback local).
func NewModel() model {
	startUser := selectInitialUser()
	if info, svcs, platformInfo, diagnostics, usr := fetchStatusViaAPI(startUser); info != nil {
		users := fetchUsersViaAPI(startUser)
		m := model{
			currentView:        viewMenu,
			info:               *info,
			interfaces:         sysinfo.NetworkInterfaces(),
			services:           svcs,
			platform:           platformInfo,
			diagnostics:        diagnostics,
			users:              users,
			currentUser:        usr,
			newUserRole:        "staff",
			setupField:         0,
			setupAdmin:         "admin",
			setupHostname:      strings.TrimSpace(platformInfoHostname(platformInfo, info.Hostname)),
			controlPlaneOnline: true,
		}
		m.refreshPrimerState()
		if m.primerRequired {
			m.currentView = viewSetup
		}
		return m
	}
	users := fetchUsersFromConfig()
	m := model{
		currentView:        viewMenu,
		info:               sysinfo.Gather(),
		interfaces:         sysinfo.NetworkInterfaces(),
		services:           gatherServices(),
		platform:           readFallbackPlatformState(),
		diagnostics:        readFallbackDiagnostics(),
		users:              users,
		currentUser:        startUser,
		newUserRole:        "staff",
		setupField:         0,
		setupAdmin:         "admin",
		setupHostname:      strings.TrimSpace(readInitialPrimerHostname()),
		controlPlaneOnline: false,
	}
	m.refreshPrimerState()
	if m.primerRequired {
		m.currentView = viewSetup
	}
	return m
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
			m.setCommandWarn("interrupt ignored; use authenticated break-glass access for shell entry")
			return m, nil
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
	if m.primerRequired && m.currentView != viewSetup {
		m.currentView = viewSetup
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

func (m *model) refreshPrimerState() {
	checks := make([]primerCheck, 0, 5)
	installedMarker := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(installedMarker); err == nil {
		checks = append(checks, primerCheck{Label: "Install marker", OK: true, Details: installedMarker})
	} else {
		checks = append(checks, primerCheck{Label: "Install marker", OK: false, Details: "missing " + installedMarker})
	}
	cfg, err := config.Load()
	if err != nil {
		checks = append(checks, primerCheck{Label: "Config load", OK: false, Details: err.Error()})
		m.primerChecks = checks
		m.primerRequired = true
		return
	}
	admins := 0
	firstAdmin := ""
	for _, u := range cfg.Users.Users {
		if strings.TrimSpace(u.Role) == "admin" && strings.TrimSpace(u.Name) != "" {
			admins++
			if firstAdmin == "" {
				firstAdmin = strings.TrimSpace(u.Name)
			}
		}
	}
	if admins > 0 {
		checks = append(checks, primerCheck{Label: "Initial admin", OK: true, Details: "admins configured"})
	} else {
		checks = append(checks, primerCheck{Label: "Initial admin", OK: false, Details: "no admin in users.yaml"})
	}
	for _, check := range takeover.EvaluateLocalInstallContract(firstAdmin) {
		checks = append(checks, primerCheck{
			Label:   check.Label,
			OK:      check.OK,
			Details: check.Details,
		})
	}
	if strings.TrimSpace(cfg.CrateOS.Platform.Hostname) != "" {
		checks = append(checks, primerCheck{Label: "Machine identity", OK: true, Details: cfg.CrateOS.Platform.Hostname})
	} else {
		checks = append(checks, primerCheck{Label: "Machine identity", OK: false, Details: "hostname not set in crateos.yaml"})
	}
	m.primerChecks = checks
	m.primerRequired = false
	for _, check := range checks {
		if !check.OK {
			m.primerRequired = true
			break
		}
	}
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
