# CrateOS Agent Reconcile

This document defines the **CrateOS agent reconcile model**.

The agent is the **control plane daemon** responsible for turning CrateOS desired state into actual machine state.

It is the brain of the platform.

The registry tells CrateOS what crates exist.
The module spec defines what a crate is.
The state machine defines valid lifecycle transitions.
The runtime executes workloads.
The agent reconcile loop decides **what should happen next**.

---

## Core principle

CrateOS is not shell-first administration.
It is a **desired-state platform**.

That means the system should not depend on users manually:

* restarting services
* editing native files directly
* remembering daemon-specific commands
* reapplying broken config after reboot

Instead, the agent continuously or explicitly reconciles the machine toward the desired CrateOS model.

---

## Agent responsibilities

The agent is responsible for:

* reading desired state
* reading current machine state
* resolving installed module definitions
* validating compatibility and dependencies
* computing diffs
* applying transitions safely
* updating crate state
* repairing drift where possible
* surfacing health, warnings, and failures to the TUI/panel

The agent is **not**:

* the user interface
* the registry itself
* the package manager
* the runtime implementation

It coordinates those systems.

---

## Desired state vs actual state

### Desired state

Desired state comes from CrateOS-managed configuration and crate metadata.

Typical sources:

```text
/srv/crateos/config/*.yaml
/srv/crateos/services/<crate>/overrides.yaml
/srv/crateos/state/desired.json
```

Desired state answers questions like:

* should this crate be installed?
* should it be enabled?
* what config values should it have?
* what ports should be exposed?
* what storage path should it use?

### Actual state

Actual state is what the machine currently reports.

Typical sources:

* package presence
* systemd unit status
* docker container status
* rendered config files
* health checks
* port checks
* path existence
* ownership/permissions
* native service status

### Reconcile goal

The agent compares actual state against desired state and moves actual state toward desired state.

---

## Reconcile loop model

At a high level, the agent loop is:

1. load desired state
2. discover installed and available crates
3. inspect actual state
4. validate state and dependencies
5. compute required transitions
6. apply transitions in safe order
7. run health checks
8. persist updated crate state
9. emit events/logs

---

## Reconcile triggers

Reconcile should support multiple trigger types.

### 1. Explicit user action

Examples:

* install crate
* enable crate
* reconfigure crate
* remove crate

### 2. Boot/startup reconcile

When the machine boots, the agent must restore desired platform state.

### 3. Periodic drift reconcile

A timer-based pass ensures:

* crashed services are detected
* config drift is noticed
* broken mappings are repaired

### 4. Event-driven reconcile (future)

Examples:

* package changed
* disk mounted/unmounted
* network state changed
* dependency crate changed

Initial implementation can focus on explicit + boot + periodic.

---

## Reconcile phases

Each agent run should be conceptually divided into phases.

### Phase 1: Load

Read:

* platform config
* installed crate records
* module manifests
* local registry cache
* prior state records

### Phase 2: Discover

Probe actual state:

* installed packages
* runtime presence
* systemd state
* docker state
* native paths
* health checks
* logs/runtime markers

### Phase 3: Validate

Check:

* module compatibility
* config validity
* dependency availability
* required storage/network assumptions
* trust/warning conditions

### Phase 4: Plan

Build an execution plan:

* install crate A
* configure crate B
* restart crate C
* disable crate D
* update proxy mapping for crate E

The plan should be deterministic and ordered.

### Phase 5: Apply

Execute planned transitions through the runtime and module lifecycle system.

### Phase 6: Verify

Run health checks and confirm the expected final states.

### Phase 7: Persist

Write updated state:

```text
/srv/crateos/state/modules.json
/srv/crateos/services/<crate>/state.json
```

### Phase 8: Emit

Send logs/events/status updates to:

* TUI
* web panel
* logs
* local event stream

---

## Planning model

The agent should not blindly “do stuff.”
It should build a clear plan first.

A reconcile plan is a sequence of actions such as:

* install module
* render config
* map storage
* enable service
* start service
* run health checks
* register reverse proxy
* open firewall rule

This gives CrateOS:

* reproducibility
* better logs
* better rollback behavior
* easier debugging

---

## Ordering rules

The agent must reconcile in a safe order.

Example order:

1. dependencies
2. storage paths
3. config rendering
4. runtime registration
5. enablement
6. startup
7. network/proxy exposure
8. health validation

### Example

For a crate that depends on Postgres:

```text
postgres -> app config -> app runtime -> proxy mapping
```

Not the other way around.

---

## Reconcile scopes

The agent should support multiple scopes.

### Full reconcile

Reconcile all crates and platform components.

### Crate reconcile

Reconcile a single named crate.

### Component reconcile

Reconcile a specific subsystem.
Examples:

* runtime only
* proxy only
* storage only

### Repair reconcile

Attempt to restore a degraded or error crate.

---

## Drift model

Drift means actual state no longer matches desired state.

Examples:

* service stopped unexpectedly
* native config changed manually
* storage path missing
* reverse proxy target stale
* firewall rules missing
* unit disabled behind CrateOS

The agent should classify drift into levels:

### Soft drift

Can be automatically repaired safely.
Examples:

* service stopped
* symlink missing
* rendered file out of date

### Hard drift

Requires warning or explicit approval.
Examples:

* storage device missing
* module package removed manually
* destructive config mismatch

### Unsupported drift

Detected but outside current repair scope.
The UI should warn clearly.

---

## State transition ownership

The agent owns state transitions.

That means:

* users request actions
* the agent validates and performs them
* state is only committed after execution and verification

The TUI/panel should never directly mutate runtime state behind the agent.

---

## Error handling model

A reconcile pass must never leave a crate in a mystery state.

### Rules

* failed actions must produce structured errors
* previous stable state should be preserved when possible
* partial failures must be visible
* state machine transitions must remain valid

### Example

If config rendering succeeds but service start fails:

```text
Configured -> Enabling -> Enabled -> Starting -> Error
```

The agent should record:

* last action
* failure reason
* suggested repair action if known

---

## Retry model

The agent should support controlled retries.

Suggested rules:

* transient errors may be retried automatically
* repeated failure should back off
* permanent misconfiguration should not be retried endlessly

Examples:

### Retryable

* service start timeout
* delayed dependency availability
* temporary port conflict during update

### Non-retryable without user fix

* invalid config value
* missing required storage path
* incompatible module version

---

## Health-aware reconciliation

Health is not just an output. It informs agent decisions.

Examples:

* `Running` -> health failure -> mark `Degraded`
* `Degraded` + fix available -> schedule repair reconcile
* `Enabled` but never healthy -> escalate to `Error`

Health checks must always happen after relevant runtime actions.

---

## Dependency reconciliation

Dependencies must be explicit and validated before applying dependent crates.

Examples:

* app depends on `postgres`
* proxy mapping depends on `nginx`
* backup module depends on storage availability

The agent should:

* detect missing dependencies
* sequence dependencies first
* block or warn on invalid dependency graphs

---

## Config rendering and apply

The agent should treat configuration as a render-and-apply system.

Flow:

1. load crate config
2. validate against config schema
3. render templates
4. place generated runtime config
5. run syntax validation if supported
6. apply runtime changes

Examples:

* `nginx -t` before reload
* validate postgres config before restart

---

## Rollback model

Where feasible, the agent should preserve last-good state.

Suggested locations:

```text
/srv/crateos/state/last-good/
/srv/crateos/services/<crate>/runtime/last-good/
```

Rollback may include:

* previous rendered config
* previous state record
* previous runtime artifact

Rollback is most useful for:

* bad config changes
* failed updates
* failed template render/apply transitions

---

## Observability

The agent should emit enough information for users to understand what happened.

At minimum:

* action started
* action succeeded/failed
* current crate state
* health summary
* drift detected
* repair attempted

Outputs may go to:

* `/srv/crateos/logs/agent.log`
* per-crate logs
* TUI status stream
* web panel events

---

## Concurrency model

Initial implementation should keep concurrency conservative.

Recommended rule:

* independent crates may reconcile in parallel later
* but stateful/dependent transitions should default to ordered execution first

Do not optimize for concurrency before correctness.

---

## Agent API model

The agent should expose an internal API for the UI and automation layer.

Examples:

* `ReconcileAll()`
* `ReconcileCrate(id)`
* `InstallCrate(id)`
* `ConfigureCrate(id)`
* `EnableCrate(id)`
* `DisableCrate(id)`
* `RemoveCrate(id)`
* `RepairCrate(id)`
* `GetCrateState(id)`
* `GetReconcilePlan(id)`

The panel/TUI should talk to the agent, not directly to runtime systems.

---

## Boot behavior

On startup, the agent must:

1. load platform state
2. load installed crates
3. inspect actual runtime state
4. restore desired active crates
5. re-register integrations (proxy, firewall, storage mappings)
6. verify health

This is what makes CrateOS recover after reboot without users redoing manual Linux tasks.

---

## First implementation scope

To keep the first implementation realistic, the initial agent should support:

* full reconcile
* single crate reconcile
* explicit install/configure/enable/disable/remove
* boot reconcile
* periodic drift check
* ordered execution
* systemd crates first
* basic docker crates later
* health checks
* state persistence

Defer initially:

* full parallel reconcile
* complex distributed locks
* cluster-wide state
* advanced rollback orchestration

---

## Summary

The CrateOS agent reconcile loop is the control plane that makes the platform real.

It:

* reads desired state
* observes actual state
* computes safe transitions
* applies them through runtime and module hooks
* validates health
* persists state
* repairs drift over time

Without the agent, CrateOS is just a set of files and wrappers.
With the agent, it becomes a managed platform.
