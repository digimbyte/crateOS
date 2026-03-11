//go:build linux
// +build linux

package users

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/crateos/crateos/internal/auth"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

// CanAccessShell checks if a user is authorized for break-glass access.
func CanAccessShell(cfg *config.Config, username string) bool {
	if runtime.GOOS != "linux" {
		return false
	}

	if !cfg.CrateOS.Access.BreakGlass.Enabled {
		return false
	}

	authz := auth.Load(cfg)
	if authz == nil {
		return false
	}

	permRequired := strings.TrimSpace(cfg.CrateOS.Access.BreakGlass.RequirePerm)
	if permRequired == "" {
		return false
	}

	return authz.Check(username, permRequired)
}

// SpawnShell spawns an interactive shell for the user.
func SpawnShell(cfg *config.Config, req ShellAccessRequest) error {
	username := strings.TrimSpace(req.User)

	// Check authorization
	if !CanAccessShell(cfg, username) {
		logShellAccess(ShellAccessLog{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User:      username,
			Result:    "denied",
			Reason:    "no break-glass permission",
			SourceIP:  req.SourceIP,
			SessionID: req.SessionID,
		})
		return fmt.Errorf("user %s does not have break-glass access", username)
	}

	// Resolve user info
	usr, err := user.Lookup(username)
	if err != nil {
		logShellAccess(ShellAccessLog{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User:      username,
			Result:    "denied",
			Reason:    fmt.Sprintf("user lookup failed: %v", err),
			SourceIP:  req.SourceIP,
			SessionID: req.SessionID,
		})
		return fmt.Errorf("user lookup failed: %w", err)
	}

	// Resolve shell
	shell := resolveUserShell(usr)

	// Log access
	logShellAccess(ShellAccessLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      username,
		Result:    "allowed",
		Reason:    req.Reason,
		SourceIP:  req.SourceIP,
		SessionID: req.SessionID,
	})

	// Spawn shell
	startTime := time.Now()
	exitCode := spawnInteractiveShell(usr, shell)

	logShellAccess(ShellAccessLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      username,
		Result:    "allowed",
		Reason:    "shell session ended",
		SourceIP:  req.SourceIP,
		SessionID: req.SessionID,
		ExitCode:  exitCode,
		Duration:  time.Since(startTime).String(),
	})

	return nil
}

// resolveUserShell determines which shell to use.
func resolveUserShell(usr *user.User) string {
	// Try to read shell from /etc/passwd
	// Note: user.User doesn't expose Shell field, so we read from /etc/passwd
	shell := readShellFromPasswd(usr.Username)
	if shell == "/usr/local/bin/crateos-shell-wrapper" || shell == "/usr/local/bin/crateos-login-shell" {
		shell = ""
	}
	if shell != "" && shell != "/usr/sbin/nologin" && shell != "/bin/false" {
		return shell
	}

	// Fall back to bash
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash"
	}

	// Fall back to sh
	return "/bin/sh"
}

// readShellFromPasswd reads the shell from /etc/passwd for a user.
func readShellFromPasswd(username string) string {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= 7 && fields[0] == username {
			return strings.TrimSpace(fields[6])
		}
	}
	return ""
}

// spawnInteractiveShell executes a shell as the user.
func spawnInteractiveShell(usr *user.User, shell string) int {
	cmd := exec.Command(shell)

	// Set up environment
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("HOME=%s", usr.HomeDir))
	env = append(env, fmt.Sprintf("USER=%s", usr.Username))
	env = append(env, fmt.Sprintf("LOGNAME=%s", usr.Username))
	if uid, gid, err := parseUserIDs(usr); err == nil {
		env = append(env, fmt.Sprintf("UID=%d", uid))
		env = append(env, fmt.Sprintf("GID=%d", gid))
	}
	cmd.Env = env

	// Change to home directory
	if err := os.Chdir(usr.HomeDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not change to home directory: %v\n", err)
	}

	// Set up credentials
	uid, gid, err := parseUserIDs(usr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse user credentials: %v\n", err)
		// Try to run anyway without credentials
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	// Run shell
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}

	return 0
}

// parseUserIDs extracts UID and GID from user.User.
func parseUserIDs(usr *user.User) (int, int, error) {
	var uid, gid int
	_, err := fmt.Sscanf(usr.Uid, "%d", &uid)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UID: %w", err)
	}

	_, err = fmt.Sscanf(usr.Gid, "%d", &gid)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid GID: %w", err)
	}

	return uid, gid, nil
}

// logShellAccess writes a shell access event to the audit log.
func logShellAccess(entry ShellAccessLog) error {
	return logAuditEvent("shell", map[string]interface{}{
		"timestamp": entry.Timestamp,
		"user":      entry.User,
		"result":    entry.Result,
		"reason":    entry.Reason,
		"source_ip": entry.SourceIP,
		"session_id": entry.SessionID,
		"exit_code": entry.ExitCode,
		"duration":  entry.Duration,
	})
}

// logAuditEvent logs a general audit event.
func logAuditEvent(eventType string, data map[string]interface{}) error {
	logDir := platform.CratePath("logs", "audit")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Use date-based log files
	now := time.Now().UTC()
	logFile := fmt.Sprintf("%s/%s-%04d%02d%02d.jsonl", 
		logDir, eventType, now.Year(), now.Month(), now.Day())

	// Add timestamp if not present
	if data["timestamp"] == nil {
		data["timestamp"] = now.Format(time.RFC3339)
	}
	data["event_type"] = eventType

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(dataBytes) + "\n")
	return err
}
