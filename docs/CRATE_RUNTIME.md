# CrateOS Crate Runtime

This document defines how **installed crates actually run on the system**.

The runtime layer is responsible for turning a crate definition + configuration + state into a **live service** on the machine.

The registry tells CrateOS *what exists*.
The state machine defines *lifecycle behavior*.
The runtime defines **how services execute and are supervised**.

---

# Runtime philosophy

CrateOS does not attempt to replace Linux primitives.

Instead it **standardizes how crates interact with them** so the user sees one consistent model.

Crates may internally use:

* systemd services
* containers (Docker)
* background daemons
* scheduled jobs
* hybrid runtimes

But to the user they are always seen as a **Crate** with a predictable lifecycle.

---

# Supported runtime types

Every crate declares a runtime type.

Example:

```yaml
runtime:
  type: systemd
```

Supported runtimes:

### systemd

Used for native Linux services.

Examples:

* nginx
* postgres
* redis
* ssh

The crate runtime will:

* install or generate a systemd unit
* manage enable/disable
* manage start/stop
* track service health

---

### docker

Used for containerized applications.

Examples:

* web apps
* microservices
* dashboards

Example:

```yaml
runtime:
  type: docker
  compose: docker-compose.yaml
```

CrateOS will:

* generate docker compose configuration
* mount storage paths
* start containers
* manage lifecycle

---

### hybrid

Some crates may use both systemd and containers.

Example:

* postgres + backup container
* nginx + sidecar services

Hybrid crates define multiple runtime components.

---

### task

For one-shot or scheduled jobs.

Examples:

* backups
* cleanup jobs
* maintenance routines

These may map to:

* systemd timers
* cron jobs

CrateOS should prefer managed systemd timers/services over raw cron where possible so timeout, overlap, actor identity, and termination behavior stay inside the control plane.

---

# Runtime directory model

All runtime data must live under the crate service root.

Example:

```
/srv/crateos/services/postgres/

  config/
  data/
  logs/
  runtime/
  backups/
  state.json
```

CrateOS should avoid scattering runtime files across the OS where possible.

When native services require default paths, CrateOS should map them back into the crate root.

Example:

```
/var/lib/postgresql -> /srv/crateos/services/postgres/data
```

---

# Runtime components

A crate runtime consists of four logical parts.

### 1. Configuration

User-defined configuration stored under:

```
/srv/crateos/services/<crate>/config/
```

These values are used to render templates.

---

### 2. Templates

Templates define how configuration becomes runtime configuration.

Example:

```
templates/postgres.conf.tpl
```

The runtime engine renders templates into:

```
runtime/
```

---

### 3. Execution

Execution defines how the service actually starts.

Examples:

* systemd unit
* docker compose
* command invocation

Execution policy should also define:

* which CrateOS-managed actor runs the workload
* whether the workload is a long-lived service or a timed job
* timeout / stop-timeout limits
* what happens on timeout
* overlap/concurrency rules for scheduled work

---

### 4. Health

Health checks determine if a crate is running correctly.

Examples:

* TCP port check
* HTTP endpoint
* CLI readiness command

Health feeds the state machine (Running / Degraded).

---

# Runtime operations

The runtime must support the following operations.

### start

Start crate runtime.

Examples:

* systemctl start
* docker compose up

---

### stop

Stop runtime processes.

---

### restart

Composite operation:

```
stop -> start
```

---

### reload

Optional soft reload.

Examples:

* nginx reload

---

### status

Return health and runtime status.

---

### logs

Expose crate logs from canonical log location.

---

# Networking integration

Crate runtime integrates with:

* CrateOS network config
* reverse proxy
* firewall

Crates should declare required ports.

Example:

```yaml
network:
  ports:
    - 5432
```

CrateOS can automatically:

* expose ports internally
* register reverse proxy
* apply firewall rules

---

# Storage integration

Crates may request storage types.

Example:

```yaml
storage:
  type: persistent
  size: dynamic
```

CrateOS maps storage to:

```
/srv/crateos/services/<crate>/data
```

Optional advanced features later:

* disk allocation
* snapshot support
* backup scheduling

---

# Logging model

All crates must expose logs through a canonical path.

Example:

```
/srv/crateos/services/<crate>/logs
```

CrateOS may also export logs to:

```
/srv/crateos/logs
```

Logrotate should manage size.

---

# Resource awareness

Future runtime versions may track:

* CPU usage
* memory usage
* disk consumption
* network throughput

These metrics feed the panel UI.

---

# Runtime abstraction layer

The runtime layer should expose a unified API internally.

Example operations:

```
runtime.start(crate)
runtime.stop(crate)
runtime.restart(crate)
runtime.status(crate)
runtime.logs(crate)
```

Internally these dispatch to runtime-specific handlers.

Example:

```
systemd.start()
docker.start()
task.start()
```

---

# Runtime responsibilities

The runtime layer is responsible for:

* executing crate workloads
* mapping config to runtime
* exposing logs
* exposing health
* integrating with networking
* integrating with storage

It should **not** decide policy.

Policy decisions belong to the **agent layer**.

---

# Non-goals

The runtime should not:

* act as a package manager
* manage registry logic
* define module lifecycle
* own system policy

Those belong to other CrateOS components.

---

# Summary

The crate runtime is the execution engine for modules.

It translates crate definitions into real Linux services while preserving a unified operational model.

Users should experience:

* predictable service behavior
* predictable filesystem layout
* predictable lifecycle operations

No matter what software is running underneath.
