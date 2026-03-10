//go:build !linux
// +build !linux

package users

import (
	"fmt"
	"os/user"

	"github.com/crateos/crateos/internal/config"
)

// SpawnShell is not supported on non-Linux platforms.
func SpawnShell(cfg *config.Config, req ShellAccessRequest) error {
	return fmt.Errorf("break-glass shell is only available on Linux")
}

// CanAccessShell returns false on non-Linux platforms.
func CanAccessShell(cfg *config.Config, username string) bool {
	return false
}

// resolveUserShell is not supported on non-Linux platforms.
func resolveUserShell(usr *user.User) string {
	return ""
}

// spawnInteractiveShell is not supported on non-Linux platforms.
func spawnInteractiveShell(usr *user.User, shell string) int {
	return 1
}

// parseUserIDs is not supported on non-Linux platforms.
func parseUserIDs(usr *user.User) (int, int, error) {
	return 0, 0, fmt.Errorf("user ID parsing not available on non-Linux platforms")
}

// readShellFromPasswd is not supported on non-Linux platforms.
func readShellFromPasswd(username string) string {
	return ""
}

// logShellAccess is not supported on non-Linux platforms.
func logShellAccess(entry ShellAccessLog) error {
	return fmt.Errorf("shell access logging not available on non-Linux platforms")
}

// logAuditEvent is not supported on non-Linux platforms.
func logAuditEvent(eventType string, data map[string]interface{}) error {
	return fmt.Errorf("audit logging not available on non-Linux platforms")
}
