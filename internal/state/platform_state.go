package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
)

type PlatformState struct {
	GeneratedAt string                 `json:"generated_at"`
	Adapters    []PlatformAdapterState `json:"adapters"`
}

type PlatformAdapterState struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Enabled       bool     `json:"enabled"`
	Status        string   `json:"status"`
	Health        string   `json:"health"`
	Summary       string   `json:"summary,omitempty"`
	LastError     string   `json:"last_error,omitempty"`
	Validation    string   `json:"validation,omitempty"`
	ValidationErr string   `json:"validation_error,omitempty"`
	Apply         string   `json:"apply,omitempty"`
	ApplyErr      string   `json:"apply_error,omitempty"`
	RenderedPaths []string `json:"rendered_paths,omitempty"`
	NativeTargets []string `json:"native_targets,omitempty"`
}

type StorageState struct {
	GeneratedAt   string               `json:"generated_at"`
	Devices       []StorageDeviceState `json:"devices,omitempty"`
	SafeTargets   []string             `json:"safe_targets,omitempty"`
	SystemTargets []string             `json:"system_targets,omitempty"`
}

type ActorOwnershipState struct {
	GeneratedAt string                    `json:"generated_at"`
	Active      int                       `json:"active"`
	Retired     int                       `json:"retired"`
	Claims      []ActorOwnershipStateItem `json:"claims,omitempty"`
}

type ActorOwnershipStateItem struct {
	Crate     string `json:"crate"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	ID        string `json:"id,omitempty"`
	User      string `json:"user,omitempty"`
	Group     string `json:"group,omitempty"`
	Home      string `json:"home,omitempty"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at,omitempty"`
	RetiredAt string `json:"retired_at,omitempty"`
}

type ActorProvisioningState struct {
	GeneratedAt   string                        `json:"generated_at"`
	LastSuccessAt string                        `json:"last_success_at,omitempty"`
	LastFailureAt string                        `json:"last_failure_at,omitempty"`
	Events        []ActorProvisioningStateEvent `json:"events,omitempty"`
	Actor         ActorProvisioningActorState   `json:"actor"`
	Provisioning  string                        `json:"provisioning,omitempty"`
	Error         string                        `json:"error,omitempty"`
}

type ActorProvisioningActorState struct {
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	ID         string `json:"id,omitempty"`
	User       string `json:"user,omitempty"`
	Group      string `json:"group,omitempty"`
	Home       string `json:"home,omitempty"`
	RuntimeDir string `json:"runtime_dir,omitempty"`
	StateDir   string `json:"state_dir,omitempty"`
}

type ActorProvisioningStateEvent struct {
	At           string `json:"at"`
	Provisioning string `json:"provisioning,omitempty"`
	Error        string `json:"error,omitempty"`
}

type StorageDeviceState struct {
	Name       string `json:"name"`
	Path       string `json:"path,omitempty"`
	Type       string `json:"type,omitempty"`
	Mountpoint string `json:"mountpoint,omitempty"`
	FSType     string `json:"fs_type,omitempty"`
	Size       string `json:"size,omitempty"`
	Class      string `json:"class,omitempty"`
	Removable  bool   `json:"removable,omitempty"`
	Rotational bool   `json:"rotational,omitempty"`
}

const maxPlatformStateAge = 20 * time.Minute

func writePlatformState(snapshot PlatformState) {
	snapshot.GeneratedAt = actualTimestamp()
	path := platform.CratePath("state", "platform-state.json")
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		log.Printf("warning: marshal platform state: %v", err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("warning: write platform state: %v", err)
	}
}

func writeActorOwnershipState(snapshot ActorOwnershipState) {
	snapshot.GeneratedAt = actualTimestamp()
	path := platform.CratePath("state", "actor-ownership-state.json")
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		log.Printf("warning: marshal actor ownership state: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("warning: create actor ownership state dir: %v", err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("warning: write actor ownership state: %v", err)
	}
}

func LoadActorOwnershipState() ActorOwnershipState {
	path := platform.CratePath("state", "actor-ownership-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return ActorOwnershipState{}
	}
	var snapshot ActorOwnershipState
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return ActorOwnershipState{}
	}
	return snapshot
}

func LoadActorProvisioningState(crateName string) ActorProvisioningState {
	path := filepath.Join(platform.CratePath("services", crateName), "runtime", "actor-provisioning.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return ActorProvisioningState{}
	}
	var snapshot ActorProvisioningState
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return ActorProvisioningState{}
	}
	return snapshot
}

func writeStorageState(snapshot StorageState) {
	snapshot.GeneratedAt = actualTimestamp()
	path := platform.CratePath("state", "storage-state.json")
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		log.Printf("warning: marshal storage state: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("warning: create storage state dir: %v", err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("warning: write storage state: %v", err)
	}
}

func LoadStorageState() StorageState {
	path := platform.CratePath("state", "storage-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return StorageState{}
	}
	var snapshot StorageState
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return StorageState{}
	}
	for i := range snapshot.Devices {
		if strings.TrimSpace(snapshot.Devices[i].Class) == "" {
			snapshot.Devices[i].Class = classifyStorageTarget(snapshot.Devices[i].Mountpoint)
		}
	}
	return snapshot
}

func LoadPlatformState() PlatformState {
	path := platform.CratePath("state", "platform-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return PlatformState{}
	}
	var snapshot PlatformState
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return PlatformState{}
	}
	for i := range snapshot.Adapters {
		if strings.TrimSpace(snapshot.Adapters[i].DisplayName) == "" {
			snapshot.Adapters[i].DisplayName = snapshot.Adapters[i].Name
		}
		if strings.TrimSpace(snapshot.Adapters[i].Status) == "" {
			snapshot.Adapters[i].Status = "unknown"
		}
		if strings.TrimSpace(snapshot.Adapters[i].Health) == "" {
			snapshot.Adapters[i].Health = "unknown"
		}
	}
	applyPlatformStateFreshness(&snapshot, time.Now().UTC())
	return snapshot
}

func platformAdapterState(name, displayName string, enabled bool) PlatformAdapterState {
	state := PlatformAdapterState{
		Name:        name,
		DisplayName: displayName,
		Enabled:     enabled,
		Status:      "disabled",
		Health:      "unknown",
		Validation:  "disabled",
		Apply:       "disabled",
	}
	if enabled {
		state.Status = "pending"
		state.Health = "pending"
		state.Validation = "pending"
		state.Apply = "pending"
	}
	return state
}

func finalizePlatformAdapterState(adapter PlatformAdapterState, issues []string) PlatformAdapterState {
	if len(issues) > 0 {
		adapter.Status = "failed"
		adapter.Health = "degraded"
		adapter.LastError = issues[len(issues)-1]
		if strings.TrimSpace(adapter.Summary) == "" {
			adapter.Summary = strings.Join(issues, "; ")
		}
		return adapter
	}
	if !adapter.Enabled {
		if strings.TrimSpace(adapter.Summary) == "" {
			adapter.Summary = "adapter disabled in desired state"
		}
		return adapter
	}
	if adapter.Status == "" || adapter.Status == "pending" {
		adapter.Status = "ready"
	}
	if adapter.Health == "" || adapter.Health == "pending" {
		adapter.Health = "ok"
	}
	if strings.TrimSpace(adapter.Summary) == "" {
		adapter.Summary = "rendered desired state successfully"
	}
	return adapter
}

func actualTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeStorageDevices(devices []sysinfo.StorageDevice) StorageState {
	state := StorageState{
		Devices:       make([]StorageDeviceState, 0, len(devices)),
		SafeTargets:   []string{},
		SystemTargets: []string{},
	}
	seenSafe := map[string]bool{}
	seenSystem := map[string]bool{}
	for _, dev := range devices {
		device := StorageDeviceState{
			Name:       strings.TrimSpace(dev.Name),
			Path:       strings.TrimSpace(dev.Path),
			Type:       strings.TrimSpace(dev.Type),
			Mountpoint: strings.TrimSpace(dev.Mountpoint),
			FSType:     strings.TrimSpace(dev.FSType),
			Size:       strings.TrimSpace(dev.Size),
			Class:      classifyStorageTarget(dev.Mountpoint),
			Removable:  dev.Removable,
			Rotational: dev.Rotational,
		}
		state.Devices = append(state.Devices, device)
		if strings.TrimSpace(device.Mountpoint) == "" {
			continue
		}
		switch device.Class {
		case "safe":
			if !seenSafe[device.Mountpoint] {
				state.SafeTargets = append(state.SafeTargets, device.Mountpoint)
				seenSafe[device.Mountpoint] = true
			}
		case "system":
			if !seenSystem[device.Mountpoint] {
				state.SystemTargets = append(state.SystemTargets, device.Mountpoint)
				seenSystem[device.Mountpoint] = true
			}
		}
	}
	return state
}

func classifyStorageTarget(mountpoint string) string {
	mountpoint = strings.TrimSpace(mountpoint)
	switch {
	case mountpoint == "":
		return "unmounted"
	case mountpoint == "/" || mountpoint == "/boot" || mountpoint == "/boot/efi":
		return "system"
	case strings.HasPrefix(mountpoint, "/snap"):
		return "system"
	case strings.HasPrefix(mountpoint, "/srv/") || strings.HasPrefix(mountpoint, "/mnt/") || strings.HasPrefix(mountpoint, "/media/"):
		return "safe"
	default:
		return "attached"
	}
}

func applyPlatformStateFreshness(snapshot *PlatformState, now time.Time) {
	generatedAtRaw := strings.TrimSpace(snapshot.GeneratedAt)
	if generatedAtRaw == "" {
		markPlatformStateStale(snapshot, "platform state missing generated_at")
		return
	}
	generatedAt, err := time.Parse(time.RFC3339, generatedAtRaw)
	if err != nil {
		markPlatformStateStale(snapshot, "platform state has invalid generated_at")
		return
	}
	age := now.Sub(generatedAt)
	if age > maxPlatformStateAge {
		markPlatformStateStale(snapshot, fmt.Sprintf("platform state stale: last agent render %s ago", age.Round(time.Second)))
	}
}

func markPlatformStateStale(snapshot *PlatformState, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "platform state stale"
	}
	for i := range snapshot.Adapters {
		snapshot.Adapters[i].Status = "failed"
		snapshot.Adapters[i].Health = "degraded"
		snapshot.Adapters[i].LastError = reason
		if strings.TrimSpace(snapshot.Adapters[i].Summary) == "" || strings.TrimSpace(snapshot.Adapters[i].Summary) == "rendered desired state successfully" {
			snapshot.Adapters[i].Summary = reason
		}
	}
}
