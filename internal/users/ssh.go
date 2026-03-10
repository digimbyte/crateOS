package users

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/auth"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

// SSHAuthRequest represents an SSH authentication request.
type SSHAuthRequest struct {
	User     string `json:"user"`
	Method   string `json:"method"` // publickey, password, etc.
	Key      string `json:"key,omitempty"`
	Password string `json:"password,omitempty"`
}

// SSHAuthResponse represents the auth result.
type SSHAuthResponse struct {
	Allowed     bool     `json:"allowed"`
	User        string   `json:"user"`
	Permissions []string `json:"permissions"`
	Home        string   `json:"home"`
	Error       string   `json:"error,omitempty"`
}

// AuthorizedKey represents a stored SSH public key.
type AuthorizedKey struct {
	Username  string    `json:"username"`
	Fingerprint string  `json:"fingerprint"`
	Key       string    `json:"key"`
	Comment   string    `json:"comment,omitempty"`
	AddedAt   string    `json:"added_at"`
	ValidFrom string    `json:"valid_from,omitempty"`
	ValidUntil string   `json:"valid_until,omitempty"`
}

// SSHAuthLog tracks authentication attempts for audit.
type SSHAuthLog struct {
	Timestamp  string `json:"timestamp"`
	User       string `json:"user"`
	Method     string `json:"method"`
	SourceIP   string `json:"source_ip,omitempty"`
	Result     string `json:"result"` // success, failed, denied
	Reason     string `json:"reason,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// ValidateSSHAuth checks if an SSH login attempt is authorized.
func ValidateSSHAuth(cfg *config.Config, req SSHAuthRequest) SSHAuthResponse {
	resp := SSHAuthResponse{
		User: strings.TrimSpace(req.User),
	}

	// Check if user exists in config
	var userEntry *config.UserEntry
	for i := range cfg.Users.Users {
		if cfg.Users.Users[i].Name == resp.User {
			userEntry = &cfg.Users.Users[i]
			break
		}
	}

	if userEntry == nil {
		resp.Allowed = false
		resp.Error = "user not found in CrateOS config"
		logSSHAuth(SSHAuthLog{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User:      resp.User,
			Method:    req.Method,
			Result:    "denied",
			Reason:    "user not found",
		})
		return resp
	}

	// Check authentication method
	switch req.Method {
	case "publickey":
		if validatePublicKey(resp.User, req.Key) {
			resp.Allowed = true
		} else {
			resp.Allowed = false
			resp.Error = "key not authorized"
		}
	case "password":
		resp.Allowed = false
		resp.Error = "password auth disabled; use SSH keys"
	default:
		resp.Allowed = false
		resp.Error = "unsupported auth method"
	}

	if !resp.Allowed {
		logSSHAuth(SSHAuthLog{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User:      resp.User,
			Method:    req.Method,
			Result:    "failed",
			Reason:    resp.Error,
		})
		return resp
	}

	// Authorization successful: populate permissions
	authz := auth.Load(cfg)
	resp.Home = userHome(resp.User)
	if authz != nil {
		if user, ok := authz.Users[resp.User]; ok {
			for perm := range user.Allow {
				resp.Permissions = append(resp.Permissions, perm)
			}
		}
	}

	logSSHAuth(SSHAuthLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      resp.User,
		Method:    req.Method,
		Result:    "success",
	})

	return resp
}

// validatePublicKey checks if a public key is authorized for the user.
func validatePublicKey(username, key string) bool {
	authKeysPath := filepath.Join(userHome(username), ".ssh", "authorized_keys")
	data, err := os.ReadFile(authKeysPath)
	if err != nil {
		return false
	}

	keyNorm := strings.TrimSpace(key)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == keyNorm {
			return true
		}
	}

	return false
}

// AddSSHKey adds a public key to a user's authorized_keys.
func AddSSHKey(username, key, comment string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("public key cannot be empty")
	}

	authKeysPath := filepath.Join(userHome(username), ".ssh", "authorized_keys")

	// Read existing keys
	var existingKeys []string
	if data, err := os.ReadFile(authKeysPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				existingKeys = append(existingKeys, line)
			}
		}
	}

	// Check for duplicate
	for _, existing := range existingKeys {
		if existing == key {
			return fmt.Errorf("key already authorized")
		}
	}

	// Append new key
	existingKeys = append(existingKeys, key)
	var buf strings.Builder
	for _, k := range existingKeys {
		buf.WriteString(k + "\n")
	}

	if err := os.WriteFile(authKeysPath, []byte(buf.String()), 0600); err != nil {
		return fmt.Errorf("failed to update authorized_keys: %w", err)
	}

	// Log the change
	logSSHAuth(SSHAuthLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      username,
		Method:    "key_add",
		Result:    "success",
		Reason:    comment,
	})

	return nil
}

// RemoveSSHKey removes a public key from a user's authorized_keys.
func RemoveSSHKey(username, key string) error {
	key = strings.TrimSpace(key)
	authKeysPath := filepath.Join(userHome(username), ".ssh", "authorized_keys")

	// Read existing keys
	data, err := os.ReadFile(authKeysPath)
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	var filteredKeys []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line != key {
			filteredKeys = append(filteredKeys, line)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("key not found in authorized_keys")
	}

	var buf strings.Builder
	for _, k := range filteredKeys {
		buf.WriteString(k + "\n")
	}

	if err := os.WriteFile(authKeysPath, []byte(buf.String()), 0600); err != nil {
		return fmt.Errorf("failed to update authorized_keys: %w", err)
	}

	return nil
}

// ListSSHKeys returns all authorized keys for a user.
func ListSSHKeys(username string) ([]AuthorizedKey, error) {
	authKeysPath := filepath.Join(userHome(username), ".ssh", "authorized_keys")
	data, err := os.ReadFile(authKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuthorizedKey{}, nil
		}
		return nil, err
	}

	var keys []AuthorizedKey
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Basic parsing: "ssh-rsa AAAA... comment"
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}

		key := AuthorizedKey{
			Username: username,
			Key:      line,
			AddedAt:  time.Now().UTC().Format(time.RFC3339),
		}

		if len(parts) >= 3 {
			key.Comment = parts[2]
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// logSSHAuth writes an authentication attempt to the audit log.
func logSSHAuth(entry SSHAuthLog) error {
	logDir := platform.CratePath("logs", "ssh")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, "auth.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// LoadSSHAuthLog returns recent SSH authentication attempts.
func LoadSSHAuthLog(limit int) ([]SSHAuthLog, error) {
	logFile := platform.CratePath("logs", "ssh", "auth.jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHAuthLog{}, nil
		}
		return nil, err
	}

	var entries []SSHAuthLog
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry SSHAuthLog
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	// Return last N entries
	if len(entries) > limit && limit > 0 {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}
