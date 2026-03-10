package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

func (m model) viewServices() string {
	return renderSplitView(m, "Services", renderServiceSelectionPanel(m), renderServiceFocusPanel(m), "  ↑↓ select crate · [e] enable · [s] start · [x] stop · [d] disable · [esc] back · [:] command · user: "+m.currentUser+" · plane: "+m.controlPlaneMode())
}

func renderServiceSelectionPanel(m model) string {
	lines := []string{}
	if len(m.services) == 0 {
		lines = append(lines, dim.Render("  No services registered."))
		return renderSelectionPanel("CRATE CONTROL", 34, lines)
	}
	for i, s := range m.services {
		nameText := s.DisplayName
		if nameText == "" {
			nameText = s.Name
		}
		line := fmt.Sprintf(
			"%s %s  %s",
			statusIndicator(s.Status),
			compactLabel(nameText, 12),
			renderBadgeRow(selectorStat("s", s.Status), selectorStat("r", boolToRail(s.Ready))),
		)
		lines = append(lines, renderSelectorLineWithGlyph(i == m.cursor, serviceRailGlyph(s), line))
	}
	return renderSelectionPanelWithMeta("CRATE CONTROL", "CRATES", 34, lines)
}

func renderServiceFocusPanel(m model) string {
	if len(m.services) == 0 {
		return renderActivePanel("ACTIVE CRATE", 68, dim.Render("No crate selected."))
	}
	s := m.services[m.currentServiceIndex()]
	nameText := s.DisplayName
	if nameText == "" {
		nameText = s.Name
	}
	var header strings.Builder
	header.WriteString(renderPanelTitleBar(strings.ToUpper(nameText), "CRATE"))
	header.WriteString("\n")
	header.WriteString(renderStatStrip(
		statusBadge(s.Status),
		healthBadge(s.Health),
		typeBadge(s.Type),
	))
	header.WriteString("\n")
	var lifecycle strings.Builder
	lifecycle.WriteString(renderBadgeRow(
		lifecycleDesiredBadge(s),
		lifecycleAutostartBadge(s),
		lifecycleRuntimeBadge(s),
		readyBadge(s.Ready),
	))
	lifecycle.WriteString("\n")
	lifecycle.WriteString(dim.Render("intent: " + lifecycleIntentText(s)))
	lifecycle.WriteString("\n")
	lifecycle.WriteString(dim.Render("units: " + lifecycleUnitCounts(s)))
	lifecycle.WriteString("\n")
	if s.DisplayName != "" && s.DisplayName != s.Name {
		lifecycle.WriteString(dim.Render("id: " + s.Name))
		lifecycle.WriteString("\n")
	}
	if s.Module {
		lifecycle.WriteString(dim.Render("module category: " + s.Category))
		lifecycle.WriteString("\n")
	}
	if s.Summary != "" {
		lifecycle.WriteString(dim.Render("summary: " + s.Summary))
		lifecycle.WriteString("\n")
	}
	if s.LastAction != "" {
		lifecycle.WriteString(dim.Render("last action: " + s.LastAction))
		if s.LastActionAt != "" {
			lifecycle.WriteString(dim.Render(" @ " + s.LastActionAt))
		}
		lifecycle.WriteString("\n")
	}
	if s.LastError != "" {
		lifecycle.WriteString(danger.Render("issue: " + s.LastError))
		lifecycle.WriteString("\n")
	}
	if s.SuggestedRepair != "" {
		lifecycle.WriteString(warn.Render("repair: " + s.SuggestedRepair))
		lifecycle.WriteString("\n")
	}
	if (s.Status == "failed" || s.Status == "partial" || s.Health == "degraded") && (s.LastGoodStatus != "" || s.LastGoodSummary != "") {
		lastGood := []string{}
		if s.LastGoodStatus != "" {
			lastGood = append(lastGood, "state:"+s.LastGoodStatus)
		}
		if s.LastGoodHealth != "" {
			lastGood = append(lastGood, "health:"+s.LastGoodHealth)
		}
		if s.LastGoodAt != "" {
			lastGood = append(lastGood, "at:"+s.LastGoodAt)
		}
		lifecycle.WriteString(ok.Render("last-good: " + strings.Join(lastGood, "  ")))
		lifecycle.WriteString("\n")
		if s.LastGoodSummary != "" {
			lifecycle.WriteString(dim.Render("last-good summary: " + s.LastGoodSummary))
			lifecycle.WriteString("\n")
		}
	}
	if s.Module && !s.PackagesInstalled && len(s.MissingPackages) > 0 {
		lifecycle.WriteString(danger.Render("missing packages: " + strings.Join(s.MissingPackages, ", ")))
		lifecycle.WriteString("\n")
	}
	if s.Stateful {
		if strings.TrimSpace(s.StorageSummary) != "" {
			lifecycle.WriteString(dim.Render("storage: " + s.StorageSummary))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.DataPath) != "" {
			lifecycle.WriteString(dim.Render("data path: " + s.DataPath))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.NativeDataPath) != "" {
			lifecycle.WriteString(dim.Render("native data: " + s.NativeDataPath))
			lifecycle.WriteString("\n")
		}
	}
	if strings.TrimSpace(s.ActorName) != "" || strings.TrimSpace(s.ExecutionMode) != "" {
		lifecycle.WriteString(dim.Render("actor: " + valueOrFallback(strings.TrimSpace(s.ActorName), "unassigned")))
		if strings.TrimSpace(s.ActorType) != "" {
			lifecycle.WriteString(dim.Render(" · type: " + s.ActorType))
		}
		lifecycle.WriteString("\n")
		if strings.TrimSpace(s.ActorID) != "" {
			lifecycle.WriteString(dim.Render("actor id: " + s.ActorID))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorUser) != "" || strings.TrimSpace(s.ActorGroup) != "" {
			lifecycle.WriteString(dim.Render("actor runtime account: " + valueOrFallback(strings.TrimSpace(s.ActorUser), "pending")))
			lifecycle.WriteString(dim.Render(":" + valueOrFallback(strings.TrimSpace(s.ActorGroup), "pending")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorHome) != "" {
			lifecycle.WriteString(dim.Render("actor home: " + s.ActorHome))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ActorProvisioning) != "" {
			lifecycle.WriteString(dim.Render("actor provisioning: " + s.ActorProvisioning))
			if strings.TrimSpace(s.ActorProvisioningError) != "" {
				lifecycle.WriteString(danger.Render(" · " + s.ActorProvisioningError))
			}
			lifecycle.WriteString("\n")
			if strings.TrimSpace(s.ActorProvisioningUpdatedAt) != "" {
				lifecycle.WriteString(dim.Render("actor provisioning updated: " + s.ActorProvisioningUpdatedAt))
				lifecycle.WriteString("\n")
			}
			if strings.TrimSpace(s.ActorProvisioningStatePath) != "" {
				lifecycle.WriteString(dim.Render("actor provisioning state: " + s.ActorProvisioningStatePath))
				lifecycle.WriteString("\n")
			}
		}
		if strings.TrimSpace(s.ActorOwnershipStatus) != "" {
			lifecycle.WriteString(dim.Render("actor ownership: " + s.ActorOwnershipStatus))
			if strings.TrimSpace(s.ActorOwnershipUpdatedAt) != "" {
				lifecycle.WriteString(dim.Render(" @ " + s.ActorOwnershipUpdatedAt))
			}
			lifecycle.WriteString("\n")
			if strings.TrimSpace(s.ActorOwnershipRetiredAt) != "" {
				lifecycle.WriteString(dim.Render("actor ownership retired: " + s.ActorOwnershipRetiredAt))
				lifecycle.WriteString("\n")
			}
		}
		if strings.TrimSpace(s.ActorRuntimeDir) != "" || strings.TrimSpace(s.ActorStateDir) != "" {
			lifecycle.WriteString(dim.Render("actor dirs: run " + valueOrFallback(strings.TrimSpace(s.ActorRuntimeDir), "pending")))
			lifecycle.WriteString(dim.Render(" · state " + valueOrFallback(strings.TrimSpace(s.ActorStateDir), "pending")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionSummary) != "" {
			lifecycle.WriteString(dim.Render("execution: " + s.ExecutionSummary))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.DeploySource) != "" {
			lifecycle.WriteString(dim.Render("deploy: " + s.DeploySource))
			if strings.TrimSpace(s.UploadPath) != "" {
				lifecycle.WriteString(dim.Render(" · intake: " + s.UploadPath))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.WorkingDir) != "" {
			lifecycle.WriteString(dim.Render("working dir: " + s.WorkingDir))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.InstallCommand) != "" {
			lifecycle.WriteString(dim.Render("install: " + s.InstallCommand))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionAdapter) != "" || strings.TrimSpace(s.ExecutionStatus) != "" {
			lifecycle.WriteString(dim.Render("runtime object: " + valueOrFallback(strings.TrimSpace(s.ExecutionAdapter), "systemd")))
			if strings.TrimSpace(s.ExecutionStatus) != "" {
				lifecycle.WriteString(dim.Render(" · " + s.ExecutionStatus))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.PrimaryUnit) != "" {
			lifecycle.WriteString(dim.Render("primary unit: " + s.PrimaryUnit))
			if strings.TrimSpace(s.CompanionUnit) != "" {
				lifecycle.WriteString(dim.Render(" · companion: " + s.CompanionUnit))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.PrimaryUnitPath) != "" {
			lifecycle.WriteString(dim.Render("unit file: " + s.PrimaryUnitPath))
			if strings.TrimSpace(s.CompanionUnitPath) != "" {
				lifecycle.WriteString(dim.Render(" · timer file: " + s.CompanionUnitPath))
			}
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ExecutionStatePath) != "" {
			lifecycle.WriteString(dim.Render("execution state: " + s.ExecutionStatePath))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.Schedule) != "" {
			lifecycle.WriteString(dim.Render("schedule: " + s.Schedule))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.Timeout) != "" || strings.TrimSpace(s.StopTimeout) != "" {
			lifecycle.WriteString(dim.Render("timeouts: run " + valueOrFallback(strings.TrimSpace(s.Timeout), "0") + " · stop " + valueOrFallback(strings.TrimSpace(s.StopTimeout), "30s")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.OnTimeout) != "" {
			lifecycle.WriteString(dim.Render("timeout action: " + s.OnTimeout + " via " + valueOrFallback(strings.TrimSpace(s.KillSignal), "SIGTERM")))
			lifecycle.WriteString("\n")
		}
		if strings.TrimSpace(s.ConcurrencyPolicy) != "" {
			lifecycle.WriteString(dim.Render("overlap: " + s.ConcurrencyPolicy))
			lifecycle.WriteString("\n")
		}
	}
	var unitGrid strings.Builder
	if len(s.Units) == 0 {
		unitGrid.WriteString(dim.Render("No unit detail available."))
		unitGrid.WriteString("\n")
	} else {
		for _, unit := range s.Units {
			line := fmt.Sprintf("%s %s  %s  %s  %s  %s",
				statusIndicator(unit.Status),
				unit.Name,
				statusBadge(unit.Status),
				healthBadge(unit.Health),
				binaryBadge("enabled", unit.Enabled),
				binaryBadge("active", unit.Active),
			)
			unitGrid.WriteString(dim.Render(line))
			unitGrid.WriteString("\n")
		}
	}
	legend := fmt.Sprintf("%s active   %s inactive   %s failed   %s unknown",
		ok.Render("●"),
		dim.Render("○"),
		danger.Render("✖"),
		warn.Render("?"),
	)
	failed, partial, staged, healthy := menuServiceCounts(m.services)
	var posture strings.Builder
	posture.WriteString(renderBadgeRow(
		selectorStat("trk", len(m.services)),
		selectorStat("ok", healthy),
		selectorStat("bad", failed+partial),
		selectorStat("stg", staged),
	))
	posture.WriteString("\n")
	posture.WriteString(dim.Render("This crate sits inside the wider desired-state fleet; use Status for aggregate diagnostics when local symptoms stack."))
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(renderSummaryCard("SYSTEM POSTURE", posture.String()))
	b.WriteString(renderSummaryCard("LIFECYCLE", lifecycle.String()))
	b.WriteString(renderSubsectionCard("UNIT GRID", unitGrid.String()))
	b.WriteString(renderActionCard("LEGEND", legend))
	return renderActivePanel("ACTIVE CRATE", 68, b.String())
}

func (m model) currentServiceIndex() int {
	if len(m.services) == 0 {
		return 0
	}
	cursor := m.cursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(m.services) {
		return len(m.services) - 1
	}
	return cursor
}

func statusIndicator(status string) string {
	switch status {
	case "active", "running":
		return ok.Render("●")
	case "inactive", "dead":
		return dim.Render("○")
	case "failed":
		return danger.Render("✖")
	case "partial", "activating":
		return warn.Render("◐")
	case "staged":
		return dim.Render("◌")
	default:
		return warn.Render("?")
	}
}

func valueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func serviceRailGlyph(s ServiceInfo) string {
	switch s.Status {
	case "failed":
		return "✖"
	case "partial":
		return "◐"
	case "active", "running":
		return "●"
	case "staged":
		return "◌"
	default:
		return "○"
	}
}

func lifecycleDesiredBadge(s ServiceInfo) string {
	if s.Desired {
		return ok.Render("desired:on")
	}
	return dim.Render("desired:off")
}

func lifecycleAutostartBadge(s ServiceInfo) string {
	if !s.Desired {
		return dim.Render("autostart:n/a")
	}
	if s.Autostart {
		return ok.Render("autostart:on")
	}
	return warn.Render("autostart:off")
}

func lifecycleRuntimeBadge(s ServiceInfo) string {
	switch {
	case !s.Desired:
		return dim.Render("runtime:disabled")
	case s.Status == "failed":
		return danger.Render("runtime:failed")
	case s.Status == "partial":
		return warn.Render("runtime:partial")
	case s.Active:
		return ok.Render("runtime:running")
	case s.Status == "staged":
		return dim.Render("runtime:waiting")
	case s.Status == "unknown":
		return warn.Render("runtime:unknown")
	default:
		return warn.Render("runtime:stopped")
	}
}

func lifecycleIntentText(s ServiceInfo) string {
	switch {
	case s.ExecutionMode == "job" && s.Schedule != "":
		return "crate is modeled as a scheduled job and should run on the declared cadence"
	case !s.Desired:
		return "crate is disabled and should not be running"
	case s.Status == "failed":
		return "crate is enabled but one or more units have failed"
	case s.Status == "partial":
		return "crate is enabled but only part of the unit set is running cleanly"
	case s.Status == "staged":
		return "crate is enabled but waiting for an explicit start"
	case s.Autostart && s.Active:
		return "crate is enabled and should be kept running automatically"
	case s.Autostart && !s.Active:
		return "crate is enabled for automatic runtime but is not currently running"
	case !s.Autostart && s.Active:
		return "crate is enabled and currently running without automatic restart intent"
	default:
		return "crate is enabled but intentionally stopped until started again"
	}
}

func statusBadge(status string) string {
	text := "state:" + status
	switch status {
	case "active", "running", "ready":
		return ok.Render(text)
	case "inactive", "dead", "disabled", "staged":
		return dim.Render(text)
	case "failed":
		return danger.Render(text)
	case "partial", "activating", "unknown":
		return warn.Render(text)
	default:
		return warn.Render(text)
	}
}

func healthBadge(health string) string {
	if strings.TrimSpace(health) == "" {
		return dim.Render("health:n/a")
	}
	text := "health:" + health
	switch health {
	case "ok", "healthy", "ready":
		return ok.Render(text)
	case "warn", "warning", "degraded":
		return warn.Render(text)
	case "fail", "failed", "error", "critical":
		return danger.Render(text)
	default:
		return dim.Render(text)
	}
}

func typeBadge(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return dim.Render("type:unknown")
	}
	return dim.Render("type:" + kind)
}
func lifecycleUnitCounts(s ServiceInfo) string {
	if len(s.Units) == 0 {
		return "no unit detail available"
	}
	running := 0
	enabled := 0
	failed := 0
	for _, unit := range s.Units {
		if unit.Active {
			running++
		}
		if unit.Enabled {
			enabled++
		}
		if unit.Status == "failed" {
			failed++
		}
	}
	return fmt.Sprintf("%d/%d running · %d/%d enabled · %d failed", running, len(s.Units), enabled, len(s.Units), failed)
}

func readyBadge(ready bool) string {
	if ready {
		return ok.Render("ready:on")
	}
	return danger.Render("ready:off")
}

func boolToRail(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
