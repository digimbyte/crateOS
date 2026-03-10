package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/crateos/crateos/internal/platform"
)

func logSourceGlyph(source logSource) string {
	switch source.Kind {
	case "journal":
		return "◉"
	case "file":
		return "▣"
	default:
		return "•"
	}
}

func crateLogDir(crate string) string {
	return platform.CratePath("services", crate, "logs")
}

func crateLogFiles(crate string) ([]string, error) {
	entries, err := os.ReadDir(crateLogDir(crate))
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(crateLogDir(crate), entry.Name()))
	}
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] > files[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
	return files, nil
}

func readLogPreview(path string) (logPreview, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return logPreview{Content: "unable to read log file", Tail: false}, path
	}
	const maxBytes = 2000
	tail := false
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
		tail = true
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
		tail = true
	}
	return logPreview{Content: strings.Join(lines, "\n"), Tail: tail}, path
}

func journalPreviewForUnit(unit string) logPreview {
	if strings.TrimSpace(unit) == "" {
		return logPreview{}
	}
	out, err := exec.Command("journalctl", "-u", unit, "-n", "20", "--no-pager", "-o", "short-iso").Output()
	if err != nil {
		return logPreview{}
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return logPreview{}
	}
	return logPreview{Content: text, Tail: true}
}

func logSourcesForService(s ServiceInfo) []logSource {
	sources := make([]logSource, 0)
	seen := map[string]struct{}{}
	if runtime.GOOS == "linux" {
		for _, unit := range s.Units {
			preview := journalPreviewForUnit(unit.Name)
			if strings.TrimSpace(preview.Content) == "" {
				continue
			}
			key := "journal:" + unit.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			sources = append(sources, logSource{
				Kind:    "journal",
				Scope:   "unit",
				Label:   unit.Name,
				Path:    unit.Name,
				Order:   0,
				Content: preview.Content,
				Tail:    preview.Tail,
			})
		}
		if strings.TrimSpace(s.Name) != "" {
			if preview := journalPreviewForUnit(s.Name); strings.TrimSpace(preview.Content) != "" {
				key := "journal:" + s.Name
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					sources = append(sources, logSource{
						Kind:    "journal",
						Scope:   "crate",
						Label:   s.Name,
						Path:    s.Name,
						Order:   1,
						Content: preview.Content,
						Tail:    preview.Tail,
					})
				}
			}
		}
	}
	files, err := crateLogFiles(s.Name)
	if err == nil {
		for _, file := range files {
			preview, previewPath := readLogPreview(file)
			sources = append(sources, logSource{
				Kind:    "file",
				Scope:   "crate",
				Label:   filepath.Base(previewPath),
				Path:    previewPath,
				Order:   2,
				Content: preview.Content,
				Tail:    preview.Tail,
			})
		}
	}
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].Order != sources[j].Order {
			return sources[i].Order < sources[j].Order
		}
		return strings.ToLower(sources[i].Label) < strings.ToLower(sources[j].Label)
	})
	return sources
}
