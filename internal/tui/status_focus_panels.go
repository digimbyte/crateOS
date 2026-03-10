package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

func crateIssueLine(s ServiceInfo) string {
	name := s.DisplayName
	if strings.TrimSpace(name) == "" {
		name = s.Name
	}
	switch {
	case s.LastError != "":
		return name + ": " + s.LastError
	case s.Status == "failed" && s.Summary != "":
		return name + ": " + s.Summary
	case s.Status == "partial" && s.Summary != "":
		return name + ": " + s.Summary
	case !s.Ready && s.Status == "staged":
		return name + ": waiting for explicit start"
	case !s.Ready && s.Health != "" && s.Health != "ok":
		return name + ": health is " + s.Health
	default:
		return ""
	}
}

func platformIssueLine(adapter PlatformAdapter) string {
	name := adapter.DisplayName
	if strings.TrimSpace(name) == "" {
		name = adapter.Name
	}
	switch {
	case adapter.LastError != "":
		return name + ": " + adapter.LastError
	case adapter.Status == "failed" && adapter.Summary != "":
		return name + ": " + adapter.Summary
	default:
		return ""
	}
}

func renderPlatformAdapterStatus(adapter PlatformAdapter) string {
	var b strings.Builder
	name := adapter.DisplayName
	if strings.TrimSpace(name) == "" {
		name = adapter.Name
	}
	b.WriteString(value.Render(name))
	b.WriteString("\n")
	b.WriteString(renderBadgeRow(
		statusBadge(adapter.Status),
		healthBadge(adapter.Health),
		binaryBadge("enabled", adapter.Enabled),
	))
	b.WriteString("\n")
	if strings.TrimSpace(adapter.Summary) != "" {
		b.WriteString(dim.Render("  summary: " + adapter.Summary))
		b.WriteString("\n")
	}
	if strings.TrimSpace(adapter.LastError) != "" {
		b.WriteString(danger.Render("  issue: " + adapter.LastError))
		b.WriteString("\n")
	}
	if strings.TrimSpace(adapter.Validation) != "" {
		line := "  validation: " + adapter.Validation
		if strings.TrimSpace(adapter.ValidationErr) != "" {
			line += " (" + adapter.ValidationErr + ")"
		}
		b.WriteString(dim.Render(line))
		b.WriteString("\n")
	}
	if strings.TrimSpace(adapter.Apply) != "" {
		line := "  apply: " + adapter.Apply
		if strings.TrimSpace(adapter.ApplyErr) != "" {
			line += " (" + adapter.ApplyErr + ")"
		}
		b.WriteString(dim.Render(line))
		b.WriteString("\n")
	}
	if len(adapter.NativeTargets) > 0 {
		b.WriteString(dim.Render("  native: " + strings.Join(adapter.NativeTargets[:minInt(len(adapter.NativeTargets), 2)], ", ")))
		b.WriteString("\n")
	}
	if len(adapter.RenderedPaths) > 0 {
		b.WriteString(dim.Render("  rendered: " + strings.Join(adapter.RenderedPaths[:minInt(len(adapter.RenderedPaths), 2)], ", ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderSystemStatusPanel(m model) string {
	var summary strings.Builder
	summary.WriteString(renderPanelLines(
		renderPanelKV("Hostname:", m.info.Hostname),
		renderPanelKV("Version:", platform.Version),
		renderPanelKV("Platform:", fmt.Sprintf("%s/%s", m.info.OS, m.info.Arch)),
		renderPanelKV("Control:", m.controlPlaneMode()),
		renderPanelKV("Time:", m.info.Time.Format(time.RFC3339)),
		renderPanelKV("CPUs:", fmt.Sprintf("%d", m.info.CPUs)),
		renderPanelKV("Go:", m.info.GoVersion),
	))
	root := platform.CrateRoot
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		summary.WriteString(renderPanelKV("Root:", ok.Render(root+" [OK]")) + "\n")
	} else {
		summary.WriteString(renderPanelKV("Root:", warn.Render(root+" [NOT FOUND]")) + "\n")
	}
	marker := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(marker); err == nil {
		summary.WriteString(renderPanelKV("Installed:", ok.Render("yes")) + "\n")
	} else {
		summary.WriteString(renderPanelKV("Installed:", dim.Render("no")) + "\n")
	}
	missing := 0
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		if _, err := os.Stat(p); err != nil {
			missing++
		}
	}
	if missing == 0 {
		summary.WriteString(renderPanelKV("Directories:", ok.Render(fmt.Sprintf("all %d present", len(platform.RequiredDirs)))) + "\n")
	} else {
		summary.WriteString(renderPanelKV("Directories:", danger.Render(fmt.Sprintf("%d/%d missing", missing, len(platform.RequiredDirs)))) + "\n")
	}
	if report, ok := readReadinessReport(); ok {
		summary.WriteString(renderPanelKV("Readiness:", report.statusText()) + "\n")
		if strings.TrimSpace(report.Summary) != "" {
			summary.WriteString(renderPanelKV("Reason:", report.Summary) + "\n")
		}
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("SYSTEM CORE", "DIAGNOSTICS"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("cpu", m.info.CPUs),
		selectorStat("os", m.info.OS),
		selectorStat("arch", m.info.Arch),
		selectorStat("dirs", fmt.Sprintf("%d/%d", len(platform.RequiredDirs)-missing, len(platform.RequiredDirs))),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("SYSTEM SUMMARY", summary.String()))
	b.WriteString(renderActionCard("GUIDANCE", renderPanelLines(
		dim.Render("Use the selector on the left to switch between core system, crate status, and platform adapters."),
		dim.Render("This module tracks host posture and install integrity rather than crate-level runtime detail."),
		dim.Render("Use Diagnostics from the control menu for ledgers and drift surfaces."),
		dim.Render("Fallback-local means read surfaces still render, but lifecycle and user writes must wait for the agent to come online."),
	)))
	return renderActivePanel("SYSTEM CORE", 72, b.String())
}

func renderServiceStatusPanel(m model) string {
	active := 0
	ready := 0
	moduleCount := 0
	failed := 0
	partial := 0
	staged := 0
	healthy := 0
	var issues []string
	var summary strings.Builder
	for _, s := range m.services {
		if s.Active {
			active++
		}
		if s.Ready {
			ready++
		}
		if s.Module {
			moduleCount++
		}
		switch s.Status {
		case "failed":
			failed++
		case "partial":
			partial++
		case "staged":
			staged++
		}
		if s.Ready && s.Health == "ok" {
			healthy++
		}
		if issue := crateIssueLine(s); issue != "" {
			issues = append(issues, issue)
		}
	}
	summary.WriteString(renderPanelLines(
		renderCountKV("Tracked:", len(m.services)),
		renderCountKV("Active:", active),
		renderCountKV("Ready:", ready),
		renderCountKV("Unready:", len(m.services)-ready),
		renderCountKV("Crates:", moduleCount),
		renderCountKV("Healthy:", healthy),
		renderPanelKV("Failed:", danger.Render(fmt.Sprintf("%d", failed))),
		renderPanelKV("Partial:", warn.Render(fmt.Sprintf("%d", partial))),
		renderPanelKV("Staged:", dim.Render(fmt.Sprintf("%d", staged))),
	))
	var issuesBlock strings.Builder
	if len(issues) == 0 {
		issuesBlock.WriteString(ok.Render("none"))
	} else {
		issuesBlock.WriteString(renderBulletLines(issues[:minInt(len(issues), 5)]))
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("CRATE STATUS", "DIAGNOSTICS"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("trk", len(m.services)),
		selectorStat("act", active),
		selectorStat("rdy", ready),
		selectorStat("bad", failed+partial),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("CRATE SUMMARY", summary.String()))
	b.WriteString(renderWarningCard("TOP ISSUES", issuesBlock.String()))
	b.WriteString(renderActionCard("OPERATOR PATH", renderPanelLines(
		dim.Render("Use Services for direct lifecycle control, start/stop transitions, and unit-level runtime inspection."),
		dim.Render("Use Platform for adapter failures when crate status issues are caused by host-facing renders or targets."),
	)))
	return renderActivePanel("CRATE STATUS", 72, b.String())
}

func renderPlatformStatusPanel(m model) string {
	readyAdapters := 0
	failedAdapters := 0
	var platformIssues []string
	var summary strings.Builder
	var adapters strings.Builder
	for _, adapter := range m.platform.Adapters {
		switch adapter.Status {
		case "ready":
			readyAdapters++
		case "failed":
			failedAdapters++
		}
		if issue := platformIssueLine(adapter); issue != "" {
			platformIssues = append(platformIssues, issue)
		}
	}
	summary.WriteString(renderPanelLines(
		renderCountKV("Tracked:", len(m.platform.Adapters)),
		renderPanelKV("Ready:", ok.Render(fmt.Sprintf("%d", readyAdapters))),
		renderPanelKV("Failed:", danger.Render(fmt.Sprintf("%d", failedAdapters))),
	))
	if len(platformIssues) == 0 {
		summary.WriteString(renderPanelKV("Issues:", ok.Render("none")) + "\n")
	} else {
		summary.WriteString(renderPanelKV("Issues:", danger.Render(fmt.Sprintf("%d", len(platformIssues)))) + "\n")
	}
	for _, adapter := range m.platform.Adapters {
		adapters.WriteString(renderPlatformAdapterStatus(adapter))
	}
	var b strings.Builder
	b.WriteString(renderPanelTitleBar("PLATFORM ADAPTERS", "DIAGNOSTICS"))
	b.WriteString("\n")
	b.WriteString(renderStatStrip(
		selectorStat("trk", len(m.platform.Adapters)),
		selectorStat("rdy", readyAdapters),
		selectorStat("bad", failedAdapters),
		selectorStat("iss", len(platformIssues)),
	))
	b.WriteString("\n")
	b.WriteString(renderSummaryCard("ADAPTER SUMMARY", summary.String()))
	b.WriteString(renderSubsectionCard("ADAPTER GRID", adapters.String()))
	if len(platformIssues) > 0 {
		b.WriteString(renderWarningCard("FAULT LINES", renderBulletLines(platformIssues[:minInt(len(platformIssues), 4)])))
	}
	b.WriteString(renderActionCard("OPERATOR PATH", renderPanelLines(
		dim.Render("Use Network for native interface state and address inspection."),
		dim.Render("Use Services when adapter faults are secondary to crate runtime failures or staged modules."),
		dim.Render("Storage shows whether the host has any safer mounted data targets beyond the system disk."),
	)))
	return renderActivePanel("PLATFORM ADAPTERS", 72, b.String())
}
