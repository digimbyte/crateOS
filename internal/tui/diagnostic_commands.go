package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateDiagnostics(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.enterMenuView()
		case "up", "k":
			if m.statusSection == 2 && m.ownershipCursor > 0 {
				m.ownershipCursor--
			} else if m.statusSection > 0 {
				m.statusSection--
			}
		case "down", "j":
			if m.statusSection == 2 && m.ownershipCursor < len(m.diagnostics.Ownership.Workloads)-1 {
				m.ownershipCursor++
			} else if m.statusSection < len(diagnosticsPanelItems())-1 {
				m.statusSection++
			}
		case "left", "h":
			if m.statusSection == 2 && m.ownershipCursor > 0 {
				m.ownershipCursor--
			}
		case "right", "l":
			if m.statusSection == 2 && m.ownershipCursor < len(m.diagnostics.Ownership.Workloads)-1 {
				m.ownershipCursor++
			}
		case "1":
			m.statusSection = 0
		case "2":
			m.statusSection = 1
		case "3":
			m.statusSection = 2
		case "4":
			m.statusSection = 3
		}
	}
	return m, nil
}

func (m model) executeDiagnosticsCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	if m.currentView != viewDiagnostics {
		m.enterDiagnosticsView()
	}
	switch mod {
	case "list":
		if len(params) > 0 && (params[0] == "actors" || params[0] == "actor" || params[0] == "ownership") {
			if len(m.diagnostics.Ownership.Workloads) == 0 {
				m.setCommandInfo("actor diagnostics: none")
				return m, nil
			}
			names := make([]string, 0, len(m.diagnostics.Ownership.Workloads))
			for _, workload := range m.diagnostics.Ownership.Workloads {
				name := strings.TrimSpace(workload.Crate)
				if name == "" {
					name = strings.TrimSpace(workload.ActorName)
				}
				if name != "" {
					names = append(names, name)
				}
			}
			m.setCommandInfo("actor diagnostics: " + strings.Join(names, ", "))
			return m, nil
		}
		m.setCommandInfo("diagnostics sections: summary, verification, ownership, config")
	case "", "summary", "1":
		m.statusSection = 0
		m.setCommandOK("diagnostics section: summary")
	case "verification", "verify", "2":
		m.statusSection = 1
		m.setCommandOK("diagnostics section: verification")
	case "ownership", "actors", "actor", "3":
		m.statusSection = 2
		if len(params) > 0 {
			target := strings.Join(params, " ")
			if m.selectOwnershipWorkload(target) {
				workload := m.diagnostics.Ownership.Workloads[m.ownershipCursor]
				m.setCommandOK("actor diagnostics: " + compactOrDefault(strings.TrimSpace(workload.Crate), "unknown"))
				return m, nil
			}
			m.setCommandWarn("unknown actor workload: " + target)
			return m, nil
		}
		m.setCommandOK("diagnostics section: ownership")
	case "config", "cfg", "4":
		m.statusSection = 3
		m.setCommandOK("diagnostics section: config")
	case "next":
		if m.statusSection == 2 && m.ownershipCursor < len(m.diagnostics.Ownership.Workloads)-1 {
			m.ownershipCursor++
			workload := m.diagnostics.Ownership.Workloads[m.ownershipCursor]
			m.setCommandInfo("actor diagnostics: " + compactOrDefault(strings.TrimSpace(workload.Crate), "unknown"))
			return m, nil
		}
		if m.statusSection < len(diagnosticsPanelItems())-1 {
			m.statusSection++
		}
		m.setCommandInfo("diagnostics section advanced")
	case "prev":
		if m.statusSection == 2 && m.ownershipCursor > 0 {
			m.ownershipCursor--
			workload := m.diagnostics.Ownership.Workloads[m.ownershipCursor]
			m.setCommandInfo("actor diagnostics: " + compactOrDefault(strings.TrimSpace(workload.Crate), "unknown"))
			return m, nil
		}
		if m.statusSection > 0 {
			m.statusSection--
		}
		m.setCommandInfo("diagnostics section reversed")
	case "select":
		if len(params) == 0 {
			m.setCommandWarn("usage: diag select <summary|verification|ownership|config|1|2|3|4>")
			return m, nil
		}
		return m.executeDiagnosticsCommand(strings.ToLower(params[0]), params[1:])
	case "focus", "show", "inspect":
		m.statusSection = 2
		if len(params) == 0 {
			m.setCommandWarn("usage: diag focus <crate|actor|user|id>")
			return m, nil
		}
		target := strings.Join(params, " ")
		if !m.selectOwnershipWorkload(target) {
			m.setCommandWarn("unknown actor workload: " + target)
			return m, nil
		}
		workload := m.diagnostics.Ownership.Workloads[m.ownershipCursor]
		m.setCommandOK("actor diagnostics: " + compactOrDefault(strings.TrimSpace(workload.Crate), "unknown"))
	default:
		m.setCommandWarn("usage: diag <summary|verification|ownership|config|actor [target]|focus <target>|next|prev>")
	}
	return m, nil
}
