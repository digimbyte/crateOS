package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func diagnosticsPanelItems() []string {
	return []string{"Summary", "Verification", "Ownership", "Config"}
}

func renderDiagnosticsVerificationPanel(m model) string {
	v := m.diagnostics.Verification
	var summary strings.Builder
	summary.WriteString(renderPanelLines(
		renderPanelKV("Status:", statusBadge(strings.TrimSpace(v.Status))),
		renderPanelKV("Agent Socket:", binaryBadge("sock", v.AgentSocket)),
		renderPanelKV("Admin Present:", binaryBadge("admin", v.AdminPresent)),
		renderPanelKV("Readiness:", compactOrDefault(v.Readiness, "unknown")),
	))
	if strings.TrimSpace(v.PlatformState) != "" {
		summary.WriteString(renderPanelKV("Platform State:", v.PlatformState) + "\n")
	}
	if strings.TrimSpace(v.StorageState) != "" {
		summary.WriteString(renderPanelKV("Storage State:", v.StorageState) + "\n")
	}
	if strings.TrimSpace(v.OwnershipState) != "" {
		summary.WriteString(renderPanelKV("Ownership State:", v.OwnershipState) + "\n")
	}
	if strings.TrimSpace(v.Summary) != "" {
		summary.WriteString(renderPanelKV("Summary:", v.Summary) + "\n")
	}
	items := []string{}
	for _, item := range v.Missing {
		items = append(items, "missing: "+item)
	}
	for _, item := range v.Warnings {
		items = append(items, "warn: "+item)
	}
	if len(items) == 0 {
		items = append(items, "verification surfaces look present")
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("MVP VERIFICATION", "INSTALL"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("miss", len(v.Missing)),
		selectorStat("warn", len(v.Warnings)),
		selectorStat("sock", boolToRail(v.AgentSocket)),
		selectorStat("adm", boolToRail(v.AdminPresent)),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("VERIFICATION SUMMARY", summary.String()))
	b.WriteString(renderSubsectionCard("CHECKS", renderBulletLines(items[:minInt(len(items), 8)])))
	b.WriteString(renderActionCard("OPERATOR PATH", renderPanelLines(
		dim.Render("This mirrors the installed-host MVP verification flow and highlights missing prerequisites before you rely on the control plane."),
		dim.Render("Use the host verifier for the final on-machine pass after install."),
	)))
	return renderActivePanel("MVP VERIFICATION", 72, b.String())
}

func compactOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func diagnosticsPanelTitle(cursor int) string {
	items := diagnosticsPanelItems()
	if cursor < 0 || cursor >= len(items) {
		return items[0]
	}
	return items[cursor]
}

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

func (m model) viewDiagnostics() string {
	hint := "  ↑↓ select section · [1-4] jump · [esc] back · [:] command"
	if m.statusSection == 2 {
		hint = "  ↑↓ browse workloads/details · [h/l] move workload · [1-4] jump section · [esc] back · [:] command"
	}
	return renderSplitView(m, "Diagnostics", renderDiagnosticsSelectionPanel(m), renderDiagnosticsFocusPanel(m), hint)
}

func renderDiagnosticsSelectionPanel(m model) string {
	lines := make([]string, 0, len(diagnosticsPanelItems()))
	for i, item := range diagnosticsPanelItems() {
		line := fmt.Sprintf("[%d] %s", i+1, diagnosticsSelectorLabel(m, item))
		lines = append(lines, renderSelectorLineWithGlyph(i == m.statusSection, diagnosticsRailGlyph(item), line))
	}
	return renderSelectionPanelWithMeta("DIAGNOSTICS", "SURFACES", 30, lines)
}

func renderDiagnosticsFocusPanel(m model) string {
	switch diagnosticsPanelTitle(m.statusSection) {
	case "Verification":
		return renderDiagnosticsVerificationPanel(m)
	case "Ownership":
		return renderDiagnosticsOwnershipPanel(m)
	case "Config":
		return renderDiagnosticsConfigPanel(m)
	default:
		return renderDiagnosticsSummaryPanel(m)
	}
}

func diagnosticsSelectorLabel(m model, item string) string {
	switch item {
	case "Summary":
		return renderBadgeRow("DIAG", selectorStat("cfg", m.diagnostics.Config.Tracked), selectorStat("plane", m.controlPlaneMode()))
	case "Verification":
		return renderBadgeRow("VERIFY", selectorStat("st", strings.TrimSpace(m.diagnostics.Verification.Status)), selectorStat("miss", len(m.diagnostics.Verification.Missing)))
	case "Ownership":
		return renderBadgeRow("ACTORS", selectorStat("wrk", m.diagnostics.Ownership.Managed), selectorStat("blk", m.diagnostics.Ownership.Blocked), selectorStat("act", m.diagnostics.Ownership.Active))
	case "Config":
		return renderBadgeRow("CONFIG", selectorStat("u", m.diagnostics.Config.Unmonitored), selectorStat("x", m.diagnostics.Config.ExternalEdits))
	default:
		return strings.ToUpper(item)
	}
}

func diagnosticsRailGlyph(item string) string {
	switch item {
	case "Summary":
		return "◆"
	case "Verification":
		return "◈"
	case "Ownership":
		return "◎"
	case "Config":
		return "▣"
	default:
		return "•"
	}
}

func renderDiagnosticsSummaryPanel(m model) string {
	var summary strings.Builder
	summary.WriteString(renderPanelLines(
		renderPanelKV("Control:", m.controlPlaneMode()),
		renderPanelKV("Verification:", statusBadge(strings.TrimSpace(m.diagnostics.Verification.Status))),
		renderCountKV("Active Actors:", m.diagnostics.Ownership.Active),
		renderPanelKV("Retired Actors:", statusCountText(m.diagnostics.Ownership.Retired)),
		renderCountKV("Tracked Configs:", m.diagnostics.Config.Tracked),
		renderPanelKV("Monitored:", statusCountText(m.diagnostics.Config.Monitored)),
		renderPanelKV("Unmonitored:", statusCountText(m.diagnostics.Config.Unmonitored)),
		renderPanelKV("External Edits:", statusCountText(m.diagnostics.Config.ExternalEdits)),
	))
	if strings.TrimSpace(m.diagnostics.Config.GeneratedAt) != "" {
		summary.WriteString(renderPanelKV("Ledger:", m.diagnostics.Config.GeneratedAt) + "\n")
	}
	if strings.TrimSpace(m.diagnostics.Verification.Summary) != "" {
		summary.WriteString(renderPanelKV("Verify:", m.diagnostics.Verification.Summary) + "\n")
	}
	if strings.TrimSpace(m.diagnostics.Ownership.GeneratedAt) != "" {
		summary.WriteString(renderPanelKV("Ownership:", m.diagnostics.Ownership.GeneratedAt) + "\n")
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("DIAGNOSTIC OVERVIEW", "CPANEL"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("vrf", strings.TrimSpace(m.diagnostics.Verification.Status)),
		selectorStat("act", m.diagnostics.Ownership.Active),
		selectorStat("ret", m.diagnostics.Ownership.Retired),
		selectorStat("cfg", m.diagnostics.Config.Tracked),
		selectorStat("mon", m.diagnostics.Config.Monitored),
		selectorStat("unm", m.diagnostics.Config.Unmonitored),
		selectorStat("ext", m.diagnostics.Config.ExternalEdits),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("DIAGNOSTIC SUMMARY", summary.String()))
	b.WriteString(renderActionCard("OPERATOR PATH", renderPanelLines(
		dim.Render("Use Verification to mirror the installed-host MVP verifier from inside the control plane."),
		dim.Render("Use Ownership to inspect active managed actor claims and retained tombstones without opening raw state files."),
		dim.Render("Use Config to inspect the tracked ledger for monitored versus external edits."),
		dim.Render("Keep diagnostics separate from runtime status so the control panel stays route-oriented."),
	)))
	return renderActivePanel("DIAGNOSTIC OVERVIEW", 72, b.String())
}

func renderDiagnosticsOwnershipPanel(m model) string {
	m.clampOwnershipCursor()
	var summary strings.Builder
	summary.WriteString(renderPanelLines(
		renderCountKV("Managed Workloads:", m.diagnostics.Ownership.Managed),
		renderCountKV("Provisioned:", m.diagnostics.Ownership.Provisioned),
		renderPanelKV("Pending:", statusCountText(m.diagnostics.Ownership.Pending)),
		renderPanelKV("Blocked:", statusCountText(m.diagnostics.Ownership.Blocked)),
		renderCountKV("Active Claims:", m.diagnostics.Ownership.Active),
		renderPanelKV("Retired Claims:", statusCountText(m.diagnostics.Ownership.Retired)),
		renderCountKV("Visible Workloads:", len(m.diagnostics.Ownership.Workloads)),
	))
	if strings.TrimSpace(m.diagnostics.Ownership.GeneratedAt) != "" {
		summary.WriteString(renderPanelKV("State:", m.diagnostics.Ownership.GeneratedAt) + "\n")
	}
	lines := []string{}
	var detail strings.Builder
	listStart := 0
	if len(m.diagnostics.Ownership.Workloads) > 8 {
		listStart = m.ownershipCursor - 3
		if listStart < 0 {
			listStart = 0
		}
		maxStart := len(m.diagnostics.Ownership.Workloads) - 8
		if listStart > maxStart {
			listStart = maxStart
		}
	}
	listEnd := minInt(len(m.diagnostics.Ownership.Workloads), listStart+8)
	for idx, workload := range m.diagnostics.Ownership.Workloads[listStart:listEnd] {
		line := compactOrDefault(strings.TrimSpace(workload.Crate), "unknown crate")
		if listStart+idx == m.ownershipCursor {
			line = "→ " + line
		}
		line += " · prov:" + compactOrDefault(strings.TrimSpace(workload.Provisioning), "pending")
		line += " · own:" + compactOrDefault(strings.TrimSpace(workload.OwnershipStatus), "unclaimed")
		if strings.TrimSpace(workload.ActorUser) != "" {
			line += " · " + workload.ActorUser
			if strings.TrimSpace(workload.ActorGroup) != "" {
				line += ":" + workload.ActorGroup
			}
		}
		if strings.TrimSpace(workload.ActorName) != "" {
			line += " · actor:" + workload.ActorName
		}
		if strings.TrimSpace(workload.ProvisioningError) != "" {
			line += " · issue:" + workload.ProvisioningError
		} else if strings.TrimSpace(workload.ProvisioningUpdatedAt) != "" {
			line += " · updated:" + workload.ProvisioningUpdatedAt
		}
		if len(workload.RecentEvents) > 0 {
			event := workload.RecentEvents[len(workload.RecentEvents)-1]
			line += " · last:" + compactOrDefault(strings.TrimSpace(event.Provisioning), "unknown")
			if strings.TrimSpace(event.Error) != "" {
				line += "(" + event.Error + ")"
			}
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "no managed actor lifecycle records recorded")
	} else {
		start := m.ownershipCursor
		if start < 0 {
			start = 0
		}
		if start >= len(m.diagnostics.Ownership.Workloads) {
			start = len(m.diagnostics.Ownership.Workloads) - 1
		}
		end := minInt(len(m.diagnostics.Ownership.Workloads), start+3)
		for idx, workload := range m.diagnostics.Ownership.Workloads[start:end] {
			label := compactOrDefault(strings.TrimSpace(workload.Crate), "unknown crate")
			if start+idx == m.ownershipCursor {
				label = "→ " + label
			}
			detail.WriteString(value.Render(label))
			detail.WriteString("\n")
			detail.WriteString(dim.Render("  provisioning: " + compactOrDefault(strings.TrimSpace(workload.Provisioning), "pending")))
			if strings.TrimSpace(workload.OwnershipStatus) != "" {
				detail.WriteString(dim.Render(" · ownership: " + workload.OwnershipStatus))
			}
			detail.WriteString("\n")
			if strings.TrimSpace(workload.ActorName) != "" || strings.TrimSpace(workload.ActorUser) != "" {
				detail.WriteString(dim.Render("  actor: " + valueOrFallback(strings.TrimSpace(workload.ActorName), "unassigned")))
				if strings.TrimSpace(workload.ActorUser) != "" {
					detail.WriteString(dim.Render(" · " + workload.ActorUser))
					if strings.TrimSpace(workload.ActorGroup) != "" {
						detail.WriteString(dim.Render(":" + workload.ActorGroup))
					}
				}
				detail.WriteString("\n")
			}
			if strings.TrimSpace(workload.ProvisioningStatePath) != "" {
				detail.WriteString(dim.Render("  state: " + workload.ProvisioningStatePath))
				detail.WriteString("\n")
			}
			if strings.TrimSpace(workload.LastSuccessAt) != "" || strings.TrimSpace(workload.LastFailureAt) != "" {
				detail.WriteString(dim.Render("  success: " + valueOrFallback(strings.TrimSpace(workload.LastSuccessAt), "never")))
				detail.WriteString(dim.Render(" · failure: " + valueOrFallback(strings.TrimSpace(workload.LastFailureAt), "never")))
				detail.WriteString("\n")
			}
			if len(workload.RecentEvents) > 0 {
				eventLines := make([]string, 0, minInt(len(workload.RecentEvents), 4))
				start := 0
				if len(workload.RecentEvents) > 4 {
					start = len(workload.RecentEvents) - 4
				}
				for _, event := range workload.RecentEvents[start:] {
					eventLine := compactOrDefault(strings.TrimSpace(event.At), "unknown time") + " · " + compactOrDefault(strings.TrimSpace(event.Provisioning), "unknown")
					if strings.TrimSpace(event.Error) != "" {
						eventLine += " · " + event.Error
					}
					eventLines = append(eventLines, eventLine)
				}
				detail.WriteString(dim.Render("  recent:"))
				detail.WriteString("\n")
				detail.WriteString(renderBulletLines(eventLines))
				detail.WriteString("\n")
			}
			detail.WriteString("\n")
		}
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("ACTOR OWNERSHIP", "DIAGNOSTICS"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("wrk", m.diagnostics.Ownership.Managed),
		selectorStat("ok", m.diagnostics.Ownership.Provisioned),
		selectorStat("blk", m.diagnostics.Ownership.Blocked),
		selectorStat("act", m.diagnostics.Ownership.Active),
		selectorStat("ret", m.diagnostics.Ownership.Retired),
		selectorStat("vis", len(m.diagnostics.Ownership.Workloads)),
		selectorStat("sel", fmt.Sprintf("%d/%d", currentOwnershipSelection(m), maxInt(len(m.diagnostics.Ownership.Workloads), 1))),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("LIFECYCLE SUMMARY", summary.String()))
	b.WriteString(renderSubsectionCard("WORKLOADS", renderBulletLines(lines)))
	if detail.Len() > 0 {
		b.WriteString(renderSubsectionCard("WORKLOAD DETAIL", strings.TrimRight(detail.String(), "\n")))
	}
	b.WriteString(renderActionCard("INTERPRETATION", renderPanelLines(
		dim.Render("provisioning comes from each crate's persisted actor-provisioning state artifact."),
		dim.Render("ownership comes from the global actor ownership summary and retained tombstones."),
		dim.Render("use diag actor <crate> or diag focus <crate>; up/down and h/l move the workload detail window."),
	)))
	return renderActivePanel("ACTOR OWNERSHIP", 72, b.String())
}

func currentOwnershipSelection(m model) int {
	if len(m.diagnostics.Ownership.Workloads) == 0 {
		return 1
	}
	return m.ownershipCursor + 1
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderDiagnosticsConfigPanel(m model) string {
	var summary strings.Builder
	summary.WriteString(renderPanelLines(
		renderCountKV("Tracked:", m.diagnostics.Config.Tracked),
		renderCountKV("Monitored:", m.diagnostics.Config.Monitored),
		renderPanelKV("Unmonitored:", statusCountText(m.diagnostics.Config.Unmonitored)),
		renderPanelKV("External:", statusCountText(m.diagnostics.Config.ExternalEdits)),
	))
	if strings.TrimSpace(m.diagnostics.Config.GeneratedAt) != "" {
		summary.WriteString(renderPanelKV("Ledger:", m.diagnostics.Config.GeneratedAt) + "\n")
	}
	lines := []string{}
	for _, file := range m.diagnostics.Config.Files {
		state := strings.TrimSpace(file.Monitoring)
		if state == "" {
			state = "monitored"
		}
		line := fmt.Sprintf("%s · %s", file.File, state)
		if strings.TrimSpace(file.LastWriter) != "" {
			line += " · writer:" + file.LastWriter
		}
		if !file.Exists {
			line += " · missing"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "no tracked config files")
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("CONFIG DIAGNOSTICS", "LEDGER"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("trk", m.diagnostics.Config.Tracked),
		selectorStat("mon", m.diagnostics.Config.Monitored),
		selectorStat("unm", m.diagnostics.Config.Unmonitored),
		selectorStat("ext", m.diagnostics.Config.ExternalEdits),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("LEDGER SUMMARY", summary.String()))
	b.WriteString(renderSubsectionCard("TRACKED FILES", renderBulletLines(lines[:minInt(len(lines), 8)])))
	b.WriteString(renderActionCard("INTERPRETATION", renderPanelLines(
		dim.Render("monitored = last change attributed to CrateOS-managed write flow"),
		dim.Render("unmonitored = file drift observed outside CrateOS-managed writes"),
	)))
	return renderActivePanel("CONFIG DIAGNOSTICS", 72, b.String())
}
