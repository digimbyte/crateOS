package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab":
			m.setupField = (m.setupField + 1) % 6
			return m, nil
		case "shift+tab":
			if m.setupField == 0 {
				m.setupField = 5
			} else {
				m.setupField--
			}
			return m, nil
		case "r":
			m.refreshPrimerStatusMessage()
			return m, nil
		case "backspace":
			if m.setupField == 0 {
				if len(m.setupHostname) > 0 {
					m.setupHostname = m.setupHostname[:len(m.setupHostname)-1]
				}
			} else if m.setupField == 1 && len(m.setupAdmin) > 0 {
				m.setupAdmin = m.setupAdmin[:len(m.setupAdmin)-1]
			}
		case "up", "k":
			if m.setupField == 0 {
				m.setupField = 5
			} else {
				m.setupField--
			}
			return m, nil
		case "down", "j":
			m.setupField = (m.setupField + 1) % 6
			return m, nil
		case "enter":
			switch m.setupField {
			case 0, 2:
				if !m.savePrimerIdentity() {
					m.setCommandError("primer save failed: hostname")
					return m, nil
				}
				m.refreshPrimerState()
				if m.primerRequired {
					m.currentView = viewSetup
					m.setCommandOK("machine identity saved")
				} else {
					m.setCommandOK("primer complete: console unlocked")
				}
				return m, nil
			case 1, 3:
				name := strings.TrimSpace(m.setupAdmin)
				if name == "" {
					m.setCommandWarn("primer admin username is required")
					return m, nil
				}
				if !m.bootstrapAdmin(name) {
					m.refreshPrimerState()
					if m.primerRequired {
						m.setCommandError("bootstrap failed: " + name)
					} else {
						m.setCommandOK("primer state refreshed")
					}
					return m, nil
				}
				m.currentView = viewSetup
				m.setCommandOK("initial admin configured: " + name)
				return m, nil
			case 4:
				if !m.applyPrimerTakeover() {
					m.currentView = viewSetup
					m.setCommandError("primer takeover repair failed")
					return m, nil
				}
				m.refreshPrimerState()
				if m.primerRequired {
					m.currentView = viewSetup
					m.setCommandOK("local install contract repaired")
				} else {
					m.setCommandOK("primer complete: console unlocked")
				}
				return m, nil
			case 5:
				if !m.provisionPrimerUsers() {
					m.currentView = viewSetup
					m.setCommandError("primer user provisioning failed")
					return m, nil
				}
				m.refreshUsers()
				m.refreshPrimerState()
				if m.primerRequired {
					m.currentView = viewSetup
					m.setCommandOK("operator account provisioning applied")
				} else {
					m.setCommandOK("primer complete: console unlocked")
				}
				return m, nil
			}
		default:
			if len(msg.String()) == 1 {
				if m.setupField == 0 {
					m.setupHostname += msg.String()
				} else if m.setupField == 1 {
					m.setupAdmin += msg.String()
				}
			}
		}
	}
	return m, nil
}

func renderPrimerField(labelText, current string, active bool) string {
	row := label.Render(labelText+": ") + value.Render(current)
	if active {
		return selectorActive.Render("▌ " + row)
	}
	return selectorIdle.Render("│ " + row)
}

func renderPrimerAction(labelText, details string, active bool) string {
	row := label.Render(labelText)
	if strings.TrimSpace(details) != "" {
		row += "  " + dim.Render(details)
	}
	if active {
		return selectorActive.Render("▌ " + row)
	}
	return selectorIdle.Render("│ " + row)
}

func (m model) viewSetup() string {
	var b strings.Builder
	b.WriteString(headerBox.Render("CrateOS Primer"))
	b.WriteString("\n\n")
	b.WriteString("This console session is locked inside the first-use installer/primer until CrateOS is fully initialized.\n\n")
	b.WriteString(section.Render("Primer checks"))
	b.WriteString("\n")
	if len(m.primerChecks) == 0 {
		b.WriteString(dim.Render("  No primer state available."))
		b.WriteString("\n")
	} else {
		for _, check := range m.primerChecks {
			status := ok.Render("[ok]")
			if !check.OK {
				status = danger.Render("[block]")
			}
			b.WriteString(fmt.Sprintf("%s %s", status, check.Label))
			if strings.TrimSpace(check.Details) != "" {
				b.WriteString("  ")
				if check.OK {
					b.WriteString(dim.Render(check.Details))
				} else {
					b.WriteString(warn.Render(check.Details))
				}
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(section.Render("Initial admin"))
	b.WriteString("\n")
	b.WriteString("Use the primer as a staged local installer: save identity, create the first admin, repair takeover/runtime contract, provision the operator account, then refresh readiness. The normal control menu stays locked until every blocking check clears.\n\n")
	b.WriteString(renderPrimerField("Hostname", m.setupHostname, m.setupField == 0))
	b.WriteString("\n")
	b.WriteString(renderPrimerField("Admin username", m.setupAdmin, m.setupField == 1))
	b.WriteString("\n\n")
	b.WriteString(section.Render("Primer actions"))
	b.WriteString("\n")
	b.WriteString(renderPrimerAction("Save machine identity", "writes hostname into crateos.yaml", m.setupField == 2))
	b.WriteString("\n")
	b.WriteString(renderPrimerAction("Configure initial admin", "creates the first local admin entry in users.yaml", m.setupField == 3))
	b.WriteString("\n")
	b.WriteString(renderPrimerAction("Repair local install contract", "ensures login shell, tty1 takeover, ssh landing, and identity files", m.setupField == 4))
	b.WriteString("\n")
	b.WriteString(renderPrimerAction("Provision local operator account", "creates the system account with crateos-login-shell", m.setupField == 5))
	b.WriteString("\n\n")
	b.WriteString(dim.Render("Use tab or ↑↓ to move between fields and actions. Press enter on the highlighted item to run that step. Press r or use ':system refresh' to re-evaluate primer checks."))
	b.WriteString("\n")
	b.WriteString(footer.Render("  primer locked · tab move · enter run step · r refresh · [:] command"))
	return b.String()
}
