package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
