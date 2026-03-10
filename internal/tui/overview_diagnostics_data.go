package tui

import (
	"strings"

	"github.com/crateos/crateos/internal/config"
)

func readFallbackDiagnostics() DiagnosticsInfo {
	ledger, err := config.LoadConfigChangeLedger()
	if err != nil {
		return DiagnosticsInfo{
			Verification: readFallbackVerificationDiagnostics(),
			Ownership:    readFallbackOwnershipDiagnostics(),
		}
	}
	info := DiagnosticsInfo{
		Config: ConfigDiagnosticsInfo{
			GeneratedAt: ledger.GeneratedAt,
			Files:       make([]ConfigDiagnosticFile, 0, len(ledger.Files)),
		},
		Verification: readFallbackVerificationDiagnostics(),
		Ownership:    readFallbackOwnershipDiagnostics(),
	}
	for _, record := range ledger.Files {
		info.Config.Tracked++
		switch strings.TrimSpace(record.Monitoring) {
		case "unmonitored":
			info.Config.Unmonitored++
		default:
			info.Config.Monitored++
		}
		if strings.TrimSpace(record.LastWriter) == "external" {
			info.Config.ExternalEdits++
		}
		info.Config.Files = append(info.Config.Files, ConfigDiagnosticFile{
			File:          record.File,
			Path:          record.Path,
			Exists:        record.Exists,
			Monitoring:    record.Monitoring,
			LastWriter:    record.LastWriter,
			LastSeenAt:    record.LastSeenAt,
			LastChangedAt: record.LastChangedAt,
		})
	}
	return info
}
