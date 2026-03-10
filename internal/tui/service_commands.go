package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateServices(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "e":
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = enableServiceDirect(name)
				m.refreshServices()
			}
		case "s":
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = startServiceDirect(name)
				m.refreshServices()
			}
		case "d":
			if m.cursor < len(m.services) {
				name := m.services[m.cursor].Name
				_ = disableServiceDirect(name)
				m.refreshServices()
			}
		case "x":
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
