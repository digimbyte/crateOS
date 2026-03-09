package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var normalizeExtAllowlist = map[string]struct{}{
	".yaml": {}, ".yml": {}, ".json": {}, ".toml": {}, ".ini": {}, ".conf": {}, ".cfg": {}, ".txt": {}, ".env": {}, ".service": {},
}

// NormalizeTreeIfNeeded recursively evaluates regular files under root and normalizes only CRLF/mixed candidates.
// Returns normalizedCount and scannedCount.
func NormalizeTreeIfNeeded(root string) (int, int, error) {
	if runtime.GOOS != "linux" {
		return 0, 0, nil
	}
	normalizedCount := 0
	scannedCount := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		scannedCount++
		normalized, normalizeErr := NormalizeFileIfNeeded(path)
		if normalizeErr != nil {
			return normalizeErr
		}
		if normalized {
			normalizedCount++
		}
		return nil
	})
	if err != nil {
		return normalizedCount, scannedCount, err
	}
	return normalizedCount, scannedCount, nil
}

// OnFTPUploadComplete normalizes a newly uploaded file when applicable.
func OnFTPUploadComplete(path string) error {
	_, err := NormalizeFileIfNeeded(path)
	return err
}

// OnFTPUploadCompleteTarget normalizes a completed FTP target path.
// If path is a file, it evaluates only that file; if path is a directory, it scans recursively.
// Returns normalizedCount and scannedCount.
func OnFTPUploadCompleteTarget(path string) (int, int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	if info.IsDir() {
		return NormalizeTreeIfNeeded(path)
	}
	normalized, err := NormalizeFileIfNeeded(path)
	if err != nil {
		return 0, 1, err
	}
	if normalized {
		return 1, 1, nil
	}
	return 0, 1, nil
}

// OnWebFormSave normalizes a file written through web-form save flows.
func OnWebFormSave(path string) error {
	_, err := NormalizeFileIfNeeded(path)
	return err
}

// OnTUISave normalizes a file written through TUI save flows.
func OnTUISave(path string) error {
	_, err := NormalizeFileIfNeeded(path)
	return err
}

// NormalizeFileIfNeeded runs dos2unix only for known text/config files that currently contain CRLF or mixed line endings.
// Returns true when normalization is applied.
func NormalizeFileIfNeeded(path string) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, nil
	}
	if !isNormalizeCandidate(path) {
		return false, nil
	}
	mode, err := DetectLineEndingMode(path)
	if err != nil {
		return false, err
	}
	if mode != "crlf" && mode != "mixed" {
		return false, nil
	}
	if err := exec.Command("dos2unix", "-q", path).Run(); err != nil {
		return false, err
	}
	return true, nil
}

func isNormalizeCandidate(path string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	_, ok := normalizeExtAllowlist[ext]
	return ok
}

func DetectLineEndingMode(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "none", nil
	}
	crlfCount := 0
	lfStandaloneCount := 0
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' {
			continue
		}
		if i > 0 && data[i-1] == '\r' {
			crlfCount++
			continue
		}
		lfStandaloneCount++
	}
	switch {
	case crlfCount > 0 && lfStandaloneCount > 0:
		return "mixed", nil
	case crlfCount > 0:
		return "crlf", nil
	case lfStandaloneCount > 0:
		return "lf", nil
	default:
		return "none", nil
	}
}
