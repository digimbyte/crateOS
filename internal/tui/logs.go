package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/crateos/crateos/internal/platform"
)

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

type logSource struct {
	Kind    string
	Scope   string
	Label   string
	Path    string
	Order   int
	Content string
	Tail    bool
}

type logPreview struct {
	Content string
	Tail    bool
}

func (m model) viewLogs() string {
	return renderSplitView(m, "Logs", renderLogServiceSelectionPanel(m), renderLogFocusPanel(m), "  ↑↓ select crate · ←→ / tab select source · [esc] back · [:] command")
}

func renderLogServiceSelectionPanel(m model) string {
	lines := []string{}
	if len(m.services) == 0 {
		lines = append(lines, dim.Render("  No crates available for log inspection."))
		return renderSelectionPanel("LOG TARGETS", 34, lines)
	}
	for i, svc := range m.services {
		name := svc.DisplayName
		if strings.TrimSpace(name) == "" {
			name = svc.Name
		}
		line := fmt.Sprintf(
			"%s %s  %s",
			statusIndicator(svc.Status),
			compactLabel(name, 12),
			renderBadgeRow(selectorStat("s", svc.Status), selectorStat("r", boolToRail(svc.Ready))),
		)
		lines = append(lines, renderSelectorLineWithGlyph(i == m.currentServiceIndex(), serviceRailGlyph(svc), line))
	}
	return renderSelectionPanelWithMeta("LOG TARGETS", "BUS", 34, lines)
}

func renderLogFocusPanel(m model) string {
	if len(m.services) == 0 {
		return renderActivePanel("ACTIVE LOG PANEL", 68, dim.Render("No crate selected."))
	}
	selected := m.currentLogService()
	sources := logSourcesForService(selected)
	sourceCursor := m.currentLogSourceIndex()
	displayName := selected.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = selected.Name
	}
	var header strings.Builder
	header.WriteString(renderPanelTitleBar(strings.ToUpper(displayName), "LOG BUS"))
	header.WriteString("\n")
	header.WriteString(renderStatStrip(
		lifecycleDesiredBadge(selected),
		lifecycleAutostartBadge(selected),
		lifecycleRuntimeBadge(selected),
		readyBadge(selected.Ready),
	))
	header.WriteString("\n")
	header.WriteString(dim.Render("units: " + lifecycleUnitCounts(selected)))
	header.WriteString("\n")
	if selected.Summary != "" {
		header.WriteString(dim.Render("summary: " + selected.Summary))
		header.WriteString("\n")
	}
	if issue := crateIssueLine(selected); issue != "" {
		header.WriteString(warn.Render("issue: " + issue))
		header.WriteString("\n")
	}
	if len(sources) == 0 {
		var b strings.Builder
		b.WriteString(header.String())
		b.WriteString(renderInsetSelectorCard("SOURCE SELECTOR", "FEEDS", "No log sources available.", nil))
		return renderActivePanel("ACTIVE LOG PANEL", 68, b.String())
	}
	sourceLines := make([]string, 0, len(sources))
	for i, source := range sources {
		line := fmt.Sprintf("%s [%s] %s", statusIndicator(sourceStatus(selected, source)), sourceKindBadge(source), sourceDisplayLabel(source))
		sourceLines = append(sourceLines, renderSelectorLineWithGlyph(i == sourceCursor, logSourceGlyph(source), line))
	}
	selectedSource := sources[sourceCursor]
	var preview strings.Builder
	preview.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("source:%d/%d", sourceCursor+1, len(sources))),
		dim.Render("kind:"+sourceKindBadge(selectedSource)),
		statusBadge(sourceStatus(selected, selectedSource)),
	))
	preview.WriteString("\n")
	preview.WriteString(value.Render(sourceDisplayLabel(selectedSource)))
	preview.WriteString("\n")
	preview.WriteString(dim.Render("detail: " + sourceDetail(selectedSource)))
	preview.WriteString("\n")
	preview.WriteString(dim.Render("preview: " + previewDetail(selectedSource)))
	preview.WriteString("\n")
	if context := sourceContextLine(selected, selectedSource); context != "" {
		preview.WriteString(dim.Render(context))
		preview.WriteString("\n")
	}
	if strings.TrimSpace(selectedSource.Path) != "" {
		preview.WriteString(dim.Render("source: " + selectedSource.Path))
		preview.WriteString("\n")
	}
	preview.WriteString("\n")
	if strings.TrimSpace(selectedSource.Content) == "" {
		preview.WriteString(dim.Render("Selected source is empty."))
		preview.WriteString("\n")
	} else {
		for _, line := range strings.Split(selectedSource.Content, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			preview.WriteString(dim.Render(line))
			preview.WriteString("\n")
		}
	}
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(renderInsetSelectorCard(
		"SOURCE SELECTOR",
		"FEEDS",
		fmt.Sprintf("%d available · %s", len(sources), sourceSummaryLine(sources)),
		sourceLines,
	))
	b.WriteString(renderSummaryCard("PREVIEW", preview.String()))
	return renderActivePanel("ACTIVE LOG PANEL", 68, b.String())
}

func logSourceGlyph(source logSource) string {
	switch source.Kind {
	case "journal":
		return "◉"
	case "file":
		return "▣"
	default:
		return "•"
	}
}

func crateLogDir(crate string) string {
	return platform.CratePath("services", crate, "logs")
}

func crateLogFiles(crate string) ([]string, error) {
	entries, err := os.ReadDir(crateLogDir(crate))
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(crateLogDir(crate), entry.Name()))
	}
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] > files[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
	return files, nil
}

func readLogPreview(path string) (logPreview, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return logPreview{Content: "unable to read log file", Tail: false}, path
	}
	const maxBytes = 2000
	tail := false
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
		tail = true
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
		tail = true
	}
	return logPreview{Content: strings.Join(lines, "\n"), Tail: tail}, path
}

func journalPreviewForUnit(unit string) logPreview {
	if strings.TrimSpace(unit) == "" {
		return logPreview{}
	}
	out, err := exec.Command("journalctl", "-u", unit, "-n", "20", "--no-pager", "-o", "short-iso").Output()
	if err != nil {
		return logPreview{}
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return logPreview{}
	}
	return logPreview{Content: text, Tail: true}
}

func logSourcesForService(s ServiceInfo) []logSource {
	sources := make([]logSource, 0)
	seen := map[string]struct{}{}
	if runtime.GOOS == "linux" {
		for _, unit := range s.Units {
			preview := journalPreviewForUnit(unit.Name)
			if strings.TrimSpace(preview.Content) == "" {
				continue
			}
			key := "journal:" + unit.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			sources = append(sources, logSource{
				Kind:    "journal",
				Scope:   "unit",
				Label:   unit.Name,
				Path:    unit.Name,
				Order:   0,
				Content: preview.Content,
				Tail:    preview.Tail,
			})
		}
		if strings.TrimSpace(s.Name) != "" {
			if preview := journalPreviewForUnit(s.Name); strings.TrimSpace(preview.Content) != "" {
				key := "journal:" + s.Name
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					sources = append(sources, logSource{
						Kind:    "journal",
						Scope:   "crate",
						Label:   s.Name,
						Path:    s.Name,
						Order:   1,
						Content: preview.Content,
						Tail:    preview.Tail,
					})
				}
			}
		}
	}
	files, err := crateLogFiles(s.Name)
	if err == nil {
		for _, file := range files {
			preview, previewPath := readLogPreview(file)
			sources = append(sources, logSource{
				Kind:    "file",
				Scope:   "crate",
				Label:   filepath.Base(previewPath),
				Path:    previewPath,
				Order:   2,
				Content: preview.Content,
				Tail:    preview.Tail,
			})
		}
	}
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].Order != sources[j].Order {
			return sources[i].Order < sources[j].Order
		}
		return strings.ToLower(sources[i].Label) < strings.ToLower(sources[j].Label)
	})
	return sources
}

func sourceStatus(s ServiceInfo, source logSource) string {
	if unit := sourceUnit(s, source); unit != nil {
		return unit.Status
	}
	return s.Status
}

func sourceUnit(s ServiceInfo, source logSource) *ServiceUnit {
	if source.Scope != "unit" || strings.TrimSpace(source.Path) == "" {
		return nil
	}
	for i := range s.Units {
		if s.Units[i].Name == source.Path {
			return &s.Units[i]
		}
	}
	return nil
}

func sourceContextLine(s ServiceInfo, source logSource) string {
	if unit := sourceUnit(s, source); unit != nil {
		return fmt.Sprintf(
			"unit: enabled:%t  active:%t  status:%s  health:%s",
			unit.Enabled,
			unit.Active,
			unit.Status,
			unit.Health,
		)
	}
	return fmt.Sprintf(
		"crate: desired:%t  autostart:%t  active:%t  ready:%t  health:%s",
		s.Desired,
		s.Autostart,
		s.Active,
		s.Ready,
		s.Health,
	)
}

func sourceDisplayLabel(source logSource) string {
	if strings.TrimSpace(source.Label) != "" {
		return source.Label
	}
	if strings.TrimSpace(source.Path) != "" {
		return filepath.Base(source.Path)
	}
	return "unknown source"
}

func sourceKindBadge(source logSource) string {
	kind := strings.TrimSpace(source.Kind)
	if kind == "" {
		return "source"
	}
	return kind
}

func sourceDetail(source logSource) string {
	scope := strings.TrimSpace(source.Scope)
	kind := strings.TrimSpace(source.Kind)
	switch {
	case scope != "" && kind != "":
		return scope + " " + kind + " preview"
	case kind != "":
		return kind + " preview"
	case scope != "":
		return scope + " source preview"
	default:
		return "log source preview"
	}
}

func sourceSummaryLine(sources []logSource) string {
	counts := map[string]int{}
	for _, source := range sources {
		counts[sourceSummaryKey(source)]++
	}
	if len(counts) == 0 {
		return "no sources"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, counts[key]))
	}
	return strings.Join(parts, "  ")
}

func previewDetail(source logSource) string {
	if source.Tail {
		return "tail preview"
	}
	return "full short preview"
}

func sourceSummaryKey(source logSource) string {
	scope := strings.TrimSpace(source.Scope)
	kind := strings.TrimSpace(source.Kind)
	switch {
	case scope != "" && kind != "":
		return scope + " " + kind + "s"
	case kind != "":
		return kind + "s"
	case scope != "":
		return scope + " sources"
	default:
		return "sources"
	}
}
