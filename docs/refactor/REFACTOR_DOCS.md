# CrateOS HTTP API Refactor - Documentation Index

**Completion Date**: March 9, 2026  
**Status**: ✅ COMPLETE  
**Type**: Architecture refactor - HTTP IPC → Direct function calls

## Quick Summary

CrateOS has been refactored to eliminate the internal HTTP API layer for user and service management. All management operations now use direct Go function calls instead of HTTP requests, following appliance-style system architecture best practices.

**Key Benefits:**
- Faster (no serialization overhead)
- Simpler (no network layer)
- Type-safe (Go function calls)
- More maintainable (less complexity)
- Reduced attack surface

## Documentation Files

### 1. **docs/refactor/REFACTOR_SUMMARY.md**
**Purpose**: Executive summary of the refactor  
**Audience**: Project leads, architects, reviewers  
**Contents**:
- Overview of changes
- Before/after architecture comparison
- Code statistics
- Validation checklist
- Migration notes for developers

**Read this if**: You want to understand what was done and why

### 2. **docs/INTERNAL_ARCHITECTURE.md**
**Purpose**: Complete system architecture documentation  
**Audience**: Developers building on CrateOS  
**Contents**:
- System architecture diagrams
- Component responsibilities
- Data flow examples
- Complete function signatures
- State file locations and schemas
- Configuration examples
- Security model
- Linux-only constraints

**Read this if**: You're developing features that interact with users, services, or virtual desktops

### 3. **docs/TUI_MANAGEMENT_FUNCTIONS.md**
**Purpose**: API reference for TUI management functions  
**Audience**: TUI developers, integration developers  
**Contents**:
- User management functions (`addUserDirect`, `updateUserDirect`, `deleteUserDirect`)
- Service management functions (`enableServiceDirect`, `disableServiceDirect`, `startServiceDirect`, `stopServiceDirect`)
- Function signatures and behavior
- Error cases
- Usage examples
- Design rationale

**Read this if**: You're modifying the TUI or adding new management features

## Key Files Changed

### Deleted (No Longer Needed)
```
internal/api/users_extended.go      (279 LOC) - User management endpoints
internal/api/virtualization.go      (175 LOC) - Virtual desktop endpoints
```

### Modified (Direct Calls Added)
```
internal/state/engine.go            (+12 LOC)  - User provisioning integration
internal/state/platform_reconcile.go (+20 LOC) - Virtual desktop reconciliation
internal/tui/app.go                 (+200 LOC) - Direct management functions
```

### Added (Documentation)
```
docs/INTERNAL_ARCHITECTURE.md       (337 LOC) - System architecture
docs/TUI_MANAGEMENT_FUNCTIONS.md    (322 LOC) - Function reference
docs/refactor/REFACTOR_SUMMARY.md   (240 LOC) - Refactor summary
```

## Integration Points

### Agent Reconciliation
User and service provisioning happens in the agent's main reconciliation loop:
- **File**: `internal/state/engine.go` (lines ~1014-1026)
- **Call**: `users.ProvisionUsers(cfg)` 
- **Effect**: Users are synced to system accounts during agent runs

### TUI Console
User and service management is done via direct function calls:
- **File**: `internal/tui/app.go` (lines ~2415-2612)
- **Functions**: `addUserDirect()`, `updateUserDirect()`, `deleteUserDirect()`, etc.
- **Effect**: Changes are written directly to config files

### Virtual Desktops
Session management is integrated into platform reconciliation:
- **File**: `internal/state/platform_reconcile.go`
- **Call**: `virtualization.ReconcileVirtualDesktop(cfg)`
- **Effect**: Session states are monitored and validated

## Configuration Files

### Users Configuration
**File**: `config/users.yaml`
**Modified by**:
- TUI management functions
- Agent provisioning

**Example**:
```yaml
users:
  roles:
    admin:
      permissions: ["*"]
  users:
    - name: alice
      role: admin
```

### Services Configuration
**File**: `config/services.yaml`
**Modified by**:
- TUI service lifecycle functions
- Agent reconciliation

**State Files**:
- `/srv/crateos/state/user-provisioning.json` - User sync state
- `/srv/crateos/state/virtualization/*.json` - Session states

## Testing Recommendations

### Unit Tests (Priority: High)
- Test each `*Direct()` function with valid/invalid inputs
- Verify config file mutations
- Check error messages

### Integration Tests (Priority: High)
- User form submission → config changes
- Service enable/disable → config changes
- Concurrent operations → no race conditions

### System Tests (Priority: Medium)
- Agent reconciliation → `/etc/passwd` updates
- Virtual desktop lifecycle → sessions tracked
- TUI → agent state synchronization

## Developer Workflow

### To Manage Users in TUI:
1. Use `addUserDirect()`, `updateUserDirect()`, `deleteUserDirect()`
2. These update the config file immediately
3. Next agent reconciliation syncs to system accounts

### To Manage Services in TUI:
1. Use `enableServiceDirect()`, `disableServiceDirect()`, etc.
2. These update the service config
3. `applyServiceAction()` and `state.RefreshCrateState()` handle lifecycle

### To Add New Management Operations:
1. Create a `*Direct()` function in `internal/tui/app.go`
2. Load config via `config.Load()`
3. Make changes to config structs
4. Save via `config.SaveUsers()` or `config.SaveServices()`
5. Handle errors appropriately

## Common Questions

### Q: Why remove the HTTP API?
A: CrateOS is an appliance, not a web service. Direct function calls are:
- Faster (no serialization)
- Simpler (no network layer)
- Type-safe (compile-time checking)
- More maintainable

### Q: What about remote access to the API?
A: The read-only status API (`/status`, `/users`, etc.) is preserved for TUI offline fallback. Management operations require local access anyway (they modify config files).

### Q: How does the agent know about user/service changes?
A: The agent's reconciliation loop runs periodically and:
- Calls `users.ProvisionUsers()` to sync users to system
- Calls `virtualization.ReconcileVirtualDesktop()` to manage sessions
- Observes config file changes and acts on them

### Q: What if multiple tools modify the config simultaneously?
A: Use last-write-wins approach:
1. Load config fresh before each operation
2. Make changes
3. Save immediately
4. No in-memory caching

### Q: Are there any breaking changes?
A: No - CLI commands work the same way, TUI UI is unchanged. Only internal architecture changed.

## References

### Related Code
- `internal/users/provisioning.go` - System account sync
- `internal/users/ssh.go` - SSH key management
- `internal/users/shell.go` - Break-glass shell
- `internal/virtualization/sessions.go` - Desktop sessions
- `internal/virtualization/reconcile.go` - Session reconciliation
- `internal/config/config.go` - Configuration interface
- `internal/api/api.go` - Read-only status API (preserved)

### External References
- CrateOS README.md - Project overview
- CrateOS CONTRIBUTING.md - Development guidelines
- Go best practices - https://golang.org/doc/effective_go

## Document Versions

| Date | Version | Changes |
|------|---------|---------|
| 2026-03-09 | 1.0 | Initial complete refactor documentation |

## Next Steps

1. **Testing**: Add comprehensive test suite for direct functions
2. **Performance**: Benchmark against old HTTP approach
3. **Documentation**: Update user-facing docs to reflect changes
4. **Integration**: Verify with other CrateOS components
5. **Release**: Include refactor summary in release notes

---

**Questions?** Refer to the specific documentation files above, or check the refactor diff in git history.
