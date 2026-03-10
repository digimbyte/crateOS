package tui

import (
	"fmt"
	"strings"
)

func renderLogHeader(selected ServiceInfo) string {
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
	return header.String()
}

func renderLogSourceSelectorSection(selected ServiceInfo, sources []logSource, sourceCursor int) string {
	sourceLines := make([]string, 0, len(sources))
	for i, source := range sources {
		line := fmt.Sprintf("%s [%s] %s", statusIndicator(sourceStatus(selected, source)), sourceKindBadge(source), sourceDisplayLabel(source))
		sourceLines = append(sourceLines, renderSelectorLineWithGlyph(i == sourceCursor, logSourceGlyph(source), line))
	}
	return renderInsetSelectorCard(
		"SOURCE SELECTOR",
		"FEEDS",
		fmt.Sprintf("%d available · %s", len(sources), sourceSummaryLine(sources)),
		sourceLines,
	)
}

func renderLogPreviewSection(selected ServiceInfo, selectedSource logSource, sourceCursor int, sourceCount int) string {
	var preview strings.Builder
	preview.WriteString(renderBadgeRow(
		dim.Render(fmt.Sprintf("source:%d/%d", sourceCursor+1, sourceCount)),
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
		return preview.String()
	}
	for _, line := range strings.Split(selectedSource.Content, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		preview.WriteString(dim.Render(line))
		preview.WriteString("\n")
	}
	return preview.String()
}
