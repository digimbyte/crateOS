# CrateOS User Interaction & Authentication System

## Overview

This document describes the complete Ubuntu user interaction layer for CrateOS, including system account provisioning, SSH authentication, authorization, and break-glass shell access.

## Architecture

### Components

```
users.yaml (Config)
    ↓
config.Load()
    ↓
users.ProvisionUsers()
    ├─ users/provisioning.go: System account creation/sync
    ├─ users/ssh.go: SSH key & auth management
    └─ users/shell.go: Break-glass shell access
    ↓
state/user-provisioning.json (Rendered state)
```

### Layering

1. **Configuration Layer**: `users.yaml` defines desired CrateOS users with roles/permissions
2. **Provisioning Layer**: `internal/users/provisioning.go` syncs config → system accounts (`/etc/passwd`)
3. **Authentication Layer**: `internal/users/ssh.go` validates SSH logins against authorized keys
4. **Session Layer**: `internal/users/shell.go` handles break-glass shell spawning with audit
5. **API Layer**: `internal/api/users_extended.go` exposes functions via HTTP

## User Provisioning

### System Account Creation

When `ProvisionUsers()` is called:

1. **Desired State**: Build list from `cfg.Users.Users`
2. **Actual State**: Probe `/etc/passwd` via `probeSystemUsers()`
3. **Reconciliation**: For each desired user:
   - If not exist: `useradd` with home dir at `/home/<username>`
   - If exist: Check if home/shell need updates
   - Auto-create `.ssh/authorized_keys` file
4. **Persist State**: Write to `state/user-provisioning.json`

### User Home Directory Structure

```
/home/<username>/
  .ssh/
    authorized_keys    (SSH public keys, 0600)
  .config/            (User configuration)
  projects/           (Default working directory)
```

### Shell Assignment

All CrateOS users get `/usr/local/bin/crateos-shell-wrapper` as their login shell. This wrapper:
- Drops users into the CrateOS console (TUI)
- Prevents direct shell access unless break-glass is enabled
- Can be overridden with custom shells via config

## SSH Authentication

### Flow

```
SSH Client
    ↓ (SSH pubkey attempt)
sshd (with ForceCommand)
    ↓ (validates key in authorized_keys)
/usr/local/bin/crateos console
    ↓ (spawns TUI)
CrateOS Console (validate user roles)
```

### Key Management API

**List user's SSH keys:**
```bash
curl -X GET http://unix/users/ssh/keys/list \
  -H "X-CrateOS-User: alice"
```

Response:
```json
{
  "keys": [
    {
      "username": "alice",
      "key": "ssh-rsa AAAA...",
      "comment": "laptop",
      "added_at": "2026-03-09T20:14:32Z"
    }
  ]
}
```

**Add SSH key:**
```bash
curl -X POST http://unix/users/ssh/keys/add \
  -H "X-CrateOS-User: alice" \
  -d '{
    "user": "alice",
    "key": "ssh-rsa AAAA...",
    "comment": "new laptop"
  }'
```

**Remove SSH key:**
```bash
curl -X POST http://unix/users/ssh/keys/remove \
  -H "X-CrateOS-User: alice" \
  -d '{
    "user": "alice",
    "key": "ssh-rsa AAAA..."
  }'
```

### SSH Auth Validation

The API endpoint `/users/ssh/auth` validates authentication:

```bash
curl -X POST http://unix/users/ssh/auth \
  -d '{
    "user": "alice",
    "method": "publickey",
    "key": "ssh-rsa AAAA..."
  }'
```

Response:
```json
{
  "allowed": true,
  "user": "alice",
  "home": "/home/alice",
  "permissions": ["svc.*", "users.view"]
}
```

### Audit Logging

All SSH authentication attempts are logged to:
```
/srv/crateos/logs/ssh/auth.jsonl
```

Each entry:
```json
{
  "timestamp": "2026-03-09T20:14:32Z",
  "user": "alice",
  "method": "publickey",
  "result": "success",
  "fingerprint": "<key fingerprint>"
}
```

Query recent attempts:
```bash
curl http://unix/users/ssh/audit?limit=50 \
  -H "X-CrateOS-User: admin"
```

## Break-Glass Shell Access

### Configuration

In `crateos.yaml`:

```yaml
access:
  break_glass:
    enabled: true
    require_permission: "shell.breakglass"
    allowed_surfaces: ["ssh"]
```

### Permission Check

Users must:
1. Be in `users.yaml`
2. Have the `shell.breakglass` permission (via role or override)
3. Request shell access via API or TUI menu

### Shell Access API

```bash
curl -X POST http://unix/users/shell/access \
  -H "X-CrateOS-User: alice" \
  -d '{
    "reason": "debugging network issue"
  }'
```

If permitted, this spawns an interactive bash/sh shell as the user (UID/GID dropped).

### Audit Trail

Shell access events logged to:
```
/srv/crateos/logs/audit/shell-YYYYMMDD.jsonl
```

Example entry:
```json
{
  "timestamp": "2026-03-09T20:14:32Z",
  "user": "alice",
  "event_type": "shell",
  "result": "allowed",
  "reason": "debugging network issue",
  "duration": "2m45s",
  "exit_code": 0
}
```

## Permissions Model

### Built-in Roles

Define in `users.yaml`:

```yaml
users:
  roles:
    admin:
      description: "Full platform access including break-glass shell"
      permissions: ["*"]
    
    operator:
      description: "Service management and logs"
      permissions:
        - "svc.*"
        - "audit.view"
        - "-shell.breakglass"  # explicitly deny
    
    viewer:
      description: "Read-only access"
      permissions:
        - "svc.view"
        - "audit.view"
```

### Permission Syntax

- `*` - full access
- `svc.*` - all service actions
- `svc.nginx.restart` - specific service action
- `users.edit` - user management
- `audit.*` - all audit logs
- `shell.breakglass` - break-glass shell access
- `-permission` - explicit deny (override role)

### Permission Enforcement Points

1. **API Layer**: Each handler calls `authz.Check(user, perm)`
2. **Service Actions**: enable/disable/start/stop require `svc.*`
3. **User Management**: add/delete/update require `users.edit`
4. **Shell Access**: requires `shell.breakglass`
5. **Audit Logs**: requires `audit.*` or admin role

## Integration with Platform State

### User Provisioning Adapter

During platform reconciliation, users are provisioned:

```go
actions, userState := reconcileUsers(cfg)
```

This produces a `PlatformAdapterState` with:
- **Status**: validation/apply results
- **Issues**: provisioning errors
- **Summary**: user count and actions taken
- **Rendered paths**: state files written

### Example State Output

```json
{
  "generated_at": "2026-03-09T20:14:32Z",
  "desired_users": [
    {
      "name": "alice",
      "role": "admin",
      "home": "/home/alice",
      "shell": "/usr/local/bin/crateos-shell-wrapper",
      "permissions": ["*"]
    }
  ],
  "actual_users": [
    {
      "name": "alice",
      "uid": 1000,
      "gid": 1000,
      "home": "/home/alice",
      "shell": "/usr/local/bin/crateos-shell-wrapper"
    }
  ],
  "reconciled": [
    {
      "user": "alice",
      "action": "skip",
      "status": "skipped",
      "timestamp": "2026-03-09T20:14:32Z"
    }
  ],
  "issues": [],
  "summary": "provisioned 1 users (0 created, 1 skipped)"
}
```

## Security Considerations

### SSH Key Storage

- Keys stored in `~/.ssh/authorized_keys` (0600 perms)
- Never stored in config or state files
- Audit log only tracks fingerprints, not full keys
- No password auth (key-only)

### Shell Access Control

- Only users with explicit `shell.breakglass` permission
- All shell sessions logged with duration and exit code
- Shell runs as the user (UID/GID dropped from root)
- Works only on Linux (non-Linux returns error)

### User Isolation

- Each CrateOS user = unique system account
- Home directories privately owned (0700)
- Supplementary groups can be configured for service access
- No privilege escalation without break-glass

## Troubleshooting

### User Not Provisioned

Check:
1. User in `users.yaml`?
2. `useradd` available on system?
3. `/home` directory exists?
4. Check `state/user-provisioning.json` for error

### SSH Login Fails

Check:
1. System account exists: `id alice`
2. Authorized key in `~/.ssh/authorized_keys`
3. SSH audit log: `/srv/crateos/logs/ssh/auth.jsonl`
4. CrateOS console accessible: `crateos console`

### Break-Glass Shell Denied

Check:
1. User has `shell.breakglass` permission
2. `access.break_glass.enabled = true` in config
3. Check shell audit log for denials

## Future Enhancements

- [ ] Multi-factor authentication (TOTP)
- [ ] SSH certificate support
- [ ] LDAP/AD integration
- [ ] Session token caching
- [ ] Role-based service account provisioning
- [ ] Virtual desktop session management
