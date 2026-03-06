package tui

import (
	"fmt"
	"strings"
)

func (m model) viewNetwork() string {
	var b strings.Builder

	b.WriteString(headerBox.Render("Network Interfaces"))
	b.WriteString("\n\n")

	if len(m.interfaces) == 0 {
		b.WriteString(dim.Render("  No network interfaces detected."))
		b.WriteString("\n")
	} else {
		for _, iface := range m.interfaces {
			// Status indicator
			var status string
			if iface.Up {
				status = ok.Render("▲ UP")
			} else {
				status = dim.Render("▼ DOWN")
			}

			b.WriteString(fmt.Sprintf("  %s  %s\n", value.Render(iface.Name), status))

			if iface.MAC != "" {
				b.WriteString(fmt.Sprintf("    %s %s\n", label.Render("MAC:"), dim.Render(iface.MAC)))
			}

			if len(iface.Addrs) > 0 {
				for _, addr := range iface.Addrs {
					b.WriteString(fmt.Sprintf("    %s %s\n", label.Render("Addr:"), value.Render(addr)))
				}
			} else {
				b.WriteString(fmt.Sprintf("    %s\n", dim.Render("no addresses")))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(footer.Render("  [esc] back"))

	return b.String()
}
