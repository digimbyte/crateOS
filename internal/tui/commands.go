package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

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
		m.setCommandWarn("ftp-complete: use agent reconciliation instead")
		return m, nil
	default:
		m.setCommandWarn("usage: system <refresh|dos2unix [config|services|all]|ftp-complete <path>>")
		return m, nil
	}
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
