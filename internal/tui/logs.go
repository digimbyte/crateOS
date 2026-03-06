package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/crateos/crateos/internal/platform"
)

func (m model) viewLogs() string {
	var b strings.Builder

	b.WriteString(headerBox.Render("Logs"))
	b.WriteString("\n\n")

	logDir := platform.CratePath("logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		b.WriteString(warn.Render(fmt.Sprintf("  Log directory not available: %s", logDir)))
		b.WriteString("\n")
		b.WriteString(dim.Render("  Run crateos-agent first to initialize the crate root."))
		b.WriteString("\n")
	} else if len(entries) == 0 {
		b.WriteString(dim.Render("  No log files yet."))
		b.WriteString("\n")
	} else {
		b.WriteString(section.Render("─── Available Logs ───"))
		b.WriteString("\n")
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, _ := e.Info()
			size := ""
			if info != nil {
				size = dim.Render(fmt.Sprintf("(%d bytes)", info.Size()))
			}
			b.WriteString(fmt.Sprintf("  %s  %s\n", value.Render(e.Name()), size))
		}
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("  Log browsing/search will be available in a future release."))
	b.WriteString("\n")
	b.WriteString(footer.Render("  [esc] back"))

	return b.String()
}
