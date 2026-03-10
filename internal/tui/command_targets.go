package tui

import "strings"

func resolveUserTargets(users []userRow, defaultUser userRow, raw string) ([]userRow, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if strings.TrimSpace(defaultUser.Name) != "" {
			return []userRow{defaultUser}, nil
		}
		return []userRow{}, nil
	}
	parts := strings.Split(raw, ",")
	out := []userRow{}
	missing := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matched := false
		for _, u := range users {
			if strings.EqualFold(u.Name, part) {
				key := strings.ToLower(strings.TrimSpace(u.Name))
				if _, ok := seen[key]; !ok {
					out = append(out, u)
					seen[key] = struct{}{}
				}
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, part)
		}
	}
	return out, missing
}

func userNames(users []userRow) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if strings.TrimSpace(u.Name) != "" {
			out = append(out, u.Name)
		}
	}
	return out
}

func resolveServiceTargets(services []ServiceInfo, defaultService ServiceInfo, raw string) ([]ServiceInfo, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ServiceInfo{defaultService}, nil
	}
	if strings.EqualFold(raw, "all") {
		out := make([]ServiceInfo, 0, len(services))
		for _, svc := range services {
			out = append(out, svc)
		}
		return out, nil
	}
	parts := strings.Split(raw, ",")
	out := []ServiceInfo{}
	missing := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matched := false
		for _, svc := range services {
			if strings.EqualFold(svc.Name, part) || strings.EqualFold(svc.DisplayName, part) {
				key := strings.ToLower(strings.TrimSpace(svc.Name))
				if _, ok := seen[key]; !ok {
					out = append(out, svc)
					seen[key] = struct{}{}
				}
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, part)
		}
	}
	return out, missing
}

func parseCSVTargets(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := []string{}
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}
