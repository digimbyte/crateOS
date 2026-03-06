package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

// Action represents a single remediation step.
type Action struct {
	Description string `json:"description"`
	Component   string `json:"component"` // "directory", "service", "network", "symlink"
	Target      string `json:"target"`
}

// Reconcile loads config, probes state, computes a diff, applies changes,
// and writes the state files. Returns the list of actions taken.
func Reconcile() ([]Action, error) {
	// Load desired state from config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Collect service names from config
	var svcNames []string
	for _, s := range cfg.Services.Services {
		svcNames = append(svcNames, s.Name)
	}
	// Always include CrateOS's own services
	svcNames = appendUnique(svcNames, "crateos-agent", "crateos-policy")

	// Probe actual state
	actual := Probe(svcNames)

	// Compute and apply diff
	actions := reconcile(cfg, actual)

	// Write state files
	if err := writeStateFile("desired.json", cfg); err != nil {
		log.Printf("warning: failed to write desired.json: %v", err)
	}
	if err := writeStateFile("actual.json", actual); err != nil {
		log.Printf("warning: failed to write actual.json: %v", err)
	}

	return actions, nil
}

func reconcile(cfg *config.Config, actual *ActualState) []Action {
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

	// ── Ensure services ── (Linux only)
	if runtime.GOOS == "linux" {
		for _, desired := range cfg.Services.Services {
			if !desired.Enabled {
				continue
			}

			var actualSvc *ServiceState
			for i := range actual.Services {
				if actual.Services[i].Name == desired.Name {
					actualSvc = &actual.Services[i]
					break
				}
			}

			// Enable if not enabled
			if actualSvc != nil && !actualSvc.Enabled {
				if err := systemctl("enable", desired.Name); err == nil {
					actions = append(actions, Action{
						Description: fmt.Sprintf("enabled %s", desired.Name),
						Component:   "service",
						Target:      desired.Name,
					})
				}
			}

			// Start if not active and autostart is on
			if desired.Autostart && (actualSvc == nil || !actualSvc.Active) {
				if err := systemctl("start", desired.Name); err == nil {
					actions = append(actions, Action{
						Description: fmt.Sprintf("started %s", desired.Name),
						Component:   "service",
						Target:      desired.Name,
					})
				}
			}
		}
	}

	// ── Export symlinks ──
	actions = append(actions, ensureExports()...)

	return actions
}

// ensureExports creates the symlink farm under /srv/crateos/export/.
func ensureExports() []Action {
	if runtime.GOOS != "linux" {
		return nil
	}

	var actions []Action
	links := map[string]string{
		"etc/NetworkManager": "/etc/NetworkManager",
		"etc/nginx":          "/etc/nginx",
		"etc/nftables.conf":  "/etc/nftables.conf",
		"etc/ssh":            "/etc/ssh",
		"var/log/journal":    "/var/log/journal",
	}

	exportBase := platform.CratePath("export")
	for rel, target := range links {
		linkPath := fmt.Sprintf("%s/%s", exportBase, rel)

		// Skip if target doesn't exist on this system
		if _, err := os.Stat(target); err != nil {
			continue
		}

		// Skip if link already exists and points correctly
		if existing, err := os.Readlink(linkPath); err == nil && existing == target {
			continue
		}

		// Ensure parent directory
		parent := linkPath[:len(linkPath)-len(rel[len(rel)-len(rel):])]
		_ = os.MkdirAll(parent, 0755)

		// Remove stale link if present
		os.Remove(linkPath)

		if err := os.Symlink(target, linkPath); err != nil {
			log.Printf("warning: symlink %s -> %s: %v", linkPath, target, err)
			continue
		}

		actions = append(actions, Action{
			Description: fmt.Sprintf("linked %s -> %s", linkPath, target),
			Component:   "symlink",
			Target:      linkPath,
		})
	}

	return actions
}

func systemctl(action, unit string) error {
	return exec.Command("systemctl", action, unit).Run()
}

func writeStateFile(name string, data interface{}) error {
	path := platform.CratePath("state", name)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}
