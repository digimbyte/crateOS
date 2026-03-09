# CrateOS Crate State Machine

This document defines the **state machine contract** for CrateOS modules (“crates”).

The purpose of the state machine is to ensure every crate behaves predictably in the panel, TUI, agent, and internal automation layer.

Crates may differ in software, runtime, and complexity, but they must all present the same **user-facing lifecycle**.

---

## Core principle

A crate is a **managed object**, not a package install.

That means users interact with crates through consistent lifecycle states such as:

* installed
* configured
* enabled
* running
* degraded
* disabled
* removed

CrateOS may perform many backend steps behind the scenes, but the visible lifecycle must remain clean and predictable.

---

## Canonical states

Every crate should move through the following canonical states:

1. **Available**
2. **Installing**
3. **Installed**
4. **Configuring**
5. **Configured**
6. **Enabling**
7. **Enabled**
8. **Starting**
9. **Running**
10. **Degraded**
11. **Stopping**
12. **Disabled**
13. **Updating**
14. **Removing**
15. **Removed**
16. **Error**

---

## State meanings

### Available

The crate exists in the registry/catalog and is not currently installed on the machine.

### Installing

CrateOS is preparing the crate for use.
This may include:

* package download/install
* directory creation
* service registration
* asset extraction
* deferred install staging

### Installed

The crate definition is active on the machine.
It may not yet be fully configured or enabled.

### Configuring

CrateOS is validating and applying module configuration.
This may include:

* generating templates
* binding storage paths
* copying secrets
* validating required inputs

### Configured

The crate has valid configuration and is ready to be enabled.

### Enabling

CrateOS is transitioning the crate into a desired active state.
This may include:

* enabling systemd units
* enabling reverse proxy mapping
* binding ports
* registering health checks

### Enabled

The crate is marked as desired-to-run, but may not yet be healthy.

### Starting

CrateOS is actively starting the crate runtime.
This may include:

* starting a systemd unit
* starting a container
* waiting for dependencies

### Running

The crate is active and all required health checks pass.

### Degraded

The crate is active but one or more health checks are failing or partial functionality is detected.

### Stopping

CrateOS is actively stopping the crate runtime.

### Disabled

The crate remains installed and configured, but is not desired to run.

### Updating

CrateOS is updating the crate version, assets, templates, or runtime config.

### Removing

CrateOS is uninstalling the crate and cleaning up its managed resources.

### Removed

The crate is no longer installed or managed by the local machine.

### Error

CrateOS attempted an operation and it failed in a way that requires intervention or rollback.

---

## Stable vs transitional states

### Stable states

These are states the user can meaningfully view as a resting condition:

* Available
* Installed
* Configured
* Enabled
* Running
* Degraded
* Disabled
* Removed
* Error

### Transitional states

These should usually be short-lived and indicate active work:

* Installing
* Configuring
* Enabling
* Starting
* Stopping
* Updating
* Removing

The UI should clearly distinguish transitional states from resting states.

---

## Core lifecycle flow

Typical crate lifecycle:

```text
Available
  -> Installing
  -> Installed
  -> Configuring
  -> Configured
  -> Enabling
  -> Enabled
  -> Starting
  -> Running
```

If health checks partially fail:

```text
Running -> Degraded
```

If the user disables the crate:

```text
Running -> Stopping -> Disabled
Degraded -> Stopping -> Disabled
Enabled -> Disabled
```

If the user removes the crate:

```text
Disabled -> Removing -> Removed
Installed -> Removing -> Removed
Configured -> Removing -> Removed
Error -> Removing -> Removed
```

---

## Allowed transitions

### From Available

* `install` -> Installing

### From Installing

* success -> Installed
* failure -> Error
* rollback -> Available

### From Installed

* `configure` -> Configuring
* `remove` -> Removing

### From Configuring

* success -> Configured
* failure -> Error
* rollback -> Installed

### From Configured

* `enable` -> Enabling
* `remove` -> Removing
* `reconfigure` -> Configuring

### From Enabling

* success -> Enabled
* failure -> Error

### From Enabled

* `start` -> Starting
* `disable` -> Disabled
* `update` -> Updating
* `remove` -> Removing

### From Starting

* health passes -> Running
* health partial -> Degraded
* failure -> Error

### From Running

* health failure -> Degraded
* `disable` -> Stopping
* `restart` -> Stopping
* `update` -> Updating
* `remove` -> Removing
* `reconfigure` -> Configuring

### From Degraded

* health recovers -> Running
* `disable` -> Stopping
* `restart` -> Stopping
* `repair` -> Configuring
* `update` -> Updating
* `remove` -> Removing

### From Stopping

* success -> Disabled
* failure -> Error

### From Disabled

* `enable` -> Enabling
* `start` -> Starting
* `update` -> Updating
* `remove` -> Removing
* `reconfigure` -> Configuring

### From Updating

* success while active -> Starting or Running
* success while inactive -> Disabled or Configured
* failure -> Error

### From Removing

* success -> Removed
* failure -> Error

### From Removed

* `install` -> Installing

### From Error

* `repair` -> Configuring
* `disable` -> Stopping or Disabled
* `remove` -> Removing
* `retry last action` -> relevant transitional state

---

## State invariants

These rules should always hold.

### Installed invariant

If a crate is in any state except `Available` or `Removed`, then:

* the crate must exist in the local module registry/state
* the canonical service directory should exist under `/srv/crateos/services/<module>/`

### Configured invariant

If a crate is `Configured`, `Enabled`, `Running`, or `Degraded`, then:

* required config must validate successfully
* configSchema required fields must be present

### Enabled invariant

If a crate is `Enabled`, `Starting`, `Running`, or `Degraded`, then:

* desired state says the crate should be active

### Running invariant

If a crate is `Running`, then:

* all required health checks must pass

### Degraded invariant

If a crate is `Degraded`, then:

* the crate is still considered active
* at least one required health check is failing or partial functionality is detected

### Disabled invariant

If a crate is `Disabled`, then:

* desired state says the crate should not be active
* it may still be installed and configured

---

## User-facing semantics

The panel/TUI should expose a simplified mental model:

| State      | User meaning                         |
| ---------- | ------------------------------------ |
| Available  | Not installed                        |
| Installed  | Installed, needs setup               |
| Configured | Ready to enable                      |
| Enabled    | Marked on, still starting or pending |
| Running    | Healthy and live                     |
| Degraded   | Running, but needs attention         |
| Disabled   | Installed, but off                   |
| Error      | Failed operation                     |
| Removed    | Uninstalled                          |

Transitional states should generally appear as activity indicators, not long-term labels.

---

## Health and degradation model

CrateOS should not assume “service started” means “healthy.”

### Running

All required health checks pass.

### Degraded

Any of the following may trigger degradation:

* health endpoint failing
* process exists but port not listening
* port listening but application not ready
* dependency missing
* storage path unavailable
* reverse proxy target unhealthy

A degraded crate should remain manageable and visible without being treated as fully dead.

---

## Desired state vs actual state

Each crate has both:

### Desired state

What CrateOS wants the crate to be.
Examples:

* installed = true
* enabled = true
* configured = true

### Actual state

What the machine currently reports.
Examples:

* package present
* service active
* health checks passing
* directories available

The agent reconciles actual state toward desired state.

This means state changes may be:

* user-triggered
* policy-triggered
* health-triggered
* repair-triggered

---

## State persistence

Global registry example:

```text
/srv/crateos/state/modules.json
```

Per-crate state example:

```text
/srv/crateos/services/postgres/state.json
```

Recommended fields:

```json
{
  "id": "postgres",
  "state": "Running",
  "installed": true,
  "configured": true,
  "enabled": true,
  "health": "ok",
  "lastAction": "start",
  "lastActionAt": "2026-03-07T12:00:00Z",
  "lastError": null,
  "version": "16"
}
```

---

## Failure handling

A failed transition should not leave the crate in an undefined state.

### Rules

* Any failed transitional action must end in `Error` or a valid rollback state.
* The previous stable state should be preserved when possible.
* The last successful state should be retained for UI display and repair logic.

Examples:

### Configure failure

```text
Installed -> Configuring -> Error
```

Possible repair:

```text
Error -> Configuring -> Configured
```

### Start failure

```text
Enabled -> Starting -> Error
```

Possible repair:

```text
Error -> Starting -> Running
```

### Remove failure

```text
Disabled -> Removing -> Error
```

Possible repair:

```text
Error -> Removing -> Removed
```

---

## Restart semantics

Restart should be treated as a composite action, not a separate permanent state.

Recommended transition:

```text
Running -> Stopping -> Starting -> Running
```

If restart fails:

```text
Running -> Stopping -> Starting -> Error
```

---

## Update semantics

Updates should preserve the user-facing model.

Examples:

### Updating an active crate

```text
Running -> Updating -> Starting -> Running
```

### Updating an inactive crate

```text
Disabled -> Updating -> Disabled
```

### Update failure

```text
Updating -> Error
```

Where possible, CrateOS should support rollback or last-good restore.

---

## Removal semantics

Removal should support policy levels such as:

* remove runtime only
* remove runtime + config
* remove runtime + config + data

These are **removal options**, not separate states.

The resulting state after success is always:

```text
Removed
```

---

## State machine rules for module authors

Module authors must design crates that fit this lifecycle.

### Required behaviors

* support install
* support configure
* support enable/disable
* support health-aware running state
* support remove

### Strong recommendations

* support update cleanly
* support repair/reconcile cleanly
* support backup/restore when stateful

### Non-goals

* custom state machines per module
* package-manager-specific UX
* modules that skip health semantics entirely

---

## Reference examples

### PostgreSQL

Typical flow:

```text
Available
-> Installing
-> Installed
-> Configuring
-> Configured
-> Enabling
-> Enabled
-> Starting
-> Running
```

Possible degraded condition:

```text
Running -> Degraded
```

when:

* port 5432 closed
* `pg_isready` fails
* storage path unavailable

### Nginx

Possible flow:

```text
Configured -> Enabling -> Enabled -> Starting -> Running
```

Possible error condition:

```text
Starting -> Error
```

when config test fails.

### Redis

Possible flow:

```text
Installed -> Configuring -> Configured -> Starting -> Running
```

Possible degraded condition:

* persistence misconfigured
* port not listening

---

## Summary

The CrateOS crate state machine exists to make all modules behave like the same class of object.

No matter what software sits underneath, the user should always see:

* a predictable set of states
* a predictable set of actions
* a clear difference between desired state, actual state, and health

This is what keeps CrateOS feeling like a platform instead of a pile of package wrappers.
