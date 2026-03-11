package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

func (m model) executeSingleCommand(raw string) (tea.Model, tea.Cmd) {
	args := tokenizeCommandInput(raw)
	if len(args) == 0 {
		m.setCommandWarn("empty command")
		return m, nil
	}
	cmd := strings.ToLower(args[0])
	if m.currentView == viewSetup {
		switch cmd {
		case "help", "?", "bootstrap", "system":
		default:
			m.setCommandWarn("primer completion required before control-plane commands")
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
		if m.primerRequired {
			m.currentView = viewSetup
			m.setCommandWarn("primer completion required before leaving setup")
			return m, nil
		}
		m.enterMenuView()
		m.setCommandOK("route: menu")
	case "quit", "exit":
		m.setCommandWarn("session exit is disabled; CrateOS is the active interface")
		return m, nil
	default:
		if action := normalizeServiceAction(mod); action != "" {
			target := cmd
			return m.executeServiceLifecycleCommand(action, target)
		}
		m.setCommandError("unknown command: " + raw + " (type: help)")
	}
	return m, nil
}
