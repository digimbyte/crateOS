package tui

func menuServiceCounts(services []ServiceInfo) (failed, partial, staged, healthy int) {
	for _, s := range services {
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
	}
	return failed, partial, staged, healthy
}

func menuTopIssues(services []ServiceInfo, limit int) []string {
	issues := make([]string, 0, limit)
	for _, s := range services {
		if issue := crateIssueLine(s); issue != "" {
			issues = append(issues, issue)
			if len(issues) >= limit {
				break
			}
		}
	}
	return issues
}

func menuPlatformCounts(adapters []PlatformAdapter) (ready, failed int) {
	for _, adapter := range adapters {
		switch adapter.Status {
		case "ready":
			ready++
		case "failed":
			failed++
		}
	}
	return ready, failed
}

func menuTopPlatformIssues(adapters []PlatformAdapter, limit int) []string {
	issues := make([]string, 0, limit)
	for _, adapter := range adapters {
		if issue := platformIssueLine(adapter); issue != "" {
			issues = append(issues, issue)
			if len(issues) >= limit {
				break
			}
		}
	}
	return issues
}
