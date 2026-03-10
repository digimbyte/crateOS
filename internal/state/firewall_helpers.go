package state

import "strings"

func sanitizeComment(value string) string {
	value = strings.ReplaceAll(value, "\"", "")
	if strings.TrimSpace(value) == "" {
		return "crateos-rule"
	}
	return value
}

func normalizeProto(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "udp":
		return "udp"
	default:
		return "tcp"
	}
}

func normalizePolicy(value string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "accept":
		return "accept"
	case "drop":
		return "drop"
	default:
		return fallback
	}
}

func maxInt(v int, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
