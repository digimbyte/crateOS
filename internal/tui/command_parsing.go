package tui

import (
	"strings"
	"unicode"
)

func tokenizeCommandInput(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	args := []string{}
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && unicode.IsSpace(rune(ch)) {
			flush()
			continue
		}
		b.WriteByte(ch)
	}
	flush()
	return args
}

func splitCommandChain(raw string) []string {
	out := []string{}
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		part := strings.TrimSpace(b.String())
		if part != "" {
			out = append(out, part)
		}
		b.Reset()
	}
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}
		if !inSingle && !inDouble {
			if ch == ';' {
				flush()
				continue
			}
			if ch == '&' && i+1 < len(raw) && raw[i+1] == '&' {
				flush()
				i++
				continue
			}
		}
		b.WriteByte(ch)
	}
	flush()
	return out
}
