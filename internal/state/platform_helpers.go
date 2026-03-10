package state

import "strings"

func summarizeCommandFailure(prefix string, output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return prefix
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " | ")
	return prefix + ": " + trimmed
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
