package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func desiredManagedUnits(desired config.ServiceEntry, mod modules.Module, hasMod bool) []string {
	if hasMod {
		return crateUnits(desired.Name, mod, hasMod)
	}
	if executionAdapterForRuntime(desired.Runtime) != "systemd" {
		if strings.TrimSpace(desired.Name) == "" {
			return nil
		}
		return []string{desired.Name}
	}
	primary, companion := inferExecutionUnits(desired, mod, hasMod)
	units := []string{}
	if primary != "" {
		units = append(units, primary)
	}
	if companion != "" {
		units = append(units, companion)
	}
	if len(units) == 0 && strings.TrimSpace(desired.Name) != "" {
		units = append(units, desired.Name)
	}
	return units
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

			// If module defines extra units, ensure they are enabled/started too.
			if hasMod && mod.Runtime() == "systemd" {
				for _, pkg := range mod.Spec.Packages {
					if !packageInstalled(pkg) {
						if err := installPackage(pkg); err == nil {
							actions = append(actions, Action{
								Description: fmt.Sprintf("installed package %s for %s", pkg, desired.Name),
								Component:   "package",
								Target:      pkg,
							})
						}
					}
				}
				for _, unit := range modules.ResolveUnits(desired.Name, mod, true) {
					if !containsServiceState(unitStates, unit) {
						_ = systemctl("enable", unit)
						if mod.InstallMode() != "staged" || desired.Autostart {
							_ = systemctl("start", unit)
						}
						actions = append(actions, Action{
							Description: fmt.Sprintf("ensured module unit %s for %s", unit, desired.Name),
							Component:   "service",
							Target:      unit,
						})
					}
				}
				for _, hc := range mod.Spec.HealthChecks {
					if hc.Type == "command" && hc.Command != "" {
						if err := exec.Command("sh", "-c", hc.Command).Run(); err != nil {
							actions = append(actions, Action{
								Description: fmt.Sprintf("health check failed for %s: %s", desired.Name, hc.Command),
								Component:   "health",
								Target:      desired.Name,
							})
						}
					}
				}
			}
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
	_ = os.MkdirAll(exportBase, 0755)
	for rel, target := range links {
		linkPath := filepath.Join(exportBase, rel)

		// Skip if target doesn't exist on this system
		if _, err := os.Stat(target); err != nil {
			continue
		}

		// Skip if link already exists and points correctly
		if existing, err := os.Readlink(linkPath); err == nil && existing == target {
			continue
		}

		// Ensure parent directory
		parent := filepath.Dir(linkPath)
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

func systemctl(action string, unit ...string) error {
	args := []string{action}
	args = append(args, unit...)
	return exec.Command("systemctl", args...).Run()
}

func packageInstalled(pkg string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return exec.Command("dpkg-query", "-W", "-f=${Status}", pkg).Run() == nil
}

func installPackage(pkg string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("package install unsupported on %s", runtime.GOOS)
	}
	return exec.Command("apt-get", "install", "-y", pkg).Run()
}

func normalizeExecutionMode(mode, runtimeName string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "service", "job":
		return mode
	}
	if strings.EqualFold(strings.TrimSpace(runtimeName), "task") {
		return "job"
	}
	return "service"
}

func normalizeDurationField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeTimeoutBehavior(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "kill":
		return "kill"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeKillSignal(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "SIGTERM":
		return "SIGTERM"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func normalizeConcurrencyPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "replace":
		return "replace"
	case "forbid":
		return "forbid"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func summarizeExecutionPolicy(crate CrateState) string {
	parts := []string{}
	if crate.ActorName != "" {
		parts = append(parts, "actor "+crate.ActorName)
	}
	if crate.ExecutionMode != "" {
		parts = append(parts, "mode "+crate.ExecutionMode)
	}
	if crate.Schedule != "" {
		parts = append(parts, "schedule "+crate.Schedule)
	}
	if crate.Timeout != "" {
		parts = append(parts, "timeout "+crate.Timeout)
	}
	if crate.ConcurrencyPolicy != "" {
		parts = append(parts, "overlap "+crate.ConcurrencyPolicy)
	}
	if crate.ExecutionStatus != "" {
		parts = append(parts, "runtime "+crate.ExecutionStatus)
	}
	return strings.Join(parts, " · ")
}

func executionAdapterForRuntime(runtimeName string) string {
	switch strings.ToLower(strings.TrimSpace(runtimeName)) {
	case "docker":
		return "docker"
	default:
		return "systemd"
	}
}

func inferExecutionStatus(desired config.ServiceEntry, hasMod bool) string {
	if hasMod {
		return "module-owned"
	}
	if normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) == "job" && translateExecutionSchedule(strings.TrimSpace(desired.Execution.Schedule)) == "" {
		return "schedule-invalid"
	}
	switch normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) {
	case "job":
		return "native-timer"
	default:
		return "native-service"
	}
}

func inferExecutionUnits(desired config.ServiceEntry, mod modules.Module, hasMod bool) (string, string) {
	if hasMod {
		units := modules.ResolveUnits(desired.Name, mod, true)
		switch len(units) {
		case 0:
			return "", ""
		case 1:
			return units[0], ""
		default:
			return units[0], units[1]
		}
	}
	name := strings.TrimSpace(desired.Name)
	if name == "" {
		return "", ""
	}
	switch normalizeExecutionMode(desired.Execution.Mode, desired.Runtime) {
	case "job":
		base := managedExecutionUnitBase(name)
		return base + ".service", base + ".timer"
	default:
		base := managedExecutionUnitBase(name)
		return base + ".service", ""
	}
}

func hasModule(mods map[string]modules.Module, name string) bool {
	_, ok := mods[name]
	return ok
}

func writeCrateState(desired config.ServiceEntry, actualByName map[string]ServiceState, mods map[string]modules.Module) {
	mod, hasMod := mods[desired.Name]
	crate := buildCrateState(desired, actualByName, mod, hasMod)
	payload := StoredCrateState{
		GeneratedAt: actualTimestamp(),
		Crate:       crate,
	}
	base := platform.CratePath("services", desired.Name)
	_ = os.MkdirAll(base, 0755)
	path := filepath.Join(base, "crate-state.json")
	if err := preservePreviousStableCrateState(path, base); err != nil {
		log.Printf("warning: preserve last-good crate state %s: %v", desired.Name, err)
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("warning: marshal crate state %s: %v", desired.Name, err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("warning: write crate state %s: %v", desired.Name, err)
	}
	if err := writeManagedActorProvisioningState(crate); err != nil {
		log.Printf("warning: write actor provisioning state %s: %v", desired.Name, err)
	}
	if err := writeManagedActorRegistry(crate); err != nil {
		log.Printf("warning: write actor registry %s: %v", desired.Name, err)
	}
	if err := writeManagedActorOwnership(crate); err != nil {
		log.Printf("warning: write actor ownership %s: %v", desired.Name, err)
	}
	writeExecutionPosture(desired, crate)
	writeManagedExecutionArtifacts(desired, crate, hasMod)
	applyManagedExecutionNativeState(desired, crate, hasMod)
}

func preservePreviousStableCrateState(currentPath string, crateBase string) error {
	b, err := os.ReadFile(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var stored StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return err
	}
	if !isStableCrateState(stored.Crate) {
		return nil
	}
	lastGoodDir := filepath.Join(crateBase, "runtime", "last-good")
	if err := os.MkdirAll(lastGoodDir, 0755); err != nil {
		return err
	}
	lastGoodPath := filepath.Join(lastGoodDir, "crate-state.json")
	return os.WriteFile(lastGoodPath, b, 0644)
}

func isStableCrateState(crate CrateState) bool {
	switch crate.Status {
	case "active", "running", "inactive", "disabled", "staged":
	default:
		return false
	}

	switch crate.Health {
	case "", "ok", "pending", "unknown":
	default:
		return false
	}

	return strings.TrimSpace(crate.LastError) == ""
}

func crateProbeNames(desired config.ServiceEntry, mods map[string]modules.Module) []string {
	mod, ok := mods[desired.Name]
	names := desiredManagedUnits(desired, mod, ok)
	names = appendUnique(names, "crateos-agent", "crateos-policy")
	return names
}

func aggregateCrateStatus(units []ServiceState) string {
	if len(units) == 0 {
		return "unknown"
	}
	activeCount := 0
	failedCount := 0
	unknownCount := 0
	for _, unit := range units {
		switch unit.Status {
		case "failed":
			failedCount++
		case "active", "running":
			activeCount++
		case "", "unknown":
			unknownCount++
		}
	}
	switch {
	case failedCount > 0:
		return "failed"
	case activeCount == len(units):
		return "active"
	case activeCount > 0:
		return "partial"
	case unknownCount == len(units):
		return "unknown"
	default:
		return "inactive"
	}
}

func aggregateCrateHealth(units []ServiceState) string {
	if len(units) == 0 {
		return "unknown"
	}
	allOK := true
	for _, unit := range units {
		if unit.Health == "degraded" {
			return "degraded"
		}
		if unit.Health != "ok" {
			allOK = false
		}
	}
	if allOK {
		return "ok"
	}
	return "unknown"
}

func summarizeCrateUnits(units []ServiceState) string {
	if len(units) == 0 {
		return ""
	}
	activeCount := 0
	enabledCount := 0
	failed := make([]string, 0)
	inactive := make([]string, 0)
	for _, unit := range units {
		if unit.Active {
			activeCount++
		}
		if unit.Enabled {
			enabledCount++
		}
		if unit.Status == "failed" {
			failed = append(failed, unit.Name)
			continue
		}
		if !unit.Active {
			inactive = append(inactive, unit.Name)
		}
	}
	switch {
	case len(failed) > 0:
		return "failed units: " + strings.Join(failed, ", ")
	case activeCount > 0 && activeCount < len(units):
		return fmt.Sprintf("partially active (%d/%d units running)", activeCount, len(units))
	case enabledCount > 0 && enabledCount < len(units):
		return fmt.Sprintf("partially enabled (%d/%d units enabled)", enabledCount, len(units))
	case len(inactive) == len(units):
		return "all units inactive"
	default:
		return ""
	}
}

func crateUnits(name string, mod modules.Module, hasMod bool) []string {
	if units := modules.ResolveUnits(name, mod, hasMod); len(units) > 0 {
		return units
	}
	return nil
}

func crateUnitStates(actual []ServiceState, units []string) []ServiceState {
	states := make([]ServiceState, 0, len(units))
	for _, unit := range units {
		found := false
		for _, svc := range actual {
			if svc.Name == unit {
				states = append(states, svc)
				found = true
				break
			}
		}
		if !found {
			states = append(states, ServiceState{Name: unit, Status: "unknown", Health: "unknown"})
		}
	}
	return states
}

func containsServiceState(states []ServiceState, name string) bool {
	for _, state := range states {
		if state.Name == name {
			return true
		}
	}
	return false
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
