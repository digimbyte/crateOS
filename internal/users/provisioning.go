package users

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

// ProvisioningState tracks the desired and actual user account state.
type ProvisioningState struct {
	GeneratedAt   string                   `json:"generated_at"`
	DesiredUsers  []UserProvisioningRecord `json:"desired_users"`
	ActualUsers   []SystemUserRecord       `json:"actual_users"`
	Reconciled    []ReconciliationRecord   `json:"reconciled"`
	Issues        []string                 `json:"issues"`
	Summary       string                   `json:"summary"`
}

// UserProvisioningRecord represents a desired user from config.
type UserProvisioningRecord struct {
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	UID         int      `json:"uid,omitempty"`
	GID         int      `json:"gid,omitempty"`
	Home        string   `json:"home"`
	Shell       string   `json:"shell"`
	Groups      []string `json:"groups,omitempty"`
	Permissions []string `json:"permissions"`
}

// SystemUserRecord represents an actual system user.
type SystemUserRecord struct {
	Name  string `json:"name"`
	UID   int    `json:"uid"`
	GID   int    `json:"gid"`
	Home  string `json:"home"`
	Shell string `json:"shell"`
}

// ReconciliationRecord tracks a provisioning action taken.
type ReconciliationRecord struct {
	User      string `json:"user"`
	Action    string `json:"action"` // create, update, delete, skip
	Status    string `json:"status"` // success, failed, skipped
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ProvisionUsers reconciles desired users with system accounts.
func ProvisionUsers(cfg *config.Config) ([]ReconciliationRecord, ProvisioningState, error) {
	if runtime.GOOS != "linux" {
		return nil, ProvisioningState{}, fmt.Errorf("user provisioning only supported on Linux")
	}

	state := ProvisioningState{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		DesiredUsers: make([]UserProvisioningRecord, 0),
		ActualUsers:  make([]SystemUserRecord, 0),
		Reconciled:   make([]ReconciliationRecord, 0),
		Issues:       make([]string, 0),
	}

	// Build desired user list from config
	for _, u := range cfg.Users.Users {
		desired := UserProvisioningRecord{
			Name:        strings.TrimSpace(u.Name),
			Role:        strings.TrimSpace(u.Role),
			Home:        userHome(u.Name),
			Shell:       defaultUserShell(u.Role),
			Groups:      []string{userPrimaryGroup(u.Name)},
			Permissions: u.Permissions,
		}
		state.DesiredUsers = append(state.DesiredUsers, desired)
	}

	// Probe actual users on system
	actualUsers := probeSystemUsers()
	state.ActualUsers = actualUsers

	// Reconcile: create/update/delete as needed
	reconciled := reconcileUserAccounts(cfg, state.DesiredUsers, actualUsers)
	state.Reconciled = reconciled

	// Validate results
	state.Issues, state.Summary = validateProvisioningState(reconciled, state.DesiredUsers)

	// Persist state
	if err := saveProvisioningState(state); err != nil {
		state.Issues = append(state.Issues, fmt.Sprintf("failed to save provisioning state: %v", err))
	}

	return reconciled, state, nil
}

// reconcileUserAccounts performs create/update/delete operations.
func reconcileUserAccounts(cfg *config.Config, desired []UserProvisioningRecord, actual []SystemUserRecord) []ReconciliationRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	records := make([]ReconciliationRecord, 0)

	actualMap := make(map[string]SystemUserRecord)
	for _, u := range actual {
		actualMap[u.Name] = u
	}

	// Process desired users
	for _, d := range desired {
		if a, exists := actualMap[d.Name]; exists {
			// User exists: check if update needed
			if needsUpdate(d, a) {
				record := ReconciliationRecord{
					User:      d.Name,
					Action:    "update",
					Status:    "skipped", // non-destructive for now
					Timestamp: now,
				}
				records = append(records, record)
			} else {
				records = append(records, ReconciliationRecord{
					User:      d.Name,
					Action:    "skip",
					Status:    "skipped",
					Timestamp: now,
				})
			}
			delete(actualMap, d.Name)
		} else {
			// User doesn't exist: create
			record := ReconciliationRecord{
				User:      d.Name,
				Action:    "create",
				Timestamp: now,
			}

			if err := createUser(d); err != nil {
				record.Status = "failed"
				record.Error = err.Error()
			} else {
				record.Status = "success"
				bootstrapUserHome(d)
			}
			records = append(records, record)
		}
	}

	// Remaining users in actualMap are extra system users (don't delete automatically)
	for username := range actualMap {
		records = append(records, ReconciliationRecord{
			User:      username,
			Action:    "delete",
			Status:    "skipped",
			Error:     "user not in desired config; manual deletion required",
			Timestamp: now,
		})
	}

	return records
}

// createUser creates a new system user account.
func createUser(desired UserProvisioningRecord) error {
	home := desired.Home
	shell := desired.Shell

	// Ensure home directory parent exists
	if err := os.MkdirAll(filepath.Dir(home), 0755); err != nil {
		return fmt.Errorf("failed to create home parent: %w", err)
	}

	// useradd arguments
	args := []string{
		"-m",                                           // create home directory
		"-d", home,                                     // home directory
		"-s", shell,                                    // login shell
		"-g", userPrimaryGroup(desired.Name),           // primary group
		"-c", fmt.Sprintf("CrateOS operator %s", desired.Name), // comment
	}

	// Add to supplementary groups if specified
	if len(desired.Groups) > 1 {
		args = append(args, "-G", strings.Join(desired.Groups[1:], ","))
	}

	args = append(args, desired.Name)

	cmd := exec.Command("useradd", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("useradd failed: %w", err)
	}

	return nil
}

// bootstrapUserHome configures user home directory.
func bootstrapUserHome(desired UserProvisioningRecord) error {
	home := desired.Home

	// Create standard subdirectories
	subdirs := []string{".ssh", ".config", "projects"}
	for _, subdir := range subdirs {
		path := filepath.Join(home, subdir)
		if err := os.MkdirAll(path, 0700); err != nil {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}
	}

	// Create .ssh/authorized_keys placeholder
	authKeysPath := filepath.Join(home, ".ssh", "authorized_keys")
	if err := os.WriteFile(authKeysPath, []byte{}, 0600); err != nil {
		return fmt.Errorf("failed to create authorized_keys: %w", err)
	}

	// Fix ownership (in case useradd didn't)
	if err := fixUserOwnership(desired.Name, home); err != nil {
		return fmt.Errorf("failed to fix home ownership: %w", err)
	}

	return nil
}

// fixUserOwnership ensures home directory is owned by the correct user.
func fixUserOwnership(username, home string) error {
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", username, userPrimaryGroup(username)), home)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}
	return nil
}

// probeSystemUsers reads actual system users from /etc/passwd.
func probeSystemUsers() []SystemUserRecord {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return []SystemUserRecord{}
	}

	records := make([]SystemUserRecord, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		uid, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			continue
		}
		gid, err := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err != nil {
			continue
		}

		records = append(records, SystemUserRecord{
			Name:  strings.TrimSpace(parts[0]),
			UID:   uid,
			GID:   gid,
			Home:  strings.TrimSpace(parts[5]),
			Shell: strings.TrimSpace(parts[6]),
		})
	}

	return records
}

// needsUpdate checks if a user account needs modification.
func needsUpdate(desired UserProvisioningRecord, actual SystemUserRecord) bool {
	if desired.Home != actual.Home {
		return true
	}
	if desired.Shell != actual.Shell {
		return true
	}
	return false
}

// validateProvisioningState checks for provisioning issues.
func validateProvisioningState(reconciled []ReconciliationRecord, desired []UserProvisioningRecord) ([]string, string) {
	issues := make([]string, 0)
	failedCount := 0
	successCount := 0
	skippedCount := 0

	for _, r := range reconciled {
		switch r.Status {
		case "success":
			successCount++
		case "failed":
			failedCount++
			if r.Error != "" {
				issues = append(issues, fmt.Sprintf("user %s: %s", r.User, r.Error))
			}
		case "skipped":
			skippedCount++
		}
	}

	summary := fmt.Sprintf("provisioned %d users (%d created, %d skipped)", len(desired), successCount, skippedCount)
	if failedCount > 0 {
		summary = fmt.Sprintf("%s; %d failed", summary, failedCount)
		issues = append(issues, fmt.Sprintf("%d user provisioning failures detected", failedCount))
	}

	return issues, summary
}

// userHome returns the home directory for a user.
func userHome(name string) string {
	return filepath.Join("/home", strings.ToLower(strings.TrimSpace(name)))
}

// userPrimaryGroup returns the primary group name for a user.
func userPrimaryGroup(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// defaultUserShell determines the shell based on role.
func defaultUserShell(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	// All CrateOS users use the crateos console as their shell
	// unless they have break-glass access
	return "/usr/local/bin/crateos-shell-wrapper"
}

// saveProvisioningState persists the provisioning state.
func saveProvisioningState(state ProvisioningState) error {
	path := platform.CratePath("state", "user-provisioning.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// LoadProvisioningState loads the saved provisioning state.
func LoadProvisioningState() (ProvisioningState, error) {
	path := platform.CratePath("state", "user-provisioning.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProvisioningState{}, nil
		}
		return ProvisioningState{}, err
	}

	var state ProvisioningState
	if err := json.Unmarshal(data, &state); err != nil {
		return ProvisioningState{}, err
	}

	return state, nil
}
