package state

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/users"
)

// Action represents a single remediation step.
type Action struct {
	Description string `json:"description"`
	Component   string `json:"component"` // "directory", "service", "network", "symlink"
	Target      string `json:"target"`
}

type ReconcileRecord struct {
	GeneratedAt string         `json:"generated_at"`
	Desired     *config.Config `json:"desired"`
	Before      *ActualState   `json:"before"`
	After       *ActualState   `json:"after"`
	Actions     []Action       `json:"actions"`
}

// Reconcile loads config, probes state, computes a diff, applies changes,
// and writes the state files. Returns the list of actions taken.
func Reconcile() ([]Action, error) {
	// Load desired state from config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	mods := modules.LoadAll(".")

	// Collect service names from config
	var svcNames []string
	for _, s := range cfg.Services.Services {
		svcNames = append(svcNames, s.Name)
	}
	// Always include CrateOS's own services
	svcNames = appendUnique(svcNames, "crateos-agent", "crateos-policy")

	// Probe actual state
	before := Probe(svcNames)

	// Compute and apply diff
	actions := reconcile(cfg, before, mods)

	// Re-probe after apply so reconcile history captures the transition, not just intent.
	after := Probe(svcNames)

	// Write state files
	if err := writeStateFile("desired.json", cfg); err != nil {
		log.Printf("warning: failed to write desired.json: %v", err)
	}
	if err := writeStateFile("actual.json", after); err != nil {
		log.Printf("warning: failed to write actual.json: %v", err)
	}
	writeServiceStates(after.Services)
	writeCrateStates(cfg, after, mods)
	if err := writeReconcileRecord(cfg, before, after, actions); err != nil {
		log.Printf("warning: failed to write reconcile record: %v", err)
	}

	return actions, nil
}

func RefreshCrateState(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	mods := modules.LoadAll(".")
	var desired *config.ServiceEntry
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == name {
			desired = &cfg.Services.Services[i]
			break
		}
	}
	if desired == nil {
		return fmt.Errorf("service %s not found", name)
	}
	actual := Probe(crateProbeNames(*desired, mods))
	actualByName := make(map[string]ServiceState, len(actual.Services))
	for _, svc := range actual.Services {
		actualByName[svc.Name] = svc
	}
	writeServiceStates(actual.Services)
	writeCrateState(*desired, actualByName, mods)
	return nil
}

func reconcile(cfg *config.Config, actual *ActualState, mods map[string]modules.Module) []Action {
	var actions []Action

	// ── Ensure directories ──
	for _, d := range platform.RequiredDirs {
		if !actual.Directories[d] {
			p := platform.CratePath(d)
			if err := os.MkdirAll(p, 0755); err != nil {
				log.Printf("error: mkdir %s: %v", p, err)
				continue
			}
			actions = append(actions, Action{
				Description: fmt.Sprintf("created directory %s", p),
				Component:   "directory",
				Target:      p,
			})
		}
	}

	// ── Ensure per-service directories ──
	for _, svc := range cfg.Services.Services {
		base := platform.CratePath("services", svc.Name)
		dirs := []string{
			filepath.Join(base, "config"),
			filepath.Join(base, "data"),
			filepath.Join(base, "logs"),
			filepath.Join(base, "runtime"),
			filepath.Join(base, "backups"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				log.Printf("error: mkdir %s: %v", d, err)
				continue
			}
			actions = append(actions, Action{
				Description: fmt.Sprintf("ensured service dir %s", d),
				Component:   "directory",
				Target:      d,
			})
		}
	}

	// ── Ensure services ── (Linux only)
	if runtime.GOOS == "linux" {
		for _, desired := range cfg.Services.Services {
			if !desired.Enabled {
				continue
			}
			mod, hasMod := mods[desired.Name]
			targetUnits := crateUnits(desired.Name, mod, hasMod)
			unitStates := crateUnitStates(actual.Services, targetUnits)
			for _, unitState := range unitStates {
				if !unitState.Enabled {
					if err := systemctl("enable", unitState.Name); err == nil {
						actions = append(actions, Action{
							Description: fmt.Sprintf("enabled %s for %s", unitState.Name, desired.Name),
							Component:   "service",
							Target:      unitState.Name,
						})
					}
				}
				if desired.Autostart && !unitState.Active {
					if err := systemctl("start", unitState.Name); err == nil {
						actions = append(actions, Action{
							Description: fmt.Sprintf("started %s for %s", unitState.Name, desired.Name),
							Component:   "service",
							Target:      unitState.Name,
						})
					}
				}
			}

			actions = append(actions, reconcileModuleUnits(desired, mod, hasMod, unitStates)...)
		}
		actions = append(actions, activateHostedManagedWorkloads(cfg, actual, mods)...)
	}

	// ── User provisioning ──
	if runtime.GOOS == "linux" {
		_, _, err := users.ProvisionUsers(cfg)
		if err != nil {
			log.Printf("warning: user provisioning failed: %v", err)
		} else {
			actions = append(actions, Action{
				Description: "provisioned CrateOS users to system accounts",
				Component:   "users",
				Target:      "system",
			})
		}
	}

	// ── Platform adapters ──
	actions = append(actions, reconcilePlatform(cfg)...)
	// ── Export symlinks ──
	actions = append(actions, ensureExports()...)

	return actions
}
