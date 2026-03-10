package tui

import "strings"

func userRoleIndicator(role string) string {
	switch role {
	case "admin":
		return danger.Render("◆")
	case "operator":
		return warn.Render("◈")
	case "staff":
		return ok.Render("●")
	case "viewer":
		return dim.Render("○")
	default:
		return warn.Render("?")
	}
}

func userRailGlyph(u userRow) string {
	switch u.Role {
	case "admin":
		return "◆"
	case "operator":
		return "◈"
	case "staff":
		return "●"
	case "viewer":
		return "○"
	default:
		return "?"
	}
}

func nextRole(role string) string {
	order := []string{"viewer", "operator", "staff", "admin"}
	for i, r := range order {
		if r == role {
			return order[(i+1)%len(order)]
		}
	}
	return "staff"
}

func togglePermPreset(current []string) []string {
	joined := strings.Join(current, ",")
	switch joined {
	case "", "logs.view,svc.list,net.status", "svc.list,logs.view,net.status":
		return []string{"users.view", "logs.view", "svc.list", "net.status"}
	case "users.view,logs.view,svc.list,net.status":
		return []string{"svc.*", "net.*", "proxy.*", "logs.view"}
	default:
		return []string{"logs.view", "svc.list", "net.status"}
	}
}
