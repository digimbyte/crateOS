package tui

import (
	"fmt"
	"strings"
)

func (m model) viewServices() string {
	var b strings.Builder

	b.WriteString(headerBox.Render("Services"))
	b.WriteString("\n\n")

	if len(m.services) == 0 {
		b.WriteString(dim.Render("  No services registered."))
		b.WriteString("\n")
	} else {
		for _, s := range m.services {
			indicator := statusIndicator(s.Status)
			name := value.Render(s.Name)
			stype := dim.Render(fmt.Sprintf("[%s]", s.Type))
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n", indicator, name, stype))
		}
	}

	b.WriteString("\n")
	b.WriteString(section.Render("─── Legend ───"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s active   %s inactive   %s failed   %s unknown\n",
		ok.Render("●"),
		dim.Render("○"),
		danger.Render("✖"),
		warn.Render("?"),
	))

	b.WriteString(footer.Render("  [esc] back"))

	return b.String()
}

func statusIndicator(status string) string {
	switch status {
	case "active", "running":
		return ok.Render("●")
	case "inactive", "dead":
		return dim.Render("○")
	case "failed":
		return danger.Render("✖")
	default:
		return warn.Render("?")
	}
}
