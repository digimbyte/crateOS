package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"github.com/crateos/crateos/internal/api"

	"github.com/crateos/crateos/internal/logs"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
)

func main() {
	log.SetPrefix("crateos-agent: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	log.Printf("starting v%s", platform.Version)

	if err := ensureCrateRoot(); err != nil {
		log.Fatalf("failed to initialize crate root: %v", err)
	}
	if err := seedDefaultConfigs(); err != nil {
		log.Printf("warning: could not seed default configs: %v", err)
	}

	if err := writeInstalledMarker(); err != nil {
		log.Printf("warning: could not write installed marker: %v", err)
	}
	apiSrv, err := api.Start()
	if err != nil {
		log.Printf("warning: could not start API server: %v", err)
	} else {
		log.Printf("api server listening on %s", platform.AgentSocket)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run initial reconciliation
	runReconcile()

	log.Println("agent alive — entering main loop")
	for {
		select {
		case s := <-sig:
			log.Printf("received %s — shutting down", s)
			if apiSrv != nil {
				apiSrv.Stop()
			}
			return
		case <-ticker.C:
			runReconcile()
		}
	}
}

func seedDefaultConfigs() error {
	configDir := platform.CratePath("config")
	defaultRoot := platform.DefaultConfigRoot
	entries, err := os.ReadDir(defaultRoot)
	if err != nil {
		// tolerate missing packaged defaults in dev environments
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(defaultRoot, entry.Name())
		dst := filepath.Join(configDir, entry.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func runReconcile() {
	actions, err := state.Reconcile()
	if err != nil {
		log.Printf("reconcile error: %v", err)
		return
	}
	if len(actions) == 0 {
		log.Println("reconcile: system in desired state")
	} else {
		for _, a := range actions {
			log.Printf("reconcile: %s", a.Description)
		}
		log.Printf("reconcile: %d actions applied", len(actions))
	}

	// Write watchdog state file to indicate agent health
	if err := writeWatchdogState(); err != nil {
		log.Printf("watchdog state write error: %v", err)
	}

	// Export curated logs from journald
	if err := logs.ExportAll(); err != nil {
		log.Printf("log export error: %v", err)
	}
}

// ensureCrateRoot creates /srv/crateos and all required subdirectories.
func ensureCrateRoot() error {
	for _, dir := range platform.RequiredDirs {
		p := filepath.Join(platform.CrateRoot, dir)
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
	}
	log.Printf("crate root verified at %s", platform.CrateRoot)
	return nil
}

// writeInstalledMarker writes state/installed.json if it does not already exist.
func writeInstalledMarker() error {
	p := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(p); err == nil {
		return nil // already exists
	}

	data := map[string]interface{}{
		"version":      platform.Version,
		"installed_at": time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0644)
}

// writeWatchdogState writes state/agent-watchdog.json to indicate agent liveliness.
func writeWatchdogState() error {
	p := platform.CratePath("state", "agent-watchdog.json")
	data := map[string]interface{}{
		"checked_at": time.Now().UTC().Format(time.RFC3339),
		"status":     "alive",
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0644)
}
