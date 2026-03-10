package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
)

func loadCrateState(name string) state.CrateState {
	path := platform.CratePath("services", name, "crate-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return state.CrateState{Name: name, DisplayName: name, Status: "unknown", Health: "unknown"}
	}
	var stored state.StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return state.CrateState{Name: name, DisplayName: name, Status: "unknown", Health: "unknown"}
	}
	applyStoredCrateStateFreshness(&stored, time.Now().UTC())
	if stored.Crate.Name == "" {
		stored.Crate.Name = name
	}
	if stored.Crate.DisplayName == "" {
		stored.Crate.DisplayName = name
	}
	if stored.Crate.Status == "" {
		stored.Crate.Status = "unknown"
	}
	if stored.Crate.Health == "" {
		stored.Crate.Health = "unknown"
	}
	return stored.Crate
}

func loadLastGoodCrateState(name string) (state.StoredCrateState, bool) {
	path := platform.CratePath("services", name, "runtime", "last-good", "crate-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return state.StoredCrateState{}, false
	}
	var stored state.StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return state.StoredCrateState{}, false
	}
	if stored.Crate.Name == "" {
		stored.Crate.Name = name
	}
	if stored.Crate.DisplayName == "" {
		stored.Crate.DisplayName = name
	}
	return stored, true
}

func applyStoredCrateStateFreshness(stored *state.StoredCrateState, now time.Time) {
	generatedAtRaw := strings.TrimSpace(stored.GeneratedAt)
	if generatedAtRaw == "" {
		markStoredCrateStateStale(stored, "crate state missing generated_at")
		return
	}
	generatedAt, err := time.Parse(time.RFC3339, generatedAtRaw)
	if err != nil {
		markStoredCrateStateStale(stored, "crate state has invalid generated_at")
		return
	}
	age := now.Sub(generatedAt)
	if age > maxCrateStateAge {
		markStoredCrateStateStale(stored, fmt.Sprintf("crate state stale: last agent render %s ago", age.Round(time.Second)))
	}
}

func markStoredCrateStateStale(stored *state.StoredCrateState, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "crate state stale"
	}
	stored.Crate.Status = "failed"
	stored.Crate.Health = "degraded"
	stored.Crate.Ready = false
	stored.Crate.LastError = reason
	if strings.TrimSpace(stored.Crate.Summary) == "" || strings.TrimSpace(stored.Crate.Summary) == "rendered desired state successfully" {
		stored.Crate.Summary = reason
	}
}
