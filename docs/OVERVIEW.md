# CrateOS Documentation Overview

This document exists to map the **broader CrateOS documentation set**.

Several core design documents already exist for CrateOS, including:

* `README.md`
* platform overview / architecture guide
* `MODULE_SPEC.md`
* `CRATE_STATE_MACHINE.md`
* `MODULE_REGISTRY.md`
* `CRATE_RUNTIME.md`
* `AGENT_RECONCILE.md`

Those documents define the core philosophy and mechanics of the platform.

This overview focuses on the **remaining documentation needed to make CrateOS actually buildable, understandable, and implementable** at a systems level.

It also captures broader concerns discussed during planning that are not fully covered in the current docs:

* concrete YAML / JSON structures
* filesystem and directory conventions
* hardware/software abstraction boundaries
* storage and partition behavior
* network and reverse proxy behavior
* runtime adapters and wrappers
* packaging/build pipeline expectations
* module authoring edge cases
* operator UX and panel behavior

---

## Current documented areas

The current documentation already covers these areas well enough to establish the platform direction:

### Product / platform direction

* what CrateOS is
* what it is not
* why it exists
* its panel-first / TUI-first philosophy

### Module model

* what a module is
* lifecycle verbs
* canonical paths
* manifest structure
* health checks
* hooks and adapters

### State behavior

* canonical lifecycle states
* stable vs transitional states
* transitions and invariants
* desired state vs actual state

### Runtime behavior

* runtime types
* runtime directory model
* runtime responsibilities
* runtime operations

### Registry behavior

* module discovery
* source types
* trust levels
* compatibility model
* local registry structure

### Agent behavior

* desired vs actual reconciliation
* planning phases
* drift repair
* retry logic
* boot reconcile

These are the **platform law** documents.

---

## Broader documentation still needed

The following documents should exist to complete the CrateOS design.

---

# 1. CRATE_CONFIG_SCHEMA.md

**Purpose:**
Define the exact config schema types used by modules and by the CrateOS panel/TUI.

### Why it matters

The module spec currently defines that modules have a `configSchema`, but it does not yet define the full typed UI/rendering contract.

### Should include

* primitive field types
* validation rules
* defaults
* nullable vs required behavior
* labels, descriptions, hints
* secret handling
* enum rendering
* nested object support
* repeatable/list structures
* conditional fields
* environment-variable backed values
* file/path selectors
* storage selector fields
* port selector fields

### Example content

```yaml
configSchema:
  - key: port
    type: port
    label: Port
    default: 5432
    required: true

  - key: listen_mode
    type: enum
    values: [local, lan, public]
    default: local

  - key: admin_password
    type: secret
    required: true
```

---

# 2. MODULE_AUTHORING_GUIDE.md

**Purpose:**
Explain how a developer creates a new CrateOS module from scratch.

### Why it matters

The spec defines what a module is, but not the practical workflow for authoring one.

### Should include

* folder layout for a module
* required files
* manifest rules
* when to use hooks
* when not to use hooks
* how to map native config/data/log paths
* how to define health checks
* how to define dependencies
* how to test a module locally
* how to package and publish it
* anti-patterns to avoid

### Example content

```text
modules/postgres/
  module.yaml
  templates/
  hooks/
  assets/
  README.md
```

---

# 3. PLATFORM_FILESYSTEM.md

**Purpose:**
Define the full CrateOS filesystem model beyond just per-module layout.

### Why it matters

CrateOS depends heavily on a canonical filesystem structure. This should be fully documented.

### Should include

* `/srv/crateos/` root tree
* config root
* services root
* logs root
* state root
* registry root
* export/symlink root
* cache/temp/runtime root
* ownership and permissions expectations
* what belongs under `/srv/crateos` vs native OS paths
* how symlinks/bind mounts are used

### Example content

```text
/srv/crateos/
  config/
  services/
  state/
  logs/
  registry/
  export/
  runtime/
  cache/
  backups/
```

---

# 4. STORAGE_AND_PARTITIONS.md

**Purpose:**
Define how CrateOS handles disks, partitions, mount points, and stateful service storage.

### Why it matters

Stateful modules like Postgres, MinIO, and Minecraft servers require a sane storage abstraction. This is one of the most important “anti-nerd Linux” features.

### Should include

* default storage model
* system disk vs service data disks
* mount discovery
* safe storage targets
* data path selection rules
* ownership/permission handling
* removable disk behavior
* missing storage recovery behavior
* backup/snapshot expectations
* future ZFS / btrfs / LVM considerations

### Example questions it should answer

* How does a user choose a data disk for Postgres?
* What happens if that disk is missing on boot?
* Can a crate migrate its storage path later?
* When does CrateOS use symlinks vs bind mounts?

---

# 5. NETWORK_AND_PROXY_MODEL.md

**Purpose:**
Define the full network model for CrateOS.

### Why it matters

Networking is a platform-level feature, not just an application detail.

### Should include

* LAN/WIFI behavior
* MAC-based network profile binding
* route priority rules
* firewall ownership model
* reverse proxy registration model
* internal vs external ports
* service exposure model
* domain and TLS flow
* cloudflared integration
* WireGuard integration
* health-aware proxy routing

### Example content

```yaml
network:
  lan:
    metric: 100
  wifi:
    metric: 600

proxy:
  enabled: true
  host: db.example.local
  target: postgres
```

---

# 6. RUNTIME_ADAPTERS.md

**Purpose:**
Define how CrateOS adapters/wrappers should be written for ugly Linux-native software.

### Why it matters

Wrappers are inevitable. Without rules, they become chaos.

### Should include

* adapter purpose
* adapter boundaries
* when a hook is acceptable
* when a dedicated adapter is required
* idempotency rules
* path remapping rules
* service ownership rules
* config render rules
* validation rules
* failure/rollback expectations

### Should explicitly cover

* Postgres path remap example
* Nginx config test + reload example
* Redis persistence example
* software that insists on native paths

---

# 7. HARDWARE_ABSTRACTION.md

**Purpose:**
Define how CrateOS sees hardware and exposes it to the panel.

### Why it matters

CrateOS wants to remove Linux trivia from the user experience, which means hardware visibility has to be normalized too.

### Should include

* CPU / RAM reporting
* temperature sensors
* fan sensors and fancontrol integration
* HDD/SSD/NVMe health
* mount and disk status
* NIC discovery
* Wi-Fi capabilities
* PCI / USB inventory
* optional GPU reporting
* hardware alerting model
* degraded hardware states

### Example JSON structure

```json
{
  "cpu": { "usage": 12, "tempC": 54 },
  "memory": { "usedMB": 4096, "totalMB": 32768 },
  "disks": [
    { "name": "nvme0n1", "health": "ok", "tempC": 41 }
  ],
  "network": [
    { "name": "lan0", "state": "up", "ip": "10.0.0.10" }
  ]
}
```

---

# 8. SOFTWARE_BASELINE.md

**Purpose:**
Define the base software included with CrateOS and the rationale for each component.

### Why it matters

The project already has a curated package list, but this should be promoted into a durable baseline document.

### Should include

* required base packages
* optional packages
* service ownership rules
* what is considered first-class
* what is intentionally excluded
* why nftables is preferred
* why tmux is primary
* why LightDM is optional
* why Cockpit is optional
* why cloudflared is bundled

---

# 9. BUILD_AND_IMAGE_PIPELINE.md

**Purpose:**
Define how CrateOS is built into artifacts.

### Why it matters

CrateOS is not just source code. It is a product delivered as:

* ISO
* qcow2
* Raspberry Pi image
* packages

### Should include

* repo structure for build targets
* build environments (WSL2, Linux, CI)
* Debian packaging
* autoinstall ISO flow
* qcow2 flow
* Pi image flow
* online vs offline image builds
* release artifacts
* signing/checksum expectations
* Windows developer workflow

---

# 10. PANEL_UX_MODEL.md

**Purpose:**
Define the TUI/panel behavior and user interaction model.

### Why it matters

CrateOS is explicitly not shell-first. The panel is not just cosmetic — it is the product.

### Should include

* navigation model
* module install flow
* config editing flow
* health/error display rules
* warning display rules
* permissions model
* break-glass shell flow
* logs browsing flow
* cleanup UI flow
* boot dashboard behavior
* local console vs SSH behavior

---

# 11. PERMISSIONS_AND_ROLES.md

**Purpose:**
Define how CrateOS handles users, roles, and action permissions.

### Why it matters

The project already discussed a Bukkit-like permission mindset. This needs a real contract.

### Should include

* user roles
* admin vs operator vs viewer
* break-glass shell access
* module-level permissions
* panel-only actions
* API permissions
* trusted local admin assumptions

---

# 12. STATE_AND_EVENT_MODEL.md

**Purpose:**
Define the structured JSON state and event objects used across the platform.

### Why it matters

The docs currently show examples in several places, but not one single source of truth for machine-readable objects.

### Should include

* module state object
* global platform state object
* event log object
* action result object
* reconcile plan object
* error object
* health object
* hardware object
* network object

### Example content

```json
{
  "event": "crate.start",
  "crate": "postgres",
  "status": "success",
  "timestamp": "2026-03-07T12:00:00Z"
}
```

---

# 13. EXAMPLE_CRATES.md

**Purpose:**
Provide fully worked examples of real crates.

### Why it matters

Specs are not enough. Real examples anchor the whole ecosystem.

### Recommended examples

* Postgres
* Nginx
* Redis
* Gitea
* Minecraft Server
* Cloudflared

For each example, show:

* manifest
* config schema
* paths
* health checks
* install mode
* runtime type
* state transitions
* rendered config outputs

---

# 14. ERROR_AND_RECOVERY_MODEL.md

**Purpose:**
Define how CrateOS surfaces errors and what recovery paths exist.

### Why it matters

A platform is judged by failure behavior more than happy path behavior.

### Should include

* transient vs hard vs unsupported errors
* repair flow
* retry flow
* rollback flow
* degraded mode rules
* storage missing behavior
* dependency missing behavior
* invalid config behavior

---

# 15. TESTING_AND_VALIDATION.md

**Purpose:**
Define how CrateOS itself and its modules are tested.

### Why it matters

This project will become wrapper hell unless module behavior is testable.

### Should include

* module validation rules
* manifest schema validation
* config schema validation
* health check tests
* integration tests in qcow2/VM
* boot-time acceptance tests
* image build smoke tests

---

## Documentation priority order

Recommended order for writing the remaining docs:

1. `CRATE_CONFIG_SCHEMA.md`
2. `PLATFORM_FILESYSTEM.md`
3. `STORAGE_AND_PARTITIONS.md`
4. `STATE_AND_EVENT_MODEL.md`
5. `MODULE_AUTHORING_GUIDE.md`
6. `NETWORK_AND_PROXY_MODEL.md`
7. `RUNTIME_ADAPTERS.md`
8. `PANEL_UX_MODEL.md`
9. `BUILD_AND_IMAGE_PIPELINE.md`
10. `TESTING_AND_VALIDATION.md`
11. `HARDWARE_ABSTRACTION.md`
12. `ERROR_AND_RECOVERY_MODEL.md`
13. `SOFTWARE_BASELINE.md`
14. `PERMISSIONS_AND_ROLES.md`
15. `EXAMPLE_CRATES.md`

---

## Practical note

CrateOS will only feel coherent if the documentation is coherent.

The project is not just defining modules and states.
It is defining an entire normalized operating model for Linux services, hardware, storage, networking, and user interaction.

That means examples, schemas, and filesystem structures matter just as much as philosophy.

The goal is not to write “nice docs.”
The goal is to make CrateOS implementable without falling back into random Linux tribal knowledge.

---

## Summary

This overview identifies the broader documentation needed to complete the CrateOS design.

The current docs establish the platform law.
The remaining docs should establish:

* concrete schemas
* concrete filesystem structures
* concrete runtime/storage/network behavior
* concrete hardware/software integration
* concrete authoring and recovery rules

That is the documentation layer required to turn CrateOS from a strong idea into a buildable platform.
