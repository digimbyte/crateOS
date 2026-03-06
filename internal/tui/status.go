package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

func (m model) viewStatus() string {
	var b strings.Builder

	b.WriteString(headerBox.Render("System Status"))
	b.WriteString("\n\n")

	// ── Identity ──
	row := func(l, v string) {
		b.WriteString(label.Render(l))
		b.WriteString(value.Render(v))
		b.WriteString("\n")
	}

	row("Hostname:", m.info.Hostname)
	row("Version:", platform.Version)
	row("Platform:", fmt.Sprintf("%s/%s", m.info.OS, m.info.Arch))
	row("Time:", m.info.Time.Format(time.RFC3339))
	row("CPUs:", fmt.Sprintf("%d", m.info.CPUs))
	row("Go:", m.info.GoVersion)
	b.WriteString("\n")

	// ── Crate root ──
	b.WriteString(section.Render("─── Crate Root ───"))
	b.WriteString("\n")

	root := platform.CrateRoot
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		row("Root:", ok.Render(root+" [OK]"))
	} else {
		row("Root:", warn.Render(root+" [NOT FOUND]"))
	}

	// Check installed marker
	marker := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(marker); err == nil {
		row("Installed:", ok.Render("yes"))
	} else {
		row("Installed:", dim.Render("no"))
	}

	// Check subdirectories
	missing := 0
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		if _, err := os.Stat(p); err != nil {
			missing++
		}
	}
	if missing == 0 {
		row("Directories:", ok.Render(fmt.Sprintf("all %d present", len(platform.RequiredDirs))))
	} else {
		row("Directories:", danger.Render(fmt.Sprintf("%d/%d missing", missing, len(platform.RequiredDirs))))
	}

	b.WriteString("\n")

	// ── Services summary ──
	b.WriteString(section.Render("─── Services ───"))
	b.WriteString("\n")
	active := 0
	for _, s := range m.services {
		if s.Status == "active" {
			active++
		}
	}
	row("Tracked:", fmt.Sprintf("%d", len(m.services)))
	row("Active:", fmt.Sprintf("%d", active))

	// ── Footer ──
	b.WriteString(footer.Render("  [esc] back"))

	return b.String()
}
