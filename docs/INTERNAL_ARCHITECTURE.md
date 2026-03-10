# CrateOS Internal Architecture - User & Virtualization Systems

## Overview

This document describes the internal architecture of CrateOS user management and virtual desktop systems. These are **internal systems** - no HTTP APIs are exposed. All interaction happens through direct Go function calls within the agent and TUI.

## Architecture

```
Configuration (users.yaml, crateos.yaml)
    ↓
CrateOS Agent (Reconciliation Loop)
    ├─ users.ProvisionUsers()        → System account sync (/etc/passwd)
    ├─ virtualization.ReconcileVirtualDesktop()  → Session state tracking
    ├─ state.ReconcilePlatform()     → Network, firewall, etc.
    └─ state.Probe()                 → Actual system state
    ↓
TUI Console (crateos console)
    └─ Direct Go function calls
       ├─ users.AddSSHKey()
       ├─ users.SpawnShell()
       ├─ virtualization.StartUserSession()
       └─ etc.
```

## Component Architecture

### 1. User Management (`internal/users/`)

**Provisioning (`provisioning.go`):**
- Reads desired users from `cfg.Users.Users`
- Probes `/etc/passwd` for actual state
- Reconciles: creates/updates/deletes system accounts via `useradd`
- Bootstraps home directories with `.ssh/authorized_keys`
- Persists state to `state/user-provisioning.json`

**SSH Management (`ssh.go`):**
- Validates SSH authentication against CrateOS users
- Manages SSH public keys in `~/.ssh/authorized_keys`
- Provides functions: `AddSSHKey()`, `RemoveSSHKey()`, `ListSSHKeys()`
- Logs auth attempts to `/srv/crateos/logs/ssh/auth.jsonl`

**Shell Access (`shell.go`):**
- Permission-gated break-glass shell access
- `CanAccessShell()` - checks `shell.breakglass` permission
- `SpawnShell()` - spawns interactive shell with UID/GID dropped
- Logs shell sessions with duration/exit code to audit log

### 2. Virtual Desktop (`internal/virtualization/`)

**Session Management (`sessions.go`):**
- Manages user desktop sessions (VNC, X11, Wayland)
- `StartSession()` - allocates display, port, starts compositor
- `StopSession()` - kills process, updates state
- `ListUserSessions()` - returns user's active sessions
- Persists session state to `state/virtualization/*.json`

**Reconciliation (`reconcile.go`):**
- `ReconcileVirtualDesktop()` - validates config, loads sessions
- `ValidateVirtualDesktopConfig()` - checks config constraints
- Helper functions: `StartUserSession()`, `StopUserSession()`, `ListUserSessions()`

### 3. Agent Integration (`internal/state/`)

**Engine (`engine.go`):**
- Main reconciliation loop in `Apply(cfg)`
- Calls `users.ProvisionUsers(cfg)` during reconciliation
- Logs provisioning actions to the action stream
- Linux-only (returns gracefully on non-Linux)

**Platform Reconciliation (`platform_reconcile.go`):**
- Imports virtualization package
- Calls `virtualization.ReconcileVirtualDesktop(cfg)`
- Includes virtual desktop validation errors in action stream
- Renders virtual desktop state diagnostics

### 4. TUI Integration (`internal/tui/`)

The TUI will import user/virtualization packages directly:

```go
import (
    "github.com/crateos/crateos/internal/users"
    "github.com/crateos/crateos/internal/virtualization"
)

// In menu handler:
if userPressedBreakGlass {
    req := users.ShellAccessRequest{
        User: currentUser,
        Reason: "debugging",
    }
    users.SpawnShell(cfg, req)
}

// For adding SSH key:
users.AddSSHKey(targetUser, keyContent, comment)

// For starting desktop session:
session, err := virtualization.StartUserSession(user, "vnc", "workspace")
```

## Data Flow

### User Provisioning Flow

```
Agent Reconciliation
    ↓
users.ProvisionUsers(cfg)
    ├─ Read: cfg.Users.Users
    ├─ Probe: /etc/passwd
    ├─ Create: useradd for new users
    ├─ Bootstrap: /home/<user> with .ssh/authorized_keys
    ├─ Persist: state/user-provisioning.json
    └─ Return: []ReconciliationRecord, ProvisioningState
    ↓
state.Apply() logs actions
```

### SSH Key Management Flow

```
TUI Menu
    ↓
User selects "Add SSH Key"
    ↓
users.AddSSHKey(username, publicKeyContent, comment)
    ├─ Read: ~username/.ssh/authorized_keys
    ├─ Append: new key
    ├─ Write: updated authorized_keys (0600)
    ├─ Log: SSH auth log
    └─ Return: error or nil
```

### Virtual Desktop Session Flow

```
TUI Menu / External Request
    ↓
virtualization.StartUserSession(user, sessionType, landing)
    ├─ Allocate: display (:10+) and port (5900+)
    ├─ Start: Xvfb with allocated display
    ├─ Start: Window manager (XFCE4)
    ├─ Persist: state/virtualization/<session-id>.json
    └─ Return: UserSession
    ↓
User connects via VNC (SSH tunnel recommended)
    ↓
virtualization.StopSession(sessionID)
    ├─ Kill: process
    ├─ Update: session state
    └─ Persist: updated state
```

## Function Signatures

### User Management

```go
// Provision all users in config
func ProvisionUsers(cfg *config.Config) (
    []ReconciliationRecord,  // Actions taken
    ProvisioningState,       // Full state snapshot
    error,
)

// SSH key operations
func AddSSHKey(username, key, comment string) error
func RemoveSSHKey(username, key string) error
func ListSSHKeys(username string) ([]AuthorizedKey, error)

// SSH auth validation
func ValidateSSHAuth(cfg *config.Config, req SSHAuthRequest) SSHAuthResponse

// Shell access
func CanAccessShell(cfg *config.Config, username string) bool
func SpawnShell(cfg *config.Config, req ShellAccessRequest) error
```

### Virtual Desktop

```go
// Session management
func (sm *SessionManager) StartSession(username, sessionType, landing string) (*UserSession, error)
func (sm *SessionManager) StopSession(sessionID string) error
func (sm *SessionManager) GetSession(sessionID string) (*UserSession, error)
func (sm *SessionManager) ListUserSessions(username string) []*UserSession
func (sm *SessionManager) ListAllSessions() []*UserSession

// Reconciliation
func ReconcileVirtualDesktop(cfg *config.Config) VirtualDesktopState
func ValidateVirtualDesktopConfig(cfg config.CrateOSConfig) []string

// Convenience functions
func StartUserSession(username, sessionType, landing string) SessionResponse
func StopUserSession(sessionID string) SessionResponse
func ListUserSessions(username string) []VirtualDesktopSessionSummary
func GetSessionInfo(sessionID string) (*VirtualDesktopSessionSummary, error)
```

## State Files

### User State
- **Location**: `/srv/crateos/state/user-provisioning.json`
- **Contents**: Desired users, actual system users, reconciliation records, issues
- **Generated by**: `users.ProvisionUsers()` and `state.Apply()`

### Virtual Desktop State
- **Location**: `/srv/crateos/state/virtualization/<session-id>.json`
- **Contents**: Individual session details (PID, port, display, timestamps)
- **Generated by**: `virtualization.SessionManager`

### Audit Logs
- **SSH**: `/srv/crateos/logs/ssh/auth.jsonl` (JSONL, one entry per line)
- **Shell**: `/srv/crateos/logs/audit/shell-YYYYMMDD.jsonl` (date-based rotation)
- **Generated by**: `users.ssh` and `users.shell` functions

## Configuration Schema

### User Configuration (users.yaml)

```yaml
users:
  roles:
    admin:
      description: "Full platform access"
      permissions: ["*"]
    operator:
      description: "Service management"
      permissions: ["svc.*", "users.view"]
  users:
    - name: alice
      role: admin
      permissions: []  # No overrides
    - name: bob
      role: operator
      permissions:
        - "shell.breakglass"  # Grant extra perm
        - "-svc.delete"       # Deny specific perm
```

### Virtual Desktop Configuration (crateos.yaml)

```yaml
access:
  virtual_desktop:
    enabled: true
    provider: "vnc"      # vnc, x11, wayland, (rdp future)
    landing: "workspace" # console, panel, workspace, recovery
```

## Security Model

### User Isolation
- Each user gets unique UID/GID
- Home directories owned by user (700 perms)
- SSH keys in `~/.ssh/authorized_keys` (600 perms)
- System accounts created via `useradd`

### Shell Access Control
- Gated by `shell.breakglass` permission
- Shell runs as user (UID/GID dropped from agent root)
- Session logged with duration/exit code
- No direct root shell access

### Virtual Desktop Security
- Sessions run as user (UID/GID)
- Display servers bound to localhost
- VNC access requires SSH tunnel for network
- Session state persisted locally

### Permission Enforcement
- Happens in agent/TUI code
- Not via HTTP (no API to bypass)
- Direct Go function calls validate permissions
- Each function checks authorization

## Linux-Only Constraints

The following are Linux-only (graceful no-op on other platforms):

- User provisioning via `useradd`
- Virtual desktop sessions (Xvfb, etc.)
- systemd integration
- `/etc/passwd` probing

Non-Linux platforms work for:
- Configuration loading
- Permission checking
- TUI console (except shell access)

## No HTTP API

**Intentionally omitted:**
- No `/users/ssh/keys/add` endpoint
- No `/virtualization/sessions/start` endpoint
- No HTTP API layer for internal functions

**Why:**
- Direct Go function calls are faster
- Type safety via Go interfaces
- No serialization overhead
- No network exposure
- Simpler security model

**TUI and agent call functions directly:**
- `users.AddSSHKey(user, key, comment)`
- `virtualization.StartUserSession(user, type, landing)`
- `users.SpawnShell(cfg, req)`

## Integration Points

### Agent Integration
- `internal/state/engine.go`: Calls `users.ProvisionUsers()` during reconciliation
- `internal/state/platform_reconcile.go`: Calls `virtualization.ReconcileVirtualDesktop()`
- Actions logged to state action stream

### TUI Integration
- Direct imports of `users` and `virtualization` packages
- Calls functions when user selects menu options
- Checks permissions via `authz.Check()` before calling

### Configuration Integration
- `users.yaml`: User configuration
- `crateos.yaml`: Virtual desktop settings
- Loaded via standard `config.Load()`

## Future Enhancements

- [ ] RDP support via xrdp
- [ ] Session idle timeout
- [ ] Per-user session limits
- [ ] Session recording for audit
- [ ] LDAP/AD integration
- [ ] MFA (TOTP)
- [ ] SSH certificate support
