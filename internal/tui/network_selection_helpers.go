package tui

import "github.com/crateos/crateos/internal/sysinfo"

func linkBadge(up bool) string {
	if up {
		return ok.Render("link:up")
	}
	return dim.Render("link:down")
}

func networkRailGlyph(iface sysinfo.NetIface) string {
	switch {
	case iface.Up && len(iface.Addrs) > 0:
		return "◉"
	case iface.Up:
		return "◌"
	default:
		return "○"
	}
}

func networkPostureCounts(interfaces []sysinfo.NetIface) (upCount, addressedCount int) {
	for _, iface := range interfaces {
		if iface.Up {
			upCount++
		}
		if len(iface.Addrs) > 0 {
			addressedCount++
		}
	}
	return upCount, addressedCount
}
