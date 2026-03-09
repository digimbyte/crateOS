# CrateOS Module Specification

This document defines the **CrateOS module model**.

A module is not the software itself. A module is the **managed definition** for how CrateOS installs, configures, runs, monitors, updates, and removes a service or capability.

CrateOS modules exist to normalize Linux service chaos into a predictable, panel-first shape.

---

## Core principle

Users should think in terms of:

* install module
* configure module
* enable module
* manage module
* remove module

Users should **not** need to think in terms of:

* apt install
* systemctl
* native log paths
* native config file locations
* package-specific daemon trivia

A module is a **managed object** inside CrateOS.

---

## Design goals

* **Panel-first UX**: the same management verbs work across all modules.
* **Predictable layout**: all modules expose canonical config/data/log paths under `/srv/crateos/services/<module>/`.
* **Idempotent operations**: install/configure/apply/remove may be run repeatedly without breaking state.
* **Declarative first**: manifests and typed config drive behavior; hooks exist only where necessary.
* **Backend flexibility**: install may be immediate, staged, or lazy without changing the UX.
* **Health-aware**: every module must define at least one health check.

---

## Module lifecycle

Every module should move through a standard lifecycle:

1. **Available** — module exists in the registry/catalog
2. **Installed** — module definition is active on the machine; assets/packages may or may not be fully installed yet
3. **Configured** — required configuration is valid
4. **Enabled** — desired state says the module should be running
5. **Running** — health checks pass
6. **Degraded** — partially working or failing health checks
7. **Disabled** — installed but not desired to run
8. **Removed** — no longer installed or managed

### Notes

* A module may be **Installed** but not yet **Configured**.
* A module may be **Configured** but not yet **Enabled**.
* A module may be **Enabled** but not yet **Running** if health checks fail.

---

## Management verbs

Every first-class module must support the same user-facing verbs:

* **install**
* **configure**
* **enable**
* **disable**
* **restart**
* **update**
* **remove**

Optional verbs:

* **backup**
* **restore**
* **reset**
* **migrate**
* **repair**

These are the only verbs the panel/TUI should expose by default.

---

## Canonical module layout

Every managed module should expose a predictable structure under:

```text
/srv/crateos/services/<module>/
  module.yaml         # resolved/installed module definition
  overrides.yaml      # local configuration overrides
  state.json          # current module state
  config/             # canonical config view
  data/               # canonical data view
  logs/               # canonical log view
  runtime/            # sockets, pids, temp/runtime metadata (optional)
  backups/            # module-specific backups (optional)
```

### Important

The canonical layout does **not** require the native application to store files directly there.
CrateOS may use:

* symlinks
* bind mounts
* generated views
* adapters/hooks

The user-facing model remains the same regardless.

---

## Native vs canonical paths

Linux applications often scatter files across the OS.
CrateOS normalizes that.

Example:

```text
Native app paths:
  /etc/postgresql
  /var/lib/postgresql
  /var/log/postgresql

CrateOS canonical paths:
  /srv/crateos/services/postgres/config
  /srv/crateos/services/postgres/data
  /srv/crateos/services/postgres/logs
```

Modules must explicitly declare both path sets.

---

## Module structure

A module has three conceptual layers:

### 1. Manifest

User-facing metadata and typed capabilities.

### 2. Runtime definition

System-facing install/runtime wiring.

### 3. State

Machine-tracked current state.

---

## Module manifest schema

Example shape:

```yaml
apiVersion: crateos/v1
kind: Module

metadata:
  id: postgres
  name: PostgreSQL
  category: database
  version: "16"
  description: Managed PostgreSQL database service
  icon: postgres
  tags: [database, sql, stateful]

spec:
  capabilities:
    - stateful
    - networked
    - backup-aware
    - requires-storage

  installMode: staged
  runtimeType: systemd
  dependencies: []

  packages:
    - postgresql
    - postgresql-contrib

  units:
    - postgresql.service

  paths:
    config:
      canonical: /srv/crateos/services/postgres/config
      native: /etc/postgresql
    data:
      canonical: /srv/crateos/services/postgres/data
      native: /var/lib/postgresql
    logs:
      canonical: /srv/crateos/services/postgres/logs
      native: /var/log/postgresql

  network:
    ports:
      - name: postgres
        port: 5432
        protocol: tcp

  configSchema:
    - key: port
      type: number
      default: 5432
      required: true

    - key: listen_mode
      type: enum
      values: [local, lan, public]
      default: local
      required: true

    - key: storage_target
      type: path
      required: true

    - key: backup_profile
      type: string
      required: false

  healthChecks:
    - type: tcp
      port: 5432

    - type: command
      command: pg_isready

  lifecycle:
    install:
      hook: hooks/install-postgres
    configure:
      hook: hooks/configure-postgres
    remove:
      hook: hooks/remove-postgres
```

---

## Required top-level fields

### `apiVersion`

Schema version.

### `kind`

Must be `Module`.

### `metadata.id`

Stable, unique module ID.
Examples:

* `postgres`
* `nginx`
* `redis`

### `metadata.name`

Human-readable name.

### `metadata.category`

Suggested categories:

* `database`
* `proxy`
* `cache`
* `storage`
* `network`
* `app`
* `platform`
* `utility`

### `spec.installMode`

Defines backend install behavior.
Allowed values:

* `immediate`
* `staged`
* `lazy`

### `spec.runtimeType`

Defines primary runtime model.
Allowed values:

* `systemd`
* `container`
* `hybrid`
* `binary`

---

## Install modes

### `immediate`

Install packages/assets at module install time.

Good for:

* small utilities
* low-risk modules

### `staged`

Register module first, apply installation after required config is valid.

Good for:

* databases
* apps requiring setup
* storage-sensitive services

### `lazy`

Only install when enabled.

Good for:

* heavy modules
* optional bundles

---

## Capabilities

Modules should declare capabilities so the panel can adapt automatically.

Suggested capabilities:

* `networked`
* `stateful`
* `web-ui`
* `database`
* `proxy-target`
* `backup-aware`
* `containerized`
* `requires-storage`
* `requires-domain`
* `requires-auth`
* `supports-tls`
* `supports-cluster`

Capabilities are descriptive, not imperative.

---

## Config schema

Each module must define a typed config schema for panel/TUI rendering.

Supported field types (initial set):

* `string`
* `number`
* `boolean`
* `enum`
* `path`
* `port`
* `size`
* `duration`
* `secret`
* `list`

Each field should support:

* `key`
* `type`
* `label` (optional, recommended)
* `description` (optional)
* `default` (optional)
* `required`
* `values` (for enums)
* `validation` (future)

---

## Health checks

Every module must define at least one health check.

Supported health check types (initial set):

* `tcp`
* `http`
* `command`
* `file`
* `process`

Examples:

```yaml
healthChecks:
  - type: tcp
    port: 5432
```

```yaml
healthChecks:
  - type: http
    url: http://127.0.0.1:8080/health
    expect: 200
```

```yaml
healthChecks:
  - type: command
    command: pg_isready
```

---

## Lifecycle hooks

Hooks are allowed, but they are **not** the module.

They are adapters for software that cannot be normalized purely through declarative metadata.

Supported hook phases:

* `install`
* `configure`
* `enable`
* `disable`
* `update`
* `remove`
* `repair`
* `backup`
* `restore`

### Hook rules

* Hooks must be idempotent.
* Hooks must have clear input/output expectations.
* Hooks should be minimal and versioned.
* Hooks should not replace the manifest as the primary source of truth.

---

## Module state tracking

CrateOS should track global module state in:

```text
/srv/crateos/state/modules.json
```

Example:

```json
{
  "postgres": {
    "installed": true,
    "configured": true,
    "enabled": true,
    "running": true,
    "health": "ok",
    "version": "16",
    "runtimeType": "systemd",
    "lastApply": "2026-03-07T12:00:00Z"
  }
}
```

Each module should also have local state:

```text
/srv/crateos/services/<module>/state.json
```

---

## Storage semantics

Stateful modules must define storage requirements clearly.

Important fields may include:

* storage target/path
* persistent vs ephemeral
* backup requirements
* migration sensitivity
* default ownership/permissions

Examples of modules needing strong storage semantics:

* PostgreSQL
* MariaDB
* Redis (optional persistence)
* MinIO
* Minecraft server

CrateOS should expose storage selection during module configuration instead of forcing users to manually move paths after install.

---

## Dependencies

Modules may depend on other modules or platform capabilities.

Examples:

* an app may depend on `postgres`
* a proxy target may depend on `nginx`
* a web app may require `requires-domain`

Dependency resolution should be explicit and visible in the UI.

---

## Reference modules

The first three modules recommended for CrateOS validation are:

### 1. Nginx

Tests:

* reverse proxy
* template rendering
* reload validation
* domain/TLS-ready structure

### 2. PostgreSQL

Tests:

* stateful storage
* config/data/log normalization
* path remap
* health checks

### 3. Redis

Tests:

* simpler stateful service
* persistence toggle
* port and health handling

If the schema works for these three, it is likely strong enough for broader use.

---

## Non-goals

Modules should **not** become:

* random collections of shell scripts
* undocumented one-off hacks
* package-manager wrappers with no state model
* app-specific exceptions that leak Linux complexity to the user

If a module cannot fit the CrateOS model cleanly, it is not ready to be first-class.

---

## Summary

CrateOS modules turn Linux services into **managed crates with predictable shape, lifecycle, and control**.

A good module:

* has typed metadata
* has a standard lifecycle
* has canonical config/data/log paths
* declares health checks
* uses hooks only as adapters
* behaves consistently inside the panel/TUI

This specification is the contract that keeps CrateOS a platform instead of a pile of wrappers.
