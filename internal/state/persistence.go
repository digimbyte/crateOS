package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
)

func writeStateFile(name string, data interface{}) error {
	path := platform.CratePath("state", name)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func writeReconcileRecord(cfg *config.Config, before *ActualState, after *ActualState, actions []Action) error {
	record := ReconcileRecord{
		GeneratedAt: actualTimestamp(),
		Desired:     cfg,
		Before:      before,
		After:       after,
		Actions:     append([]Action(nil), actions...),
	}
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	stateDir := platform.CratePath("state")
	lastGoodDir := platform.CratePath("state", "last-good")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(lastGoodDir, 0755); err != nil {
		return err
	}

	latestPath := filepath.Join(stateDir, "reconcile-latest.json")
	historyPath := filepath.Join(lastGoodDir, fmt.Sprintf("reconcile-%s.json", time.Now().UTC().Format("20060102T150405Z")))
	if err := os.WriteFile(latestPath, b, 0644); err != nil {
		return err
	}
	return os.WriteFile(historyPath, b, 0644)
}

func writeServiceStates(services []ServiceState) {
	for _, svc := range services {
		base := platform.CratePath("services", svc.Name)
		_ = os.MkdirAll(base, 0755)
		path := filepath.Join(base, "state.json")
		b, err := json.MarshalIndent(svc, "", "  ")
		if err != nil {
			log.Printf("warning: marshal service state %s: %v", svc.Name, err)
			continue
		}
		if err := os.WriteFile(path, b, 0644); err != nil {
			log.Printf("warning: write service state %s: %v", svc.Name, err)
		}
	}
}

func writeCrateStates(cfg *config.Config, actual *ActualState, mods map[string]modules.Module) {
	actualByName := make(map[string]ServiceState, len(actual.Services))
	for _, svc := range actual.Services {
		actualByName[svc.Name] = svc
	}

	for _, desired := range cfg.Services.Services {
		writeCrateState(desired, actualByName, mods)
	}
	if err := pruneManagedActorOwnership(cfg); err != nil {
		log.Printf("warning: prune actor ownership index: %v", err)
	}
}
