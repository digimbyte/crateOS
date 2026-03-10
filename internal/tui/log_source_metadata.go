package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func sourceStatus(s ServiceInfo, source logSource) string {
	if unit := sourceUnit(s, source); unit != nil {
		return unit.Status
	}
	return s.Status
}

func sourceUnit(s ServiceInfo, source logSource) *ServiceUnit {
	if source.Scope != "unit" || strings.TrimSpace(source.Path) == "" {
		return nil
	}
	for i := range s.Units {
		if s.Units[i].Name == source.Path {
			return &s.Units[i]
		}
	}
	return nil
}

func sourceContextLine(s ServiceInfo, source logSource) string {
	if unit := sourceUnit(s, source); unit != nil {
		return fmt.Sprintf(
			"unit: enabled:%t  active:%t  status:%s  health:%s",
			unit.Enabled,
			unit.Active,
			unit.Status,
			unit.Health,
		)
	}
	return fmt.Sprintf(
		"crate: desired:%t  autostart:%t  active:%t  ready:%t  health:%s",
		s.Desired,
		s.Autostart,
		s.Active,
		s.Ready,
		s.Health,
	)
}

func sourceDisplayLabel(source logSource) string {
	if strings.TrimSpace(source.Label) != "" {
		return source.Label
	}
	if strings.TrimSpace(source.Path) != "" {
		return filepath.Base(source.Path)
	}
	return "unknown source"
}

func sourceKindBadge(source logSource) string {
	kind := strings.TrimSpace(source.Kind)
	if kind == "" {
		return "source"
	}
	return kind
}

func sourceDetail(source logSource) string {
	scope := strings.TrimSpace(source.Scope)
	kind := strings.TrimSpace(source.Kind)
	switch {
	case scope != "" && kind != "":
		return scope + " " + kind + " preview"
	case kind != "":
		return kind + " preview"
	case scope != "":
		return scope + " source preview"
	default:
		return "log source preview"
	}
}

func sourceSummaryLine(sources []logSource) string {
	counts := map[string]int{}
	for _, source := range sources {
		counts[sourceSummaryKey(source)]++
	}
	if len(counts) == 0 {
		return "no sources"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, counts[key]))
	}
	return strings.Join(parts, "  ")
}

func previewDetail(source logSource) string {
	if source.Tail {
		return "tail preview"
	}
	return "full short preview"
}

func sourceSummaryKey(source logSource) string {
	scope := strings.TrimSpace(source.Scope)
	kind := strings.TrimSpace(source.Kind)
	switch {
	case scope != "" && kind != "":
		return scope + " " + kind + "s"
	case kind != "":
		return kind + "s"
	case scope != "":
		return scope + " sources"
	default:
		return "sources"
	}
}
