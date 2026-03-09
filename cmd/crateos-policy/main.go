package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

const (
	maxPlatformStateAge = 20 * time.Minute
	maxWatchdogStateAge = 3 * time.Minute
	maxReadinessReportAge = 3 * time.Minute
)

type readinessReport struct {
	CheckedAt string   `json:"checked_at"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Failures  []string `json:"failures,omitempty"`
}

// checks is the list of policy assertions to verify.
var checks = []struct {
	name string
	fn   func() error
}{
	{"crate root exists", checkCrateRoot},
	{"installed marker present", checkInstalledMarker},
	{"required dirs present", checkRequiredDirs},
	{"agent socket present", checkAgentSocket},
	{"platform state present", checkPlatformState},
	{"watchdog state present", checkWatchdogState},
}

func main() {
	log.SetPrefix("crateos-policy: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	log.Printf("policy check v%s", platform.Version)

	failed := 0
	failures := []string{}
	for _, c := range checks {
		if err := c.fn(); err != nil {
			log.Printf("FAIL  %s: %v", c.name, err)
			failed++
			failures = append(failures, fmt.Sprintf("%s: %v", c.name, err))
		} else {
			log.Printf("OK    %s", c.name)
		}
	}
	if err := writeReadinessReport(failures); err != nil {
		log.Printf("FAIL  readiness report write: %v", err)
		failed++
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("policy check: %d/%d checks failed\n", failed, len(checks))
		os.Exit(1)
	}
	fmt.Printf("policy check: all %d checks passed\n", len(checks))
}

func checkCrateRoot() error {
	info, err := os.Stat(platform.CrateRoot)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", platform.CrateRoot)
	}
	return nil
}

func checkAgentSocket() error {
	info, err := os.Stat(platform.AgentSocket)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s is not a socket", platform.AgentSocket)
	}
	return nil
}

func checkPlatformState() error {
	p := platform.CratePath("state", "platform-state.json")
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if time.Since(info.ModTime()) > maxPlatformStateAge {
		return fmt.Errorf("stale: older than %s", maxPlatformStateAge)
	}
	return nil
}

func checkWatchdogState() error {
	p := platform.CratePath("state", "agent-watchdog.json")
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if time.Since(info.ModTime()) > maxWatchdogStateAge {
		return fmt.Errorf("stale: older than %s", maxWatchdogStateAge)
	}
	return nil
}

func checkInstalledMarker() error {
	p := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	return nil
}

func checkRequiredDirs() error {
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("%s missing: %w", d, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", d)
		}
	}
	return nil
}

func writeReadinessReport(failures []string) error {
	report := readinessReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "ready",
		Summary:   fmt.Sprintf("control plane ready (policy cadence <= %s, report freshness <= %s)", 2*time.Minute, maxReadinessReportAge),
	}
	if len(failures) > 0 {
		report.Status = "degraded"
		report.Summary = failures[0]
		report.Failures = failures
	}
	path := platform.CratePath("state", "readiness-report.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
