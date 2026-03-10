package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

func writeManagedActorOwnershipState(index managedActorOwnershipIndex) {
	snapshot := ActorOwnershipState{
		Claims: make([]ActorOwnershipStateItem, 0, len(index.Actors)),
	}
	for _, actor := range index.Actors {
		item := ActorOwnershipStateItem{
			Crate:     actor.Crate,
			Name:      actor.Name,
			Type:      actor.Type,
			ID:        actor.ID,
			User:      actor.User,
			Group:     actor.Group,
			Home:      actor.Home,
			UpdatedAt: actor.UpdatedAt,
			RetiredAt: actor.RetiredAt,
			Status:    "retired",
		}
		if actor.Active {
			item.Status = "claimed"
			snapshot.Active++
		} else {
			snapshot.Retired++
		}
		snapshot.Claims = append(snapshot.Claims, item)
	}
	writeActorOwnershipState(snapshot)
}

func managedActorIDSeed(crateName, actorName string, ordinal int) string {
	base := strings.TrimSpace(actorName)
	if base == "" {
		base = strings.TrimSpace(crateName)
	}
	seed := fmt.Sprintf("%s|%s|%d", strings.TrimSpace(crateName), base, ordinal)
	sum := sha256.Sum256([]byte(seed))
	hexID := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexID[0:8], hexID[8:12], hexID[12:16], hexID[16:20], hexID[20:32])
}

func managedActorAccountName(actorID string, ordinal int) string {
	compact := strings.ReplaceAll(strings.TrimSpace(actorID), "-", "")
	if compact == "" {
		return ""
	}
	suffix := ""
	if ordinal > 0 {
		suffix = fmt.Sprintf("-%d", ordinal)
	}
	maxCompact := 32 - len("ca-") - len(suffix)
	if maxCompact > len(compact) {
		maxCompact = len(compact)
	}
	if maxCompact < 8 {
		maxCompact = 8
	}
	return "ca-" + compact[:maxCompact] + suffix
}

type managedActorRegistry struct {
	GeneratedAt string                   `json:"generated_at"`
	Actor       managedActorRegistryItem `json:"actor"`
}

type managedActorRegistryItem struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	ID        string `json:"id,omitempty"`
	User      string `json:"user,omitempty"`
	Group     string `json:"group,omitempty"`
	Home      string `json:"home,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type managedActorProvisioningState struct {
	GeneratedAt   string                          `json:"generated_at"`
	LastSuccessAt string                          `json:"last_success_at,omitempty"`
	LastFailureAt string                          `json:"last_failure_at,omitempty"`
	Events        []managedActorProvisioningEvent `json:"events,omitempty"`
	Actor         struct {
		Name       string `json:"name,omitempty"`
		Type       string `json:"type,omitempty"`
		ID         string `json:"id,omitempty"`
		User       string `json:"user,omitempty"`
		Group      string `json:"group,omitempty"`
		Home       string `json:"home,omitempty"`
		RuntimeDir string `json:"runtime_dir,omitempty"`
		StateDir   string `json:"state_dir,omitempty"`
	} `json:"actor"`
	Provisioning string `json:"provisioning,omitempty"`
	Error        string `json:"error,omitempty"`
}

type managedActorProvisioningEvent struct {
	At           string `json:"at"`
	Provisioning string `json:"provisioning,omitempty"`
	Error        string `json:"error,omitempty"`
}

type managedActorOwnershipIndex struct {
	GeneratedAt string                      `json:"generated_at"`
	Actors      []managedActorOwnershipItem `json:"actors,omitempty"`
}

type managedActorOwnershipItem struct {
	Crate     string `json:"crate"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	ID        string `json:"id"`
	User      string `json:"user"`
	Group     string `json:"group,omitempty"`
	Home      string `json:"home,omitempty"`
	Active    bool   `json:"active"`
	UpdatedAt string `json:"updated_at,omitempty"`
	RetiredAt string `json:"retired_at,omitempty"`
}

func managedActorRegistryPath(crateName string) string {
	return filepath.Join(platform.CratePath("services", crateName), "runtime", "actor-registry.json")
}

func managedActorOwnershipIndexPath() string {
	return platform.CratePath("registry", "managed-actors.json")
}

func managedActorProvisioningStatePath(crateName string) string {
	return filepath.Join(platform.CratePath("services", crateName), "runtime", "actor-provisioning.json")
}

const managedActorTombstoneRetention = 30 * 24 * time.Hour

func appendManagedActorProvisioningEvent(events []managedActorProvisioningEvent, event managedActorProvisioningEvent, limit int) []managedActorProvisioningEvent {
	if limit <= 0 {
		limit = 10
	}
	if strings.TrimSpace(event.At) == "" {
		event.At = actualTimestamp()
	}
	if len(events) > 0 {
		last := events[len(events)-1]
		if last.Provisioning == event.Provisioning && strings.TrimSpace(last.Error) == strings.TrimSpace(event.Error) {
			events[len(events)-1] = event
			return events
		}
	}
	events = append(events, event)
	if len(events) > limit {
		events = append([]managedActorProvisioningEvent(nil), events[len(events)-limit:]...)
	}
	return events
}

func loadManagedActorOwnershipIndex() managedActorOwnershipIndex {
	path := managedActorOwnershipIndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return managedActorOwnershipIndex{Actors: []managedActorOwnershipItem{}}
	}
	var index managedActorOwnershipIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return managedActorOwnershipIndex{Actors: []managedActorOwnershipItem{}}
	}
	if index.Actors == nil {
		index.Actors = []managedActorOwnershipItem{}
	}
	return index
}

func pruneExpiredManagedActorTombstones(index managedActorOwnershipIndex, now time.Time) managedActorOwnershipIndex {
	if len(index.Actors) == 0 {
		return index
	}
	filtered := index.Actors[:0]
	for _, actor := range index.Actors {
		if actor.Active {
			filtered = append(filtered, actor)
			continue
		}
		retiredAt := strings.TrimSpace(actor.RetiredAt)
		if retiredAt == "" {
			filtered = append(filtered, actor)
			continue
		}
		ts, err := time.Parse(time.RFC3339, retiredAt)
		if err != nil || now.Sub(ts) <= managedActorTombstoneRetention {
			filtered = append(filtered, actor)
		}
	}
	index.Actors = filtered
	return index
}

func writeManagedActorOwnershipIndex(index managedActorOwnershipIndex) error {
	path := managedActorOwnershipIndexPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	index = pruneExpiredManagedActorTombstones(index, time.Now().UTC())
	index.GeneratedAt = actualTimestamp()
	if index.Actors == nil {
		index.Actors = []managedActorOwnershipItem{}
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func writeManagedActorOwnership(crate CrateState) error {
	index := loadManagedActorOwnershipIndex()
	if strings.TrimSpace(crate.Name) == "" || strings.TrimSpace(crate.ActorID) == "" || strings.TrimSpace(crate.ActorUser) == "" || strings.TrimSpace(crate.ExecutionMode) == "" {
		for i := range index.Actors {
			if strings.TrimSpace(index.Actors[i].Crate) != strings.TrimSpace(crate.Name) {
				continue
			}
			index.Actors[i].Active = false
			index.Actors[i].UpdatedAt = actualTimestamp()
			if strings.TrimSpace(index.Actors[i].RetiredAt) == "" {
				index.Actors[i].RetiredAt = actualTimestamp()
			}
		}
		return writeManagedActorOwnershipIndex(index)
	}
	entry := managedActorOwnershipItem{
		Crate:     crate.Name,
		Name:      crate.ActorName,
		Type:      crate.ActorType,
		ID:        crate.ActorID,
		User:      crate.ActorUser,
		Group:     crate.ActorGroup,
		Home:      crate.ActorHome,
		Active:    true,
		UpdatedAt: actualTimestamp(),
	}
	replaced := false
	for i := range index.Actors {
		if strings.TrimSpace(index.Actors[i].Crate) != strings.TrimSpace(crate.Name) {
			continue
		}
		index.Actors[i] = entry
		replaced = true
		break
	}
	if !replaced {
		index.Actors = append(index.Actors, entry)
	}
	if err := writeManagedActorOwnershipIndex(index); err != nil {
		return err
	}
	writeManagedActorOwnershipState(index)
	return nil
}

func pruneManagedActorOwnership(cfg *config.Config) error {
	index := loadManagedActorOwnershipIndex()
	if len(index.Actors) == 0 {
		if err := writeManagedActorOwnershipIndex(index); err != nil {
			return err
		}
		writeManagedActorOwnershipState(index)
		return nil
	}
	activeCrates := map[string]config.ServiceEntry{}
	for _, svc := range cfg.Services.Services {
		activeCrates[strings.TrimSpace(svc.Name)] = svc
	}
	for i := range index.Actors {
		name := strings.TrimSpace(index.Actors[i].Crate)
		if name == "" {
			continue
		}
		svc, ok := activeCrates[name]
		if !ok {
			index.Actors[i].Active = false
			if strings.TrimSpace(index.Actors[i].RetiredAt) == "" {
				index.Actors[i].RetiredAt = actualTimestamp()
			}
			continue
		}
		if strings.TrimSpace(svc.Actor.Name) == "" {
			index.Actors[i].Active = false
			if strings.TrimSpace(index.Actors[i].RetiredAt) == "" {
				index.Actors[i].RetiredAt = actualTimestamp()
			}
			continue
		}
		index.Actors[i].Active = true
		index.Actors[i].RetiredAt = ""
	}
	if err := writeManagedActorOwnershipIndex(index); err != nil {
		return err
	}
	writeManagedActorOwnershipState(index)
	return nil
}

func writeManagedActorProvisioningState(crate CrateState) error {
	if strings.TrimSpace(crate.Name) == "" {
		return nil
	}
	path := managedActorProvisioningStatePath(crate.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var previous managedActorProvisioningState
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &previous)
	}
	payload := managedActorProvisioningState{
		GeneratedAt:   actualTimestamp(),
		LastSuccessAt: previous.LastSuccessAt,
		LastFailureAt: previous.LastFailureAt,
		Events:        append([]managedActorProvisioningEvent(nil), previous.Events...),
		Provisioning:  crate.ActorProvisioning,
		Error:         crate.ActorProvisioningError,
	}
	switch crate.ActorProvisioning {
	case "provisioned":
		payload.LastSuccessAt = payload.GeneratedAt
	case "blocked":
		payload.LastFailureAt = payload.GeneratedAt
	}
	payload.Events = appendManagedActorProvisioningEvent(payload.Events, managedActorProvisioningEvent{
		At:           payload.GeneratedAt,
		Provisioning: crate.ActorProvisioning,
		Error:        crate.ActorProvisioningError,
	}, 10)
	payload.Actor.Name = crate.ActorName
	payload.Actor.Type = crate.ActorType
	payload.Actor.ID = crate.ActorID
	payload.Actor.User = crate.ActorUser
	payload.Actor.Group = crate.ActorGroup
	payload.Actor.Home = crate.ActorHome
	payload.Actor.RuntimeDir = crate.ActorRuntimeDir
	payload.Actor.StateDir = crate.ActorStateDir
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func managedActorClaimedByAnotherCrate(index managedActorOwnershipIndex, crateName, actorID, account string) bool {
	for _, actor := range index.Actors {
		if strings.TrimSpace(actor.Crate) == strings.TrimSpace(crateName) {
			continue
		}
		if !actor.Active {
			continue
		}
		if actorID != "" && strings.TrimSpace(actor.ID) == strings.TrimSpace(actorID) {
			return true
		}
		if account != "" && strings.TrimSpace(actor.User) == strings.TrimSpace(account) {
			return true
		}
	}
	return false
}

func managedActorOwnershipRecord(crateName string) (managedActorOwnershipItem, bool) {
	index := loadManagedActorOwnershipIndex()
	for _, actor := range index.Actors {
		if strings.TrimSpace(actor.Crate) != strings.TrimSpace(crateName) {
			continue
		}
		return actor, true
	}
	return managedActorOwnershipItem{}, false
}

func loadManagedActorRegistry(crateName string) (managedActorRegistryItem, bool) {
	path := managedActorRegistryPath(crateName)
	data, err := os.ReadFile(path)
	if err != nil {
		return managedActorRegistryItem{}, false
	}
	var registry managedActorRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return managedActorRegistryItem{}, false
	}
	if strings.TrimSpace(registry.Actor.ID) == "" || strings.TrimSpace(registry.Actor.User) == "" {
		return managedActorRegistryItem{}, false
	}
	return registry.Actor, true
}

func writeManagedActorRegistry(crate CrateState) error {
	if strings.TrimSpace(crate.Name) == "" || strings.TrimSpace(crate.ActorID) == "" || strings.TrimSpace(crate.ActorUser) == "" {
		return nil
	}
	path := managedActorRegistryPath(crate.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	payload := managedActorRegistry{
		GeneratedAt: actualTimestamp(),
		Actor: managedActorRegistryItem{
			Name:      crate.ActorName,
			Type:      crate.ActorType,
			ID:        crate.ActorID,
			User:      crate.ActorUser,
			Group:     crate.ActorGroup,
			Home:      crate.ActorHome,
			UpdatedAt: actualTimestamp(),
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func resolveManagedActorIdentity(crateName, actorName string) (string, string, string) {
	if registry, ok := loadManagedActorRegistry(crateName); ok {
		return registry.ID, registry.User, valueOrDefault(strings.TrimSpace(registry.Group), registry.User)
	}
	index := loadManagedActorOwnershipIndex()
	seen := map[string]struct{}{}
	for ordinal := 0; ordinal < 16; ordinal++ {
		actorID := managedActorIDSeed(crateName, actorName, ordinal)
		account := managedActorAccountName(actorID, ordinal)
		if account == "" {
			continue
		}
		if _, exists := seen[account]; exists {
			continue
		}
		if managedActorClaimedByAnotherCrate(index, crateName, actorID, account) {
			continue
		}
		seen[account] = struct{}{}
		return actorID, account, account
	}
	return "", "", ""
}

func protectedManagedActorNames() map[string]struct{} {
	protected := map[string]struct{}{
		"root": {}, "admin": {}, "ubuntu": {}, "nobody": {}, "daemon": {},
		"www-data": {}, "crateos-agent": {}, "crateos-policy": {}, "crateos": {},
	}
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		for _, user := range cfg.Users.Users {
			name := strings.ToLower(strings.TrimSpace(user.Name))
			if name != "" {
				protected[name] = struct{}{}
			}
		}
	}
	return protected
}

func managedActorIdentityIssue(crate CrateState) string {
	if crate.ExecutionMode == "" || crate.Module {
		return ""
	}
	if strings.TrimSpace(crate.ActorName) == "" {
		return "managed workload requires actor.name"
	}
	if strings.TrimSpace(crate.ActorID) == "" || strings.TrimSpace(crate.ActorUser) == "" {
		return "managed actor identity could not be allocated"
	}
	if !strings.HasPrefix(strings.TrimSpace(crate.ActorUser), "ca-") {
		return "managed actor runtime account escaped the CrateOS-owned namespace"
	}
	protected := protectedManagedActorNames()
	if _, blocked := protected[strings.ToLower(strings.TrimSpace(crate.ActorUser))]; blocked {
		return "managed actor runtime account collides with a protected identity"
	}
	if _, blocked := protected[strings.ToLower(strings.TrimSpace(crate.ActorGroup))]; blocked {
		return "managed actor runtime group collides with a protected identity"
	}
	return ""
}

func applyManagedActorProvisioningPosture(crate *CrateState) {
	crate.ActorProvisioning = ""
	crate.ActorProvisioningError = ""
	crate.ActorProvisioningUpdatedAt = actualTimestamp()
	crate.ActorProvisioningStatePath = ""
	crate.ActorOwnershipStatus = ""
	crate.ActorOwnershipUpdatedAt = ""
	crate.ActorOwnershipRetiredAt = ""
	if crate.ExecutionMode == "" || crate.Module {
		return
	}
	crate.ActorProvisioningStatePath = managedActorProvisioningStatePath(crate.Name)
	if record, ok := managedActorOwnershipRecord(crate.Name); ok {
		if record.Active {
			crate.ActorOwnershipStatus = "claimed"
		} else {
			crate.ActorOwnershipStatus = "retired"
		}
		crate.ActorOwnershipUpdatedAt = record.UpdatedAt
		crate.ActorOwnershipRetiredAt = record.RetiredAt
	}
	if issue := managedActorIdentityIssue(*crate); issue != "" {
		crate.ActorProvisioning = "blocked"
		crate.ActorProvisioningError = issue
		return
	}
	if runtime.GOOS != "linux" {
		crate.ActorProvisioning = "pending"
		return
	}
	if issue := managedActorAdoptionIssue(*crate); issue != "" {
		crate.ActorProvisioning = "blocked"
		crate.ActorProvisioningError = issue
		return
	}
	if linuxAccountExists(crate.ActorUser) && linuxGroupExists(crate.ActorGroup) && pathExists(crate.ActorHome) && pathExists(crate.ActorRuntimeDir) && pathExists(crate.ActorStateDir) {
		crate.ActorProvisioning = "provisioned"
		return
	}
	crate.ActorProvisioning = "pending"
}
