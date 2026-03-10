package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/api"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
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

func (m model) executeSystemCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	switch mod {
	case "refresh", "reload":
		m.refreshOverview()
		m.setCommandOK("state refreshed")
		return m, nil
	case "dos2unix", "normalize", "lineendings":
		scope := "config"
		if len(params) > 0 {
			scope = strings.ToLower(strings.TrimSpace(params[0]))
		}
		count, err := runDos2UnixHealth(scope)
		if err != nil {
			m.setCommandError("dos2unix failed: " + err.Error())
			return m, nil
		}
		m.setCommandOK(fmt.Sprintf("dos2unix normalized files: %d (scope:%s)", count, scope))
		return m, nil
	case "ftp-complete", "upload-complete":
		target := strings.TrimSpace(strings.Join(params, " "))
		if target == "" {
			m.setCommandWarn("usage: system ftp-complete <path>")
			return m, nil
		}
		// Note: FTP upload completion is handled by agent reconciliation
		// Direct user action via TUI is not necessary
		m.setCommandWarn("ftp-complete: use agent reconciliation instead")
		return m, nil
	default:
		m.setCommandWarn("usage: system <refresh|dos2unix [config|services|all]|ftp-complete <path>>")
		return m, nil
	}
}

func (m model) controlPlaneMode() string {
	if m.controlPlaneOnline {
		return "agent-live"
	}
	return "fallback-local"
}

func (m *model) requireLiveControlPlane(action string) bool {
	if m.controlPlaneOnline {
		return true
	}
	if strings.TrimSpace(action) == "" {
		action = "write action"
	}
	m.setCommandWarn(action + " unavailable: control plane offline, console is using fallback state")
	return false
}

func resolveUserTargets(users []userRow, defaultUser userRow, raw string) ([]userRow, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if strings.TrimSpace(defaultUser.Name) != "" {
			return []userRow{defaultUser}, nil
		}
		return []userRow{}, nil
	}
	parts := strings.Split(raw, ",")
	out := []userRow{}
	missing := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matched := false
		for _, u := range users {
			if strings.EqualFold(u.Name, part) {
				key := strings.ToLower(strings.TrimSpace(u.Name))
				if _, ok := seen[key]; !ok {
					out = append(out, u)
					seen[key] = struct{}{}
				}
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, part)
		}
	}
	return out, missing
}

func userNames(users []userRow) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if strings.TrimSpace(u.Name) != "" {
			out = append(out, u.Name)
		}
	}
	return out
}

func runDos2UnixHealth(scope string) (int, error) {
	if runtime.GOOS != "linux" {
		return 0, fmt.Errorf("dos2unix health check is only supported on linux hosts")
	}
	paths := []string{}
	switch scope {
	case "config":
		paths = []string{platform.CratePath("config")}
	case "services":
		paths = []string{platform.CratePath("services")}
	case "all":
		paths = []string{platform.CratePath("config"), platform.CratePath("services"), platform.CratePath("modules")}
	default:
		return 0, fmt.Errorf("unsupported scope: %s", scope)
	}
	targets := []string{}
	for _, root := range paths {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			normalized, normalizeErr := config.NormalizeFileIfNeeded(path)
			if normalizeErr != nil {
				return normalizeErr
			}
			if normalized {
				targets = append(targets, path)
			}
			return nil
		}); err != nil {
			return len(targets), err
		}
	}
	return len(targets), nil
}

func (m model) updateStatus(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.statusSection > 0 {
				m.statusSection--
			}
		case "down", "j":
			if m.statusSection < len(statusPanelItems())-1 {
				m.statusSection++
			}
		case "1":
			m.statusSection = 0
		case "2":
			m.statusSection = 1
		case "3":
			m.statusSection = 2
		}
	}
	return m, nil
}

func resolveServiceTargets(services []ServiceInfo, defaultService ServiceInfo, raw string) ([]ServiceInfo, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ServiceInfo{defaultService}, nil
	}
	if strings.EqualFold(raw, "all") {
		out := make([]ServiceInfo, 0, len(services))
		for _, svc := range services {
			out = append(out, svc)
		}
		return out, nil
	}
	parts := strings.Split(raw, ",")
	out := []ServiceInfo{}
	missing := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matched := false
		for _, svc := range services {
			if strings.EqualFold(svc.Name, part) || strings.EqualFold(svc.DisplayName, part) {
				key := strings.ToLower(strings.TrimSpace(svc.Name))
				if _, ok := seen[key]; !ok {
					out = append(out, svc)
					seen[key] = struct{}{}
				}
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, part)
		}
	}
	return out, missing
}

func tokenizeCommandInput(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	args := []string{}
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && unicode.IsSpace(rune(ch)) {
			flush()
			continue
		}
		b.WriteByte(ch)
	}
	flush()
	return args
}
func (m model) executeStatusCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewStatus {
		m.enterStatusView()
	}
	switch mod {
	case "list":
		m.setCommandInfo("status sections: system, services, platform")
		return m, nil
	case "":
		m.setCommandOK("route: status")
	case "system", "sys", "1":
		m.statusSection = 0
		m.setCommandOK("status section: system")
	case "services", "svc", "2":
		m.statusSection = 1
		m.setCommandOK("status section: services")
	case "platform", "plat", "3":
		m.statusSection = 2
		m.setCommandOK("status section: platform")
	case "next":
		if m.statusSection < len(statusPanelItems())-1 {
			m.statusSection++
		}
		m.setCommandInfo("status section advanced")
	case "prev":
		if m.statusSection > 0 {
			m.statusSection--
		}
		m.setCommandInfo("status section reversed")
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: status select <system|services|platform|1|2|3>")
			return m, nil
		}
		return m.executeStatusCommand(strings.ToLower(params[0]), params[1:])
	default:
		m.setCommandWarn("usage: status <system|services|platform|next|prev>")
	}
	return m, nil
}

func (m model) commandLaneEnabled() bool {
	if m.currentView == viewUsers && m.userFormOpen {
		return false
	}
	return true
}
func (m *model) setCommandStatus(level, message string) {
	m.commandStatusLevel = level
	m.commandStatus = message
}

func (m *model) setCommandInfo(message string) {
	m.setCommandStatus("info", message)
}

func (m *model) setCommandOK(message string) {
	m.setCommandStatus("ok", message)
}

func (m *model) setCommandWarn(message string) {
	m.setCommandStatus("warn", message)
}

func (m *model) setCommandError(message string) {
	m.setCommandStatus("error", message)
}

func (m model) commandHelpText() string {
	switch m.currentView {
	case viewServices:
		return "services help: svc <list|enable|start|stop|disable|install|uninstall|restart|next|prev|select> [service|service1,service2|all], or <service> <action>"
	case viewUsers:
		return "users help: user <list|add|rename|set|role|perms|delete|next|prev|select> [name|name1,name2]"
	case viewLogs:
		return "logs help: log <list|next|prev|select> [service|service1,service2], log service <list|next|prev|select> [service|service1,service2], log source <list|next|prev|select> [source|source1,source2]"
	case viewNetwork:
		return "network help: net <list|next|prev|select> [iface|iface1,iface2], nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewDiagnostics:
		return "diagnostics help: diag <summary|verification|ownership|config|actor [target]|focus <target>|next|prev|select>, list actors, nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewStatus:
		return "status help: status <system|services|platform|next|prev>, nav <view>, system <refresh|ftp-complete <path|dir>>"
	case viewSetup:
		return "setup help: bootstrap <admin> (setup is locked until bootstrap completes)"
	default:
		return "grammar: command mod params... | commands: help list nav status diag svc user log net bootstrap system back quit | system: refresh | dos2unix [config|services|all] | ftp-complete <path|dir> | service-as-command: <service> <action> | chaining: cmd1; cmd2 or cmd1 && cmd2"
	}
}

func (m model) updateCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.commandMode = false
		m.commandInput = ""
		m.setCommandWarn("command mode canceled")
	case "enter":
		return m.executeCommandInput()
	case "backspace":
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.commandInput += strings.ToLower(msg.String())
		}
	}
	return m, nil
}

func (m model) executeCommandInput() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.commandInput)
	m.commandMode = false
	m.commandInput = ""
	if raw == "" {
		m.setCommandWarn("empty command")
		return m, nil
	}
	parts := splitCommandChain(raw)
	if len(parts) > 1 {
		var lastCmd tea.Cmd
		for i, part := range parts {
			var (
				next tea.Model
				cmd  tea.Cmd
			)
			next, cmd = m.executeSingleCommand(part)
			resolved, ok := next.(model)
			if !ok {
				if ptr, ok := next.(*model); ok && ptr != nil {
					resolved = *ptr
				} else {
					m.setCommandError(fmt.Sprintf("internal model mismatch at step %d", i+1))
					return m, cmd
				}
			}
			m = resolved
			if cmd != nil {
				lastCmd = cmd
			}
			if m.commandStatusLevel == "error" {
				m.setCommandError(fmt.Sprintf("chain aborted at step %d: %s", i+1, part))
				break
			}
		}
		return m, lastCmd
	}
	return m.executeSingleCommand(raw)
}

func splitCommandChain(raw string) []string {
	out := []string{}
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		part := strings.TrimSpace(b.String())
		if part != "" {
			out = append(out, part)
		}
		b.Reset()
	}
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}
		if !inSingle && !inDouble {
			if ch == ';' {
				flush()
				continue
			}
			if ch == '&' && i+1 < len(raw) && raw[i+1] == '&' {
				flush()
				i++
				continue
			}
		}
		b.WriteByte(ch)
	}
	flush()
	return out
}

func parseCSVTargets(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}

func (m model) executeSingleCommand(raw string) (tea.Model, tea.Cmd) {
	args := tokenizeCommandInput(raw)
	if len(args) == 0 {
		m.setCommandWarn("empty command")
		return m, nil
	}
	cmd := strings.ToLower(args[0])
	if m.currentView == viewSetup {
		switch cmd {
		case "help", "?", "bootstrap", "quit", "exit":
			// allowed during setup lock
		default:
			m.setCommandWarn("bootstrap required before control-plane commands")
			return m, nil
		}
	}
	mod := ""
	params := []string{}
	if len(args) > 1 {
		mod = strings.ToLower(args[1])
		params = args[2:]
	}
	switch cmd {
	case "help", "?":
		m.setCommandInfo(m.commandHelpText())
	case "list":
		switch mod {
		case "", "auto":
			switch m.currentView {
			case viewServices:
				return m.executeServiceCommand("list", nil)
			case viewUsers:
				return m.executeUserCommand("list", nil)
			case viewLogs:
				return m.executeLogCommand("list", nil)
			case viewNetwork:
				return m.executeNetCommand("list", nil)
			case viewDiagnostics:
				return m.executeDiagnosticsCommand("list", nil)
			case viewStatus:
				return m.executeStatusCommand("list", nil)
			default:
				m.setCommandWarn("usage: list <services|users|logs|sources|net|status|diagnostics>")
				return m, nil
			}
		case "services", "svc":
			return m.executeServiceCommand("list", nil)
		case "users", "user":
			return m.executeUserCommand("list", nil)
		case "logs", "log":
			return m.executeLogCommand("list", nil)
		case "sources", "source":
			return m.executeLogSourceSubcommand([]string{"list"})
		case "net", "network", "interfaces":
			return m.executeNetCommand("list", nil)
		case "status":
			return m.executeStatusCommand("list", nil)
		case "diagnostics", "diag":
			return m.executeDiagnosticsCommand("list", nil)
		default:
			m.setCommandWarn("usage: list <services|users|logs|sources|net|status|diagnostics>")
			return m, nil
		}
	case "install", "uninstall", "enable", "start", "stop", "disable", "restart":
		action := normalizeServiceAction(cmd)
		target := strings.TrimSpace(strings.Join(append([]string{mod}, params...), " "))
		if target == "" {
			m.setCommandWarn("usage: <install|uninstall|enable|start|stop|disable|restart> <service|service1,service2|all>")
			return m, nil
		}
		return m.executeServiceLifecycleCommand(action, target)
	case "bootstrap":
		name := strings.TrimSpace(strings.Join(append([]string{mod}, params...), " "))
		if name == "" {
			m.setCommandWarn("usage: bootstrap <admin>")
			return m, nil
		}
		if !m.requireLiveControlPlane("bootstrap") {
			return m, nil
		}
		if ok := m.bootstrapAdmin(name); !ok {
			m.setCommandError("bootstrap failed: " + name)
			return m, nil
		}
		m.setCommandOK("bootstrap complete: " + name)
		return m, nil
	case "status":
		return m.executeStatusCommand(mod, params)
	case "diag", "diagnostics":
		return m.executeDiagnosticsCommand(mod, params)
	case "svc":
		return m.executeServiceCommand(mod, params)
	case "user":
		return m.executeUserCommand(mod, params)
	case "log":
		return m.executeLogCommand(mod, params)
	case "net":
		return m.executeNetCommand(mod, params)
	case "nav", "route", "go":
		return m.executeNavigationCommand(mod)
	case "system":
		return m.executeSystemCommand(mod, params)
	case "back":
		m.enterMenuView()
		m.setCommandOK("route: menu")
	case "quit", "exit":
		m.quitting = true
		return m, tea.Quit
	default:
		if action := normalizeServiceAction(mod); action != "" {
			target := cmd
			return m.executeServiceLifecycleCommand(action, target)
		}
		m.setCommandError("unknown command: " + raw + " (type: help)")
	}
	return m, nil
}

func (m model) executeNavigationCommand(target string) (tea.Model, tea.Cmd) {
	switch target {
	case "setup":
		m.currentView = viewSetup
		m.setCommandOK("route: setup")
	case "menu", "home":
		m.enterMenuView()
		m.setCommandOK("route: menu")
	case "status":
		m.enterStatusView()
		m.setCommandOK("route: status")
	case "diagnostics", "diag":
		m.enterDiagnosticsView()
		m.setCommandOK("route: diagnostics")
	case "services", "service", "svc":
		m.enterServicesView()
		m.setCommandOK("route: services")
	case "users", "userdir":
		m.enterUsersView()
		m.setCommandOK("route: users")
	case "logs", "log":
		m.enterLogsView()
		m.setCommandOK("route: logs")
	case "network", "net":
		m.enterNetworkView()
		m.setCommandOK("route: network")
	default:
		m.setCommandWarn("usage: nav <setup|menu|status|diagnostics|services|users|logs|network>")
	}
	return m, nil
}

func (m model) executeServiceCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewServices {
		m.enterServicesView()
	}
	switch mod {
	case "list":
		if len(m.services) == 0 {
			m.setCommandWarn("services: none")
			return m, nil
		}
		names := make([]string, 0, len(m.services))
		for _, s := range m.services {
			name := strings.TrimSpace(s.Name)
			if name == "" {
				name = strings.TrimSpace(s.DisplayName)
			}
			if name != "" {
				names = append(names, name)
			}
		}
		m.setCommandInfo("services: " + strings.Join(names, ", "))
		return m, nil
	case "enable", "start", "stop", "disable", "install", "uninstall", "restart":
		target := ""
		if len(params) > 0 {
			target = strings.Join(params, " ")
		}
		action := normalizeServiceAction(mod)
		return m.executeServiceLifecycleCommand(action, target)
	case "next":
		if len(m.services) == 0 {
			m.setCommandWarn("no services available")
			return m, nil
		}
		if m.cursor < len(m.services)-1 {
			m.cursor++
		}
		m.setCommandInfo("service selector advanced")
		return m, nil
	case "prev":
		if len(m.services) == 0 {
			m.setCommandWarn("no services available")
			return m, nil
		}
		if m.cursor > 0 {
			m.cursor--
		}
		m.setCommandInfo("service selector reversed")
		return m, nil
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: svc select <service|service1,service2>")
			return m, nil
		}
		targets := parseCSVTargets(strings.Join(params, " "))
		if len(targets) == 0 {
			m.setCommandWarn("usage: svc select <service|service1,service2>")
			return m, nil
		}
		selected := []string{}
		missing := []string{}
		for _, target := range targets {
			found := false
			for i, s := range m.services {
				if strings.EqualFold(s.Name, target) || strings.EqualFold(s.DisplayName, target) {
					m.cursor = i
					selected = append(selected, s.Name)
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, target)
			}
		}
		if len(selected) == 0 {
			m.setCommandError("service not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(missing) > 0 {
			m.setCommandWarn("svc select partial: ok=" + strings.Join(selected, ",") + " missing=" + strings.Join(missing, ","))
			return m, nil
		}
		m.setCommandOK("services selected: " + strings.Join(selected, ","))
		return m, nil
	case "":
		m.setCommandOK("route: services")
		return m, nil
	default:
		m.setCommandWarn("usage: svc <list|enable|start|stop|disable|install|uninstall|restart|next|prev|select> [service|service1,service2|all]")
		return m, nil
	}
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
func (m model) executeLogCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewLogs {
		m.enterLogsView()
	}
	if len(m.services) == 0 {
		m.setCommandWarn("no log services available")
		return m, nil
	}
	switch mod {
	case "list":
		names := make([]string, 0, len(m.services))
		for _, s := range m.services {
			if strings.TrimSpace(s.Name) != "" {
				names = append(names, s.Name)
			}
		}
		if len(names) == 0 {
			m.setCommandWarn("log services: none")
			return m, nil
		}
		m.setCommandInfo("log services: " + strings.Join(names, ", "))
		return m, nil
	case "", "service":
		return m.executeLogServiceSubcommand(params)
	case "source":
		return m.executeLogSourceSubcommand(params)
	case "next":
		return m.executeLogServiceSubcommand([]string{"next"})
	case "prev":
		return m.executeLogServiceSubcommand([]string{"prev"})
	case "select":
		return m.executeLogServiceSubcommand(append([]string{"select"}, params...))
	default:
		m.setCommandWarn("usage: log <list|next|prev|select> [service|service1,service2] | log source <list|next|prev|select> [source|source1,source2]")
		return m, nil
	}
}

func (m model) executeLogServiceSubcommand(params []string) (tea.Model, tea.Cmd) {
	if len(params) == 0 {
		m.setCommandOK("route: logs")
		return m, nil
	}
	action := strings.ToLower(params[0])
	switch action {
	case "list":
		names := make([]string, 0, len(m.services))
		for _, s := range m.services {
			if strings.TrimSpace(s.Name) != "" {
				names = append(names, s.Name)
			}
		}
		if len(names) == 0 {
			m.setCommandWarn("log services: none")
			return m, nil
		}
		m.setCommandInfo("log services: " + strings.Join(names, ", "))
		return m, nil
	case "next":
		if m.cursor < len(m.services)-1 {
			m.cursor++
		}
		m.logSourceCursor = 0
		m.setCommandInfo("log service selector advanced")
	case "prev":
		if m.cursor > 0 {
			m.cursor--
		}
		m.logSourceCursor = 0
		m.setCommandInfo("log service selector reversed")
	case "select":
		if len(params) < 2 {
			m.setCommandWarn("usage: log select <service|service1,service2>")
			return m, nil
		}
		targets := parseCSVTargets(strings.Join(params[1:], " "))
		if len(targets) == 0 {
			m.setCommandWarn("usage: log select <service|service1,service2>")
			return m, nil
		}
		selected := []string{}
		missing := []string{}
		for _, target := range targets {
			found := false
			for i, s := range m.services {
				if strings.EqualFold(s.Name, target) || strings.EqualFold(s.DisplayName, target) {
					m.cursor = i
					m.logSourceCursor = 0
					selected = append(selected, s.Name)
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, target)
			}
		}
		if len(selected) == 0 {
			m.setCommandError("service not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(missing) > 0 {
			m.setCommandWarn("log select partial: ok=" + strings.Join(selected, ",") + " missing=" + strings.Join(missing, ","))
			return m, nil
		}
		m.setCommandOK("log services selected: " + strings.Join(selected, ","))
	default:
		m.setCommandWarn("usage: log <list|next|prev|select> [service|service1,service2]")
	}
	return m, nil
}

func (m model) executeLogSourceSubcommand(params []string) (tea.Model, tea.Cmd) {
	sources := logSourcesForService(m.currentLogService())
	if len(sources) == 0 {
		m.setCommandWarn("no log sources available")
		return m, nil
	}
	if len(params) == 0 {
		m.setCommandWarn("usage: log source <list|next|prev|select> [source]")
		return m, nil
	}
	action := strings.ToLower(params[0])
	switch action {
	case "list":
		names := make([]string, 0, len(sources))
		for _, source := range sources {
			names = append(names, sourceDisplayLabel(source))
		}
		m.setCommandInfo("log sources: " + strings.Join(names, ", "))
		return m, nil
	case "next":
		if m.logSourceCursor < len(sources)-1 {
			m.logSourceCursor++
		}
		m.setCommandInfo("log source advanced")
	case "prev":
		if m.logSourceCursor > 0 {
			m.logSourceCursor--
		}
		m.setCommandInfo("log source reversed")
	case "select":
		if len(params) < 2 {
			m.setCommandWarn("usage: log source select <source|source1,source2>")
			return m, nil
		}
		targets := parseCSVTargets(strings.Join(params[1:], " "))
		if len(targets) == 0 {
			m.setCommandWarn("usage: log source select <source|source1,source2>")
			return m, nil
		}
		selected := []string{}
		missing := []string{}
		for _, target := range targets {
			found := false
			for i, source := range sources {
				if strings.EqualFold(source.Label, target) || strings.EqualFold(source.Path, target) {
					m.logSourceCursor = i
					selected = append(selected, sourceDisplayLabel(source))
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, target)
			}
		}
		if len(selected) == 0 {
			m.setCommandError("log source not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(missing) > 0 {
			m.setCommandWarn("log source select partial: ok=" + strings.Join(selected, ",") + " missing=" + strings.Join(missing, ","))
			return m, nil
		}
		m.setCommandOK("log sources selected: " + strings.Join(selected, ","))
	default:
		m.setCommandWarn("usage: log source <list|next|prev|select> [source|source1,source2]")
	}
	return m, nil
}

func (m model) executeNetCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewNetwork {
		m.enterNetworkView()
	}
	if len(m.interfaces) == 0 {
		m.setCommandWarn("no interfaces available")
		return m, nil
	}
	switch mod {
	case "list":
		names := make([]string, 0, len(m.interfaces))
		for _, iface := range m.interfaces {
			if strings.TrimSpace(iface.Name) != "" {
				names = append(names, iface.Name)
			}
		}
		if len(names) == 0 {
			m.setCommandWarn("interfaces: none")
			return m, nil
		}
		m.setCommandInfo("interfaces: " + strings.Join(names, ", "))
		return m, nil
	case "":
		m.setCommandOK("route: network")
	case "next":
		if m.cursor < len(m.interfaces)-1 {
			m.cursor++
		}
		m.setCommandInfo("network selector advanced")
	case "prev":
		if m.cursor > 0 {
			m.cursor--
		}
		m.setCommandInfo("network selector reversed")
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: net select <interface|iface1,iface2>")
			return m, nil
		}
		targets := parseCSVTargets(strings.Join(params, " "))
		if len(targets) == 0 {
			m.setCommandWarn("usage: net select <interface|iface1,iface2>")
			return m, nil
		}
		selected := []string{}
		missing := []string{}
		for _, target := range targets {
			found := false
			for i, iface := range m.interfaces {
				if strings.EqualFold(iface.Name, target) {
					m.cursor = i
					selected = append(selected, iface.Name)
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, target)
			}
		}
		if len(selected) == 0 {
			m.setCommandError("interface not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(missing) > 0 {
			m.setCommandWarn("net select partial: ok=" + strings.Join(selected, ",") + " missing=" + strings.Join(missing, ","))
			return m, nil
		}
		m.setCommandOK("interfaces selected: " + strings.Join(selected, ","))
	default:
		m.setCommandWarn("usage: net <list|next|prev|select> [interface|iface1,iface2]")
	}
	return m, nil
}

func (m model) executeServiceLifecycleCommand(cmd, target string) (tea.Model, tea.Cmd) {
	if m.currentView != viewServices {
		m.enterServicesView()
	}
	if !m.requireLiveControlPlane("service lifecycle") {
		return m, nil
	}
	if len(m.services) == 0 {
		m.setCommandWarn("no services available")
		return m, nil
	}
	target = strings.TrimSpace(target)
		targetServices, missing := resolveServiceTargets(m.services, m.services[m.currentServiceIndex()], target)
		if len(missing) > 0 {
			m.setCommandError("service not found: " + strings.Join(missing, ", "))
			return m, nil
		}
		if len(targetServices) == 0 {
			m.setCommandWarn("no services resolved for action")
			return m, nil
		}
		applied := []string{}
		failed := []string{}
		for _, svc := range targetServices {
			var err error
			switch cmd {
			case "enable":
				err = enableServiceDirect(svc.Name)
			case "start":
				err = startServiceDirect(svc.Name)
			case "stop":
				err = stopServiceDirect(svc.Name)
			case "disable":
				err = disableServiceDirect(svc.Name)
			case "restart":
				err = stopServiceDirect(svc.Name)
				if err == nil {
					err = startServiceDirect(svc.Name)
				}
			}
			if err != nil {
				failed = append(failed, svc.Name)
				continue
			}
			applied = append(applied, svc.Name)
		}
	if len(applied) == 0 {
		m.setCommandError(fmt.Sprintf("%s failed for %s", cmd, strings.Join(failed, ", ")))
		return m, nil
	}
	m.refreshServices()
	if len(failed) > 0 {
		m.setCommandWarn(fmt.Sprintf("%s partial: ok=%s failed=%s", cmd, strings.Join(applied, ","), strings.Join(failed, ",")))
		return m, nil
	}
	m.setCommandOK(fmt.Sprintf("%s applied to %s", cmd, strings.Join(applied, ",")))
	return m, nil
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

func (m model) updateNetwork(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.interfaces)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

type PlatformInfo struct {
	GeneratedAt string            `json:"generated_at"`
	Adapters    []PlatformAdapter `json:"adapters"`
}

type DiagnosticsInfo struct {
	Config       ConfigDiagnosticsInfo       `json:"config"`
	Verification VerificationDiagnosticsInfo `json:"verification"`
	Ownership    OwnershipDiagnosticsInfo    `json:"ownership"`
}

type ConfigDiagnosticsInfo struct {
	GeneratedAt   string                 `json:"generated_at"`
	Tracked       int                    `json:"tracked"`
	Monitored     int                    `json:"monitored"`
	Unmonitored   int                    `json:"unmonitored"`
	ExternalEdits int                    `json:"external_edits"`
	Files         []ConfigDiagnosticFile `json:"files"`
}

type ConfigDiagnosticFile struct {
	File          string `json:"file"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Monitoring    string `json:"monitoring"`
	LastWriter    string `json:"last_writer"`
	LastSeenAt    string `json:"last_seen_at"`
	LastChangedAt string `json:"last_changed_at"`
}

type VerificationDiagnosticsInfo struct {
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	Missing        []string `json:"missing"`
	Warnings       []string `json:"warnings"`
	PlatformState  string   `json:"platform_state"`
	Readiness      string   `json:"readiness"`
	StorageState   string   `json:"storage_state"`
	OwnershipState string   `json:"ownership_state"`
	AgentSocket    bool     `json:"agent_socket"`
	AdminPresent   bool     `json:"admin_present"`
}

type OwnershipDiagnosticsInfo struct {
	GeneratedAt string                     `json:"generated_at"`
	Managed     int                        `json:"managed"`
	Provisioned int                        `json:"provisioned"`
	Pending     int                        `json:"pending"`
	Blocked     int                        `json:"blocked"`
	Active      int                        `json:"active"`
	Retired     int                        `json:"retired"`
	Claims      []OwnershipDiagnosticClaim `json:"claims"`
	Workloads   []ActorLifecycleDiagnostic `json:"workloads"`
}

type OwnershipDiagnosticClaim struct {
	Crate     string `json:"crate"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	User      string `json:"user"`
	Group     string `json:"group"`
	Home      string `json:"home"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
	RetiredAt string `json:"retired_at"`
}

type ActorLifecycleDiagnostic struct {
	Crate                 string                          `json:"crate"`
	ActorName             string                          `json:"actor_name"`
	ActorType             string                          `json:"actor_type"`
	ActorID               string                          `json:"actor_id"`
	ActorUser             string                          `json:"actor_user"`
	ActorGroup            string                          `json:"actor_group"`
	ActorHome             string                          `json:"actor_home"`
	Provisioning          string                          `json:"provisioning"`
	ProvisioningError     string                          `json:"provisioning_error"`
	ProvisioningUpdatedAt string                          `json:"provisioning_updated_at"`
	LastSuccessAt         string                          `json:"last_success_at"`
	LastFailureAt         string                          `json:"last_failure_at"`
	ProvisioningStatePath string                          `json:"provisioning_state_path"`
	OwnershipStatus       string                          `json:"ownership_status"`
	OwnershipUpdatedAt    string                          `json:"ownership_updated_at"`
	OwnershipRetiredAt    string                          `json:"ownership_retired_at"`
	RecentEvents          []ActorLifecycleEventDiagnostic `json:"recent_events"`
}

type ActorLifecycleEventDiagnostic struct {
	At           string `json:"at"`
	Provisioning string `json:"provisioning"`
	Error        string `json:"error"`
}

type PlatformAdapter struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Enabled       bool     `json:"enabled"`
	Status        string   `json:"status"`
	Health        string   `json:"health"`
	Summary       string   `json:"summary"`
	LastError     string   `json:"last_error"`
	Validation    string   `json:"validation"`
	ValidationErr string   `json:"validation_error"`
	Apply         string   `json:"apply"`
	ApplyErr      string   `json:"apply_error"`
	RenderedPaths []string `json:"rendered_paths"`
	NativeTargets []string `json:"native_targets"`
}

func readFallbackPlatformState() PlatformInfo {
	snapshot := state.LoadPlatformState()
	info := PlatformInfo{
		GeneratedAt: snapshot.GeneratedAt,
		Adapters:    make([]PlatformAdapter, 0, len(snapshot.Adapters)),
	}
	for _, adapter := range snapshot.Adapters {
		info.Adapters = append(info.Adapters, PlatformAdapter{
			Name:          adapter.Name,
			DisplayName:   adapter.DisplayName,
			Enabled:       adapter.Enabled,
			Status:        adapter.Status,
			Health:        adapter.Health,
			Summary:       adapter.Summary,
			LastError:     adapter.LastError,
			Validation:    adapter.Validation,
			ValidationErr: adapter.ValidationErr,
			Apply:         adapter.Apply,
			ApplyErr:      adapter.ApplyErr,
			RenderedPaths: append([]string(nil), adapter.RenderedPaths...),
			NativeTargets: append([]string(nil), adapter.NativeTargets...),
		})
	}
	return info
}

func readFallbackOwnershipDiagnostics() OwnershipDiagnosticsInfo {
	snapshot := state.LoadActorOwnershipState()
	info := OwnershipDiagnosticsInfo{
		GeneratedAt: strings.TrimSpace(snapshot.GeneratedAt),
		Active:      snapshot.Active,
		Retired:     snapshot.Retired,
		Claims:      make([]OwnershipDiagnosticClaim, 0, len(snapshot.Claims)),
		Workloads:   []ActorLifecycleDiagnostic{},
	}
	claimsByCrate := map[string]state.ActorOwnershipStateItem{}
	for _, claim := range snapshot.Claims {
		claimsByCrate[strings.TrimSpace(claim.Crate)] = claim
		info.Claims = append(info.Claims, OwnershipDiagnosticClaim{
			Crate:     claim.Crate,
			Name:      claim.Name,
			Type:      claim.Type,
			ID:        claim.ID,
			User:      claim.User,
			Group:     claim.Group,
			Home:      claim.Home,
			Status:    claim.Status,
			UpdatedAt: claim.UpdatedAt,
			RetiredAt: claim.RetiredAt,
		})
	}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		for _, svc := range cfg.Services.Services {
			if strings.TrimSpace(svc.Actor.Name) == "" && strings.TrimSpace(svc.Execution.Mode) == "" {
				continue
			}
			info.Managed++
			provisioning := state.LoadActorProvisioningState(svc.Name)
			item := ActorLifecycleDiagnostic{
				Crate:                 svc.Name,
				ActorName:             strings.TrimSpace(provisioning.Actor.Name),
				ActorType:             strings.TrimSpace(provisioning.Actor.Type),
				ActorID:               strings.TrimSpace(provisioning.Actor.ID),
				ActorUser:             strings.TrimSpace(provisioning.Actor.User),
				ActorGroup:            strings.TrimSpace(provisioning.Actor.Group),
				ActorHome:             strings.TrimSpace(provisioning.Actor.Home),
				Provisioning:          strings.TrimSpace(provisioning.Provisioning),
				ProvisioningError:     strings.TrimSpace(provisioning.Error),
				ProvisioningUpdatedAt: strings.TrimSpace(provisioning.GeneratedAt),
				LastSuccessAt:         strings.TrimSpace(provisioning.LastSuccessAt),
				LastFailureAt:         strings.TrimSpace(provisioning.LastFailureAt),
				ProvisioningStatePath: platform.CratePath("services", svc.Name, "runtime", "actor-provisioning.json"),
				RecentEvents:          make([]ActorLifecycleEventDiagnostic, 0, len(provisioning.Events)),
			}
			for _, event := range provisioning.Events {
				item.RecentEvents = append(item.RecentEvents, ActorLifecycleEventDiagnostic{
					At:           strings.TrimSpace(event.At),
					Provisioning: strings.TrimSpace(event.Provisioning),
					Error:        strings.TrimSpace(event.Error),
				})
			}
			if item.ActorName == "" {
				item.ActorName = strings.TrimSpace(svc.Actor.Name)
			}
			if claim, ok := claimsByCrate[strings.TrimSpace(svc.Name)]; ok {
				item.OwnershipStatus = strings.TrimSpace(claim.Status)
				item.OwnershipUpdatedAt = strings.TrimSpace(claim.UpdatedAt)
				item.OwnershipRetiredAt = strings.TrimSpace(claim.RetiredAt)
				if item.ActorName == "" {
					item.ActorName = strings.TrimSpace(claim.Name)
				}
				if item.ActorType == "" {
					item.ActorType = strings.TrimSpace(claim.Type)
				}
				if item.ActorID == "" {
					item.ActorID = strings.TrimSpace(claim.ID)
				}
				if item.ActorUser == "" {
					item.ActorUser = strings.TrimSpace(claim.User)
				}
				if item.ActorGroup == "" {
					item.ActorGroup = strings.TrimSpace(claim.Group)
				}
				if item.ActorHome == "" {
					item.ActorHome = strings.TrimSpace(claim.Home)
				}
			}
			switch item.Provisioning {
			case "provisioned":
				info.Provisioned++
			case "blocked":
				info.Blocked++
			default:
				info.Pending++
			}
			info.Workloads = append(info.Workloads, item)
		}
	}
	return info
}

func readFallbackVerificationDiagnostics() VerificationDiagnosticsInfo {
	info := VerificationDiagnosticsInfo{
		Status:   "ready",
		Missing:  []string{},
		Warnings: []string{},
	}
	requiredPaths := []struct {
		path  string
		label string
	}{
		{platform.CratePath("state", "installed.json"), "installed marker"},
		{platform.CratePath("state", "platform-state.json"), "platform state"},
		{platform.CratePath("state", "readiness-report.json"), "readiness report"},
		{platform.CratePath("state", "storage-state.json"), "storage state"},
		{platform.CratePath("state", "actor-ownership-state.json"), "actor ownership state"},
	}
	for _, item := range requiredPaths {
		if _, err := os.Stat(item.path); err != nil {
			info.Missing = append(info.Missing, item.label)
		}
	}
	if _, err := os.Stat(platform.AgentSocket); err == nil {
		info.AgentSocket = true
	}
	if rows := fetchUsersFromConfig(); len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(strings.TrimSpace(row.Role), "admin") {
				info.AdminPresent = true
				break
			}
		}
	}
	info.PlatformState = strings.TrimSpace(readFallbackPlatformState().GeneratedAt)
	if info.PlatformState == "" {
		info.Warnings = append(info.Warnings, "platform state not rendered yet")
	}
	if report, ok := readReadinessReport(); ok {
		info.Readiness = strings.TrimSpace(report.Status)
		if info.Readiness == "" {
			info.Readiness = "unknown"
		}
		if info.Readiness != "ready" {
			info.Warnings = append(info.Warnings, "readiness report is not ready")
		}
	} else {
		info.Warnings = append(info.Warnings, "readiness report unreadable")
	}
	if storage := state.LoadStorageState(); strings.TrimSpace(storage.GeneratedAt) != "" {
		info.StorageState = strings.TrimSpace(storage.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "storage posture not rendered yet")
	}
	if ownership := state.LoadActorOwnershipState(); strings.TrimSpace(ownership.GeneratedAt) != "" {
		info.OwnershipState = strings.TrimSpace(ownership.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "actor ownership state not rendered yet")
	}
	if !info.AgentSocket {
		info.Warnings = append(info.Warnings, "agent socket unavailable")
	}
	if !info.AdminPresent {
		info.Missing = append(info.Missing, "admin operator")
	}
	switch {
	case len(info.Missing) > 0:
		info.Status = "failed"
		info.Summary = "verification prerequisites missing"
	case len(info.Warnings) > 0:
		info.Status = "degraded"
		info.Summary = "verification surfaces present with warnings"
	default:
		info.Summary = "verification surfaces present"
	}
	return info
}

func readFallbackDiagnostics() DiagnosticsInfo {
	ledger, err := config.LoadConfigChangeLedger()
	if err != nil {
		return DiagnosticsInfo{
			Verification: readFallbackVerificationDiagnostics(),
			Ownership:    readFallbackOwnershipDiagnostics(),
		}
	}
	info := DiagnosticsInfo{
		Config: ConfigDiagnosticsInfo{
			GeneratedAt: ledger.GeneratedAt,
			Files:       make([]ConfigDiagnosticFile, 0, len(ledger.Files)),
		},
		Verification: readFallbackVerificationDiagnostics(),
		Ownership:    readFallbackOwnershipDiagnostics(),
	}
	for _, record := range ledger.Files {
		info.Config.Tracked++
		switch strings.TrimSpace(record.Monitoring) {
		case "unmonitored":
			info.Config.Unmonitored++
		default:
			info.Config.Monitored++
		}
		if strings.TrimSpace(record.LastWriter) == "external" {
			info.Config.ExternalEdits++
		}
		info.Config.Files = append(info.Config.Files, ConfigDiagnosticFile{
			File:          record.File,
			Path:          record.Path,
			Exists:        record.Exists,
			Monitoring:    record.Monitoring,
			LastWriter:    record.LastWriter,
			LastSeenAt:    record.LastSeenAt,
			LastChangedAt: record.LastChangedAt,
		})
	}
	return info
}

func (m *model) refreshOverview() {
	if info, svcs, platformInfo, diagnostics, _ := fetchStatusViaAPI(m.currentUser); info != nil {
		m.info = *info
		m.services = svcs
		m.platform = platformInfo
		m.diagnostics = diagnostics
		m.controlPlaneOnline = true
		return
	}
	m.refreshServices()
	m.platform = readFallbackPlatformState()
	m.diagnostics = readFallbackDiagnostics()
	m.info = sysinfo.Gather()
	m.controlPlaneOnline = false
}

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

func (m model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.logSourceCursor = 0
			}
		case "down", "j":
			if m.cursor < len(m.services)-1 {
				m.cursor++
				m.logSourceCursor = 0
			}
		case "right", "l", "tab":
			sources := logSourcesForService(m.currentLogService())
			if m.logSourceCursor < len(sources)-1 {
				m.logSourceCursor++
			}
		case "left", "h", "shift+tab":
			if m.logSourceCursor > 0 {
				m.logSourceCursor--
			}
		}
	}
	return m, nil
}

func (m model) currentLogService() ServiceInfo {
	if len(m.services) == 0 {
		return ServiceInfo{}
	}
	cursor := m.cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.services) {
		cursor = len(m.services) - 1
	}
	return m.services[cursor]
}

func (m model) currentLogSourceIndex() int {
	sources := logSourcesForService(m.currentLogService())
	if len(sources) == 0 {
		return 0
	}
	cursor := m.logSourceCursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(sources) {
		return len(sources) - 1
	}
	return cursor
}

func (m model) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "backspace":
			if len(m.setupAdmin) > 0 {
				m.setupAdmin = m.setupAdmin[:len(m.setupAdmin)-1]
			}
		case "enter":
			name := strings.TrimSpace(m.setupAdmin)
			if name == "" {
				return m, nil
			}
			if !m.requireLiveControlPlane("bootstrap") {
				return m, nil
			}
			if m.bootstrapAdmin(name) {
				m.setCommandOK("bootstrap complete: " + name)
			}
		default:
			if len(msg.String()) == 1 {
				m.setupAdmin += msg.String()
			}
		}
	}
	return m, nil
}

func (m model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.userFormOpen {
			switch msg.String() {
			case "esc":
				m.closeUserForm()
			case "tab", "down":
				m.userFormField = (m.userFormField + 1) % 3
			case "shift+tab", "up":
				m.userFormField = (m.userFormField + 2) % 3
			case "backspace":
				m.backspaceUserFormField()
			case "enter":
				name := strings.TrimSpace(m.userFormName)
				role := strings.TrimSpace(m.userFormRole)
				perms := parsePermList(m.userFormPerms)
				if name == "" || role == "" {
					return m, nil
				}
				if !m.requireLiveControlPlane("user update") {
					return m, nil
				}
				if m.userFormEdit {
					_ = updateUserDirect(m.userFormTarget, name, role, perms)
				} else {
					_ = addUserDirect(name, role, perms)
				}
				m.refreshUsers()
				m.closeUserForm()
			default:
				if len(msg.String()) == 1 {
					m.appendUserFormField(msg.String())
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "a":
			m.openAddUserForm()
		case "enter":
			if m.cursor < len(m.users) {
				m.openEditUserForm(m.users[m.cursor])
			}
		case "r": // cycle selected user's role, or next-add role if none selected
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user role cycle") {
					return m, nil
				}
				u := m.users[m.cursor]
				_ = updateUserDirect(u.Name, "", nextRole(u.Role), nil)
				m.refreshUsers()
			} else {
				m.newUserRole = nextRole(m.newUserRole)
			}
		case "p": // toggle selected user's perms preset
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user permissions update") {
					return m, nil
				}
				u := m.users[m.cursor]
				_ = updateUserDirect(u.Name, "", "", togglePermPreset(u.Perms))
				m.refreshUsers()
			}
		case "s": // set current user
			if m.cursor < len(m.users) {
				m.currentUser = m.users[m.cursor].Name
			}
		case "d": // delete user
			if m.cursor < len(m.users) {
				if !m.requireLiveControlPlane("user delete") {
					return m, nil
				}
				name := m.users[m.cursor].Name
				if name != "" {
					_ = deleteUserDirect(name)
					m.refreshUsers()
				}
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.users)-1 {
				m.cursor++
			}
		}
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

// updateGeneric handles ESC/backspace to return to the menu from any sub-view.
func (m model) updateGeneric(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		}
	}
	return m, nil
}

func (m model) updateServices(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "e": // enable
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = enableServiceDirect(name)
				m.refreshServices()
			}
		case "s": // start
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = startServiceDirect(name)
				m.refreshServices()
			}
		case "d": // disable
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = disableServiceDirect(name)
				m.refreshServices()
			}
		case "x": // stop
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = stopServiceDirect(name)
				m.refreshServices()
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.services)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// Run starts the TUI in alt-screen mode.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ── Helpers ─────────────────────────────────────────────────────────

// Direct user management functions (replaces HTTP API calls)
func addUserDirect(name, role string, perms []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Check if user already exists
	for _, u := range cfg.Users.Users {
		if u.Name == name {
			return fmt.Errorf("user already exists")
		}
	}
	// Add new user
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name:        name,
		Role:        role,
		Permissions: perms,
	})
	return config.SaveUsers(cfg)
}

func updateUserDirect(targetName, newName, newRole string, newPerms []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return fmt.Errorf("target name required")
	}
	// Find and update user
	updated := false
	for i := range cfg.Users.Users {
		if cfg.Users.Users[i].Name == targetName {
			// Handle rename
			newName = strings.TrimSpace(newName)
			if newName != "" && newName != targetName {
				// Check if new name already exists
				for j := range cfg.Users.Users {
					if j != i && cfg.Users.Users[j].Name == newName {
						return fmt.Errorf("user already exists")
					}
				}
				cfg.Users.Users[i].Name = newName
			}
			// Update role if provided
			if newRole != "" {
				cfg.Users.Users[i].Role = newRole
			}
			// Update permissions if provided
			if newPerms != nil {
				cfg.Users.Users[i].Permissions = newPerms
			}
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("user not found")
	}
	if err := config.SaveUsers(cfg); err != nil {
		return err
	}
	return nil
}

func deleteUserDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("user name required")
	}
	// Filter out the user
	var filtered []config.UserEntry
	found := false
	for _, u := range cfg.Users.Users {
		if u.Name != name {
			filtered = append(filtered, u)
		} else {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("user not found")
	}
	cfg.Users.Users = filtered
	return config.SaveUsers(cfg)
}

// Direct service management functions (replaces HTTP API calls)
func enableServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	// Find and enable service
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = shouldAutostartOnEnable(name, mods)
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			// Apply service action
			applyServiceAction(name, serviceActionEnableOnly, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func disableServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	// Find and disable service
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = false
			cfg.Services.Services[i].Autostart = false
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			// Apply service action
			applyServiceAction(name, serviceActionDisable, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func startServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	// Find and start service
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = true
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			// Apply service action
			applyServiceAction(name, serviceActionStart, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

func stopServiceDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("service name required")
	}
	mods := modules.LoadAll(".")
	// Find and stop service
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = false
			if err := config.SaveServices(cfg); err != nil {
				return err
			}
			// Apply service action
			applyServiceAction(name, serviceActionStop, mods)
			return state.RefreshCrateState(name)
		}
	}
	return fmt.Errorf("service not found")
}

// Service action types and helper functions
type serviceAction string

const (
	serviceActionEnableOnly serviceAction = "enable-only"
	serviceActionDisable    serviceAction = "disable"
	serviceActionStart      serviceAction = "start"
	serviceActionStop       serviceAction = "stop"
)

func applyServiceAction(name string, action serviceAction, mods map[string]modules.Module) {
	targets := []string{name}
	if mod, ok := mods[name]; ok {
		if units := modules.ResolveUnits(name, mod, true); len(units) > 0 {
			targets = units
		}
	}
	for _, target := range targets {
		switch action {
		case serviceActionEnableOnly:
			systemctlNoError("enable", target)
		case serviceActionDisable:
			systemctlNoError("stop", target)
			systemctlNoError("disable", target)
		case serviceActionStart:
			systemctlNoError("enable", target)
			systemctlNoError("start", target)
		case serviceActionStop:
			systemctlNoError("stop", target)
		}
	}
}

func shouldAutostartOnEnable(name string, mods map[string]modules.Module) bool {
	if mod, ok := mods[name]; ok {
		return mod.InstallMode() != "staged"
	}
	return true
}

func systemctlNoError(action, unit string) {
	if runtime.GOOS != "linux" {
		return
	}
	_ = exec.Command("systemctl", action, unit).Run()
}

func gatherServices() []ServiceInfo {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	actual := state.Probe(state.CollectServiceNames(cfg))
	actualByName := make(map[string]state.ServiceState, len(actual.Services))
	for _, svc := range actual.Services {
		actualByName[svc.Name] = svc
	}

	mods := modules.LoadAll(".")
	svcs := make([]ServiceInfo, 0, len(cfg.Services.Services))
	for _, desired := range cfg.Services.Services {
		service := ServiceInfo{
			Name:      desired.Name,
			Type:      modules.ResolveRuntime(desired.Runtime, mods[desired.Name], false),
			Status:    "unknown",
			Health:    "unknown",
			Desired:   desired.Enabled,
			Autostart: desired.Autostart,
			Enabled:   desired.Enabled,
			Ready:     !desired.Enabled,
		}
		if mod, ok := mods[desired.Name]; ok {
			service.Module = true
			service.DisplayName = mod.DisplayName()
			service.Category = mod.Metadata.Category
			service.Type = modules.ResolveRuntime(desired.Runtime, mod, true)
			for _, unit := range modules.ResolveUnits(desired.Name, mod, true) {
				unitState, ok := actualByName[unit]
				if !ok {
					unitState = state.ServiceState{Name: unit, Status: "unknown", Health: "unknown"}
				}
				service.Units = append(service.Units, ServiceUnit{
					Name:    unitState.Name,
					Active:  unitState.Active,
					Enabled: unitState.Enabled,
					Status:  unitState.Status,
					Health:  unitState.Health,
				})
			}
		}
		if service.DisplayName == "" {
			service.DisplayName = desired.Name
		}
		if len(service.Units) == 0 {
			if unitState, ok := actualByName[desired.Name]; ok {
				service.Units = append(service.Units, ServiceUnit{
					Name:    unitState.Name,
					Active:  unitState.Active,
					Enabled: unitState.Enabled,
					Status:  unitState.Status,
					Health:  unitState.Health,
				})
			}
		}
		service.Status, service.Health, service.Enabled, service.Ready = summarizeServiceState(desired.Enabled, service.Units)
		service.Active = service.Status == "active" || service.Status == "partial"
		service.LastError = readFallbackCrateLastError(desired.Name)
		svcs = append(svcs, service)
	}
	return svcs
}

func (m *model) refreshServices() {
	if info, svcs, platformInfo, _, _ := fetchStatusViaAPI(m.currentUser); info != nil {
		m.info = *info
		m.services = svcs
		m.platform = platformInfo
		m.controlPlaneOnline = true
		return
	}
	m.services = gatherServices()
	m.platform = readFallbackPlatformState()
	m.info = sysinfo.Gather()
	m.controlPlaneOnline = false
}

func summarizeServiceState(desired bool, units []ServiceUnit) (status string, health string, enabled bool, ready bool) {
	if len(units) == 0 {
		if desired {
			return "unknown", "unknown", true, false
		}
		return "inactive", "unknown", false, true
	}

	activeCount := 0
	enabledCount := 0
	failedCount := 0
	healthyCount := 0
	for _, unit := range units {
		if unit.Active {
			activeCount++
		}
		if unit.Enabled {
			enabledCount++
		}
		if unit.Status == "failed" {
			failedCount++
		}
		if unit.Health == "ok" {
			healthyCount++
		}
	}

	switch {
	case failedCount > 0:
		status = "failed"
	case activeCount == len(units):
		status = "active"
	case activeCount > 0:
		status = "partial"
	case desired:
		status = "inactive"
	default:
		status = "inactive"
	}

	switch {
	case healthyCount == len(units):
		health = "ok"
	case activeCount > 0:
		health = "degraded"
	default:
		health = "unknown"
	}

	enabled = enabledCount == len(units)
	ready = !desired || (activeCount == len(units) && healthyCount == len(units))
	return status, health, enabled, ready
}

func readFallbackCrateLastError(name string) string {
	path := platform.CratePath("services", name, "crate-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var stored struct {
		Crate struct {
			LastError string `json:"last_error"`
			Summary   string `json:"summary"`
		} `json:"crate"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return ""
	}
	if strings.TrimSpace(stored.Crate.LastError) != "" {
		return stored.Crate.LastError
	}
	return stored.Crate.Summary
}

func (m *model) refreshUsers() {
	if rows := fetchUsersViaAPI(m.currentUser); rows != nil {
		m.users = rows
		m.controlPlaneOnline = true
		return
	}
	m.users = fetchUsersFromConfig()
}

// fetchStatusViaAPI tries the local agent socket; returns nil on failure.
func fetchStatusViaAPI(user string) (*sysinfo.Info, []ServiceInfo, PlatformInfo, DiagnosticsInfo, string) {
	c := api.NewClient(user)
	raw, err := c.Status()
	if err != nil {
		return nil, nil, PlatformInfo{}, DiagnosticsInfo{}, user
	}
	var info sysinfo.Info
	if b, err := json.Marshal(raw["sysinfo"]); err == nil {
		_ = json.Unmarshal(b, &info)
	}
	var platformInfo PlatformInfo
	if b, err := json.Marshal(raw["platform"]); err == nil {
		_ = json.Unmarshal(b, &platformInfo)
	}
	var diagnostics DiagnosticsInfo
	if b, err := json.Marshal(raw["diagnostics"]); err == nil {
		_ = json.Unmarshal(b, &diagnostics)
	}
	if actorRaw, err := c.ActorDiagnostics(); err == nil {
		if ownership, ok := actorRaw["ownership"]; ok {
			if b, err := json.Marshal(ownership); err == nil {
				_ = json.Unmarshal(b, &diagnostics.Ownership)
			}
		}
	}
	svcResp, err := c.Services()
	if err != nil {
		return &info, nil, platformInfo, diagnostics, user
	}
	var svcs []ServiceInfo
	if services, ok := svcResp["services"]; ok {
		if b, err := json.Marshal(services); err == nil {
			_ = json.Unmarshal(b, &svcs)
		}
	}
	return &info, svcs, platformInfo, diagnostics, user
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

func fetchUsersViaAPI(user string) []userRow {
	c := api.NewClient(user)
	raw, err := c.Users()
	if err != nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var cfg struct {
		Users []struct {
			Name  string   `json:"name" yaml:"name"`
			Role  string   `json:"role" yaml:"role"`
			Perms []string `json:"permissions" yaml:"permissions"`
		} `json:"users"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil
	}
	var rows []userRow
	for _, u := range cfg.Users {
		rows = append(rows, userRow{Name: u.Name, Role: u.Role, Perms: u.Perms})
	}
	return rows
}

func fetchUsersFromConfig() []userRow {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	rows := make([]userRow, 0, len(cfg.Users.Users))
	for _, u := range cfg.Users.Users {
		rows = append(rows, userRow{Name: u.Name, Role: u.Role, Perms: u.Permissions})
	}
	return rows
}

func defaultUser() string {
	if rows := fetchUsersViaAPI(""); rows != nil && len(rows) > 0 {
		return rows[0].Name
	}
	return "crate"
}

func selectInitialUser() string {
	if rows := fetchUsersViaAPI(""); rows != nil && len(rows) > 0 {
		return rows[0].Name
	}
	if rows := fetchUsersFromConfig(); len(rows) > 0 {
		return rows[0].Name
	}
	return "crate"
}

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

func (m model) viewSetup() string {
	var b strings.Builder
	b.WriteString(headerBox.Render("First Boot Setup"))
	b.WriteString("\n\n")
	b.WriteString("Create the initial root admin for this CrateOS install.\n\n")
	b.WriteString(label.Render("Admin username: "))
	b.WriteString(value.Render(m.setupAdmin))
	b.WriteString("\n\n")
	b.WriteString(dim.Render("Press enter to bootstrap the platform."))
	b.WriteString("\n")
	b.WriteString(footer.Render("  type username · enter create admin · ctrl+c quit"))
	return b.String()
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
