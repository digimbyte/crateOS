package tui

import "strings"

func renderCommandStrip(m model, text string) string {
	lines := []string{renderCommandBus(text)}
	if !m.commandLaneEnabled() {
		return commandStrip.Render(strings.Join(lines, "\n"))
	}
	if m.commandMode {
		lines = append(lines, renderCommandPrompt(m.commandInput))
	} else {
		lines = append(lines, commandPromptHint.Render("[:] command mode · type help"))
	}
	if strings.TrimSpace(m.commandStatus) != "" {
		lines = append(lines, renderCommandStatusLine(m.commandStatusLevel, m.commandStatus))
	}
	return commandStrip.Render(strings.Join(lines, "\n"))
}

func renderCommandStatusLine(level, message string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "ok", "success":
		return commandStatusOK.Render(message)
	case "warn", "warning":
		return commandStatusWarn.Render(message)
	case "error", "err":
		return commandStatusError.Render(message)
	default:
		return commandStatusInfo.Render(message)
	}
}

func renderCommandPrompt(input string) string {
	cursor := commandPromptCursor.Render("█")
	if strings.TrimSpace(input) == "" {
		return commandPromptLabel.Render("INPUT") + commandPromptSep.Render(" :: ") + commandPromptInput.Render(cursor)
	}
	return commandPromptLabel.Render("INPUT") + commandPromptSep.Render(" :: ") + commandPromptInput.Render(input) + commandPromptInput.Render(cursor)
}

func renderCommandBus(text string) string {
	parts := strings.Split(text, "·")
	items := make([]string, 0, len(parts)+1)
	items = append(items, commandBusLabel.Render("COMMAND BUS"))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, formatCommandBusItem(part))
	}
	return strings.Join(items, commandBusSep.Render("  │  "))
}

func formatCommandBusItem(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	keyParts := make([]string, 0, len(fields))
	descStart := len(fields)
	for i, field := range fields {
		if looksLikeCommandToken(field) {
			keyParts = append(keyParts, field)
			continue
		}
		descStart = i
		break
	}
	if len(keyParts) == 0 {
		return dim.Render(text)
	}
	keyText := commandBusKey.Render(strings.Join(keyParts, " "))
	if descStart >= len(fields) {
		return keyText
	}
	return keyText + commandBusSep.Render(" ") + dim.Render(strings.Join(fields[descStart:], " "))
}

func looksLikeCommandToken(field string) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
		return true
	}
	switch field {
	case "↑↓", "←→", "tab", "shift+tab", "enter", "esc", "backspace", "q":
		return true
	}
	return false
}
