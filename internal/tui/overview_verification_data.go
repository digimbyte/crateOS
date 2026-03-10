package tui

import (
	"os"
	"strings"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
)

func readFallbackVerificationDiagnostics() VerificationDiagnosticsInfo {
	info := VerificationDiagnosticsInfo{
		Status:   "ready",
		Missing:  []string{},
		Warnings: []string{},
	}
	requiredPaths := []struct {
		path  string
		label string
	}{
		{platform.CratePath("state", "installed.json"), "installed marker"},
		{platform.CratePath("state", "platform-state.json"), "platform state"},
		{platform.CratePath("state", "readiness-report.json"), "readiness report"},
		{platform.CratePath("state", "storage-state.json"), "storage state"},
		{platform.CratePath("state", "actor-ownership-state.json"), "actor ownership state"},
	}
	for _, item := range requiredPaths {
		if _, err := os.Stat(item.path); err != nil {
			info.Missing = append(info.Missing, item.label)
		}
	}
	if _, err := os.Stat(platform.AgentSocket); err == nil {
		info.AgentSocket = true
	}
	if rows := fetchUsersFromConfig(); len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(strings.TrimSpace(row.Role), "admin") {
				info.AdminPresent = true
				break
			}
		}
	}
	info.PlatformState = strings.TrimSpace(readFallbackPlatformState().GeneratedAt)
	if info.PlatformState == "" {
		info.Warnings = append(info.Warnings, "platform state not rendered yet")
	}
	if report, ok := readReadinessReport(); ok {
		info.Readiness = strings.TrimSpace(report.Status)
		if info.Readiness == "" {
			info.Readiness = "unknown"
		}
		if info.Readiness != "ready" {
			info.Warnings = append(info.Warnings, "readiness report is not ready")
		}
	} else {
		info.Warnings = append(info.Warnings, "readiness report unreadable")
	}
	if storage := state.LoadStorageState(); strings.TrimSpace(storage.GeneratedAt) != "" {
		info.StorageState = strings.TrimSpace(storage.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "storage posture not rendered yet")
	}
	if ownership := state.LoadActorOwnershipState(); strings.TrimSpace(ownership.GeneratedAt) != "" {
		info.OwnershipState = strings.TrimSpace(ownership.GeneratedAt)
	} else {
		info.Warnings = append(info.Warnings, "actor ownership state not rendered yet")
	}
	if !info.AgentSocket {
		info.Warnings = append(info.Warnings, "agent socket unavailable")
	}
	if !info.AdminPresent {
		info.Missing = append(info.Missing, "admin operator")
	}
	switch {
	case len(info.Missing) > 0:
		info.Status = "failed"
		info.Summary = "verification prerequisites missing"
	case len(info.Warnings) > 0:
		info.Status = "degraded"
		info.Summary = "verification surfaces present with warnings"
	default:
		info.Summary = "verification surfaces present"
	}
	return info
}
