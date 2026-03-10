package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

type readinessReportView struct {
	CheckedAt string   `json:"checked_at"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Failures  []string `json:"failures"`
}

const maxReadinessReportAge = 3 * time.Minute

func readReadinessReport() (readinessReportView, bool) {
	data, err := os.ReadFile(platform.CratePath("state", "readiness-report.json"))
	if err != nil {
		return readinessReportView{}, false
	}
	var report readinessReportView
	if err := json.Unmarshal(data, &report); err != nil {
		return readinessReportView{}, false
	}
	report.applyFreshness(time.Now().UTC())
	return report, true
}

func (r readinessReportView) statusText() string {
	switch strings.TrimSpace(r.Status) {
	case "ready":
		return ok.Render("ready")
	case "degraded":
		return danger.Render("degraded")
	default:
		return warn.Render("unknown")
	}
}

func (r *readinessReportView) applyFreshness(now time.Time) {
	checkedAtRaw := strings.TrimSpace(r.CheckedAt)
	if checkedAtRaw == "" {
		r.markDegraded("readiness report missing checked_at")
		return
	}
	checkedAt, err := time.Parse(time.RFC3339, checkedAtRaw)
	if err != nil {
		r.markDegraded("readiness report has invalid checked_at")
		return
	}
	age := now.Sub(checkedAt)
	if age > maxReadinessReportAge {
		r.markDegraded(fmt.Sprintf("readiness report stale: last policy update %s ago", age.Round(time.Second)))
	}
}

func (r *readinessReportView) markDegraded(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "readiness report degraded"
	}
	r.Status = "degraded"
	r.Summary = reason
	if len(r.Failures) == 0 || strings.TrimSpace(r.Failures[0]) != reason {
		r.Failures = append([]string{reason}, r.Failures...)
	}
}
