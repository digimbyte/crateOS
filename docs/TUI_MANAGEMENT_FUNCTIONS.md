# TUI Direct Management Functions

**Purpose**: Reference guide for the direct management functions added to `internal/tui/app.go` to replace HTTP API calls.

**Location**: `internal/tui/app.go` (lines 2415-2612)

## Overview

The TUI console uses direct Go function calls to manage users and services instead of making HTTP requests to the API server. These functions provide a clean interface between the TUI UI layer and the configuration layer.

## User Management Functions

### addUserDirect()

Adds a new user to the CrateOS configuration.

```go
func addUserDirect(name, role string, perms []string) error
```

**Parameters:**
- `name` (string): Username - must be unique
- `role` (string): User role (admin, operator, staff, viewer, etc.)
- `perms` ([]string): Additional permissions beyond role defaults

**Returns:**
- `error`: Non-nil if user already exists or config save fails

**Behavior:**
1. Loads current configuration
2. Checks for duplicate username
3. Appends new `config.UserEntry` to user list
4. Saves configuration to disk via `config.SaveUsers()`

**Error Cases:**
- "user already exists" - if `name` matches existing user
- config load/save errors propagated from config package

**Used By:**
- User command handler: `user add <name> <role>`
- User form submission dialog

### updateUserDirect()

Updates an existing user's name, role, or permissions.

```go
func updateUserDirect(targetName, newName, newRole string, newPerms []string) error
```

**Parameters:**
- `targetName` (string): Current username to locate and update
- `newName` (string): New username (rename), empty string = no change
- `newRole` (string): New role, empty string = no change
- `newPerms` ([]string): New permissions list, nil = no change

**Returns:**
- `error`: Non-nil if user not found, rename conflicts, or config save fails

**Behavior:**
1. Loads current configuration
2. Finds user entry matching `targetName`
3. If `newName` provided and different:
   - Checks for duplicate with new name
   - Updates user's name
4. If `newRole` provided and non-empty:
   - Updates user's role
5. If `newPerms` provided (non-nil):
   - Replaces permissions list
6. Saves configuration via `config.SaveUsers()`

**Error Cases:**
- "target name required" - if targetName is empty/whitespace
- "user not found" - if targetName doesn't match any user
- "user already exists" - if newName conflicts with another user
- config load/save errors

**Used By:**
- User rename command: `user rename <old> <new>`
- User role cycle: pressing 'r' on selected user
- User permissions toggle: pressing 'p' on selected user
- User form edit submission

**Example Usage:**
```go
// Rename user
updateUserDirect("alice", "alice2", "", nil)

// Change role only
updateUserDirect("bob", "", "admin", nil)

// Update permissions
updateUserDirect("charlie", "", "", []string{"shell.access", "users.view"})

// Do all three
updateUserDirect("dave", "david", "operator", []string{"svc.list"})
```

### deleteUserDirect()

Removes a user from the CrateOS configuration.

```go
func deleteUserDirect(name string) error
```

**Parameters:**
- `name` (string): Username to delete

**Returns:**
- `error`: Non-nil if user not found or config save fails

**Behavior:**
1. Loads current configuration
2. Filters user list, removing any entry matching `name`
3. If user was found, saves updated list via `config.SaveUsers()`
4. If user was not found, returns error

**Error Cases:**
- "user name required" - if name is empty/whitespace
- "user not found" - if name doesn't match any user
- config load/save errors

**Used By:**
- User delete command: `user delete <name>`
- Keyboard shortcut 'd' on selected user

## Service Management Functions

### enableServiceDirect()

Enables a service and configures it to autostart if appropriate.

```go
func enableServiceDirect(name string) error
```

**Parameters:**
- `name` (string): Service name from configuration

**Returns:**
- `error`: Non-nil if service not found or config save fails

**Behavior:**
1. Loads current configuration
2. Finds service entry matching `name`
3. Sets `Enabled = true`
4. Sets `Autostart = shouldAutostartOnEnable(name, mods)` (smart autostart)
5. Saves via `config.SaveServices(cfg)`
6. Calls `applyServiceAction(name, serviceActionEnableOnly, mods)`
7. Calls `state.RefreshCrateState(name)` to update state files

**Error Cases:**
- "service name required" - if name is empty
- "service not found" - if name doesn't match any service
- config save errors
- state refresh errors

**Used By:**
- Service enable command: `svc enable <service>`
- Keyboard shortcut 'e' on selected service
- Service lifecycle handler

### disableServiceDirect()

Disables a service and prevents autostart.

```go
func disableServiceDirect(name string) error
```

**Parameters:**
- `name` (string): Service name from configuration

**Returns:**
- `error`: Non-nil if service not found or config save fails

**Behavior:**
1. Loads current configuration
2. Finds service entry matching `name`
3. Sets `Enabled = false`
4. Sets `Autostart = false`
5. Saves via `config.SaveServices(cfg)`
6. Calls `applyServiceAction(name, serviceActionDisable, mods)`
7. Calls `state.RefreshCrateState(name)` to update state files

**Error Cases:**
- "service name required"
- "service not found"
- config/state save errors

**Used By:**
- Service disable command: `svc disable <service>`
- Keyboard shortcut 'd' on selected service
- Service lifecycle handler

### startServiceDirect()

Starts a service and enables autostart.

```go
func startServiceDirect(name string) error
```

**Parameters:**
- `name` (string): Service name from configuration

**Returns:**
- `error`: Non-nil if service not found or config save fails

**Behavior:**
1. Loads current configuration
2. Finds service entry matching `name`
3. Sets `Enabled = true`
4. Sets `Autostart = true` (autostart on start)
5. Saves via `config.SaveServices(cfg)`
6. Calls `applyServiceAction(name, serviceActionStart, mods)`
7. Calls `state.RefreshCrateState(name)` to update state files

**Error Cases:**
- "service name required"
- "service not found"
- config/state save errors

**Used By:**
- Service start command: `svc start <service>`
- Keyboard shortcut 's' on selected service
- Service lifecycle handler
- Service restart (stop then start)

### stopServiceDirect()

Stops a service but keeps it enabled (no autostart).

```go
func stopServiceDirect(name string) error
```

**Parameters:**
- `name` (string): Service name from configuration

**Returns:**
- `error`: Non-nil if service not found or config save fails

**Behavior:**
1. Loads current configuration
2. Finds service entry matching `name`
3. Sets `Enabled = true` (remains enabled for manual operation)
4. Sets `Autostart = false` (don't restart on boot)
5. Saves via `config.SaveServices(cfg)`
6. Calls `applyServiceAction(name, serviceActionStop, mods)`
7. Calls `state.RefreshCrateState(name)` to update state files

**Error Cases:**
- "service name required"
- "service not found"
- config/state save errors

**Used By:**
- Service stop command: `svc stop <service>`
- Keyboard shortcut 'x' on selected service
- Service lifecycle handler
- Service restart (stop then start)

## Design Notes

### Why Direct Functions?

These functions replace HTTP API calls because:

1. **Type Safety** - Go function calls catch type mismatches at compile time
2. **Performance** - No serialization/deserialization overhead
3. **Simplicity** - No network layer to maintain
4. **Consistency** - All management operations follow same code path
5. **Appliance Model** - CrateOS is a system tool, not a web service

### Error Handling

All functions:
- Return errors explicitly (Go idiom)
- Use `fmt.Errorf()` for user-readable messages
- Propagate config package errors unchanged
- Check preconditions before making changes

TUI handlers:
- Log errors via `m.setCommandError()`
- Display user-friendly error messages
- Refresh UI state after successful operations
- Show partial success when multiple operations fail

### State Consistency

1. Functions load config fresh each time (simple, safe)
2. Changes are written immediately to disk
3. Agent's reconciliation loop picks up changes
4. No in-memory state divergence
5. Multiple CLI instances are safe (last-write-wins on config)

### Integration with Agent

- User changes appear in config files
- Agent's `users.ProvisionUsers()` syncs to `/etc/passwd`
- Service changes in config control service lifecycle
- Virtual desktop state is reconciled by agent
- TUI reflects agent's current state via `fetchStatusViaAPI()`

## Testing Checklist

- [ ] `addUserDirect()` - duplicate detection, persistence
- [ ] `updateUserDirect()` - rename, role change, perms update
- [ ] `deleteUserDirect()` - removal, missing user error
- [ ] Service functions - config mutations, action invocation
- [ ] Concurrent operations - no race conditions
- [ ] Config file consistency - valid YAML after operations
- [ ] Error messages - user-friendly and actionable

## See Also

- `docs/INTERNAL_ARCHITECTURE.md` - System design overview
- `docs/refactor/REFACTOR_SUMMARY.md` - Refactor rationale and changes
- `internal/config/config.go` - Configuration package interface
- `internal/tui/app.go` - Full TUI implementation
