# CrateOS

CrateOS is a curated, appliance-style server experience built on **Ubuntu Server LTS**. It presents a **single enforced control surface** (a retro “Pitboy/DOS choose‑your‑adventure” CrateOS console and optional web panel) and a **single canonical filesystem root** for configuration, services, logs, and state.

**Goal:** provide a robust “it just works” server platform with predictable defaults, self-healing policies, and a clean UX—while still allowing advanced users to “pop the hood” deliberately.

---

## Design principles

* **One root directory** users care about (Minecraft modpack vibe).
* **One supported control plane** (CrateOS console + API). No memorizing thousands of CLI flags.
* **Declarative configs** live under the Crate root; the OS is treated as an implementation detail.
* **Idempotent apply**: changes are applied via a controller/agent and can be re-applied safely.
* **Guardrails, not a prison**: normal access is curated; admin escape hatches exist by intent.
* **Modular services**: microservice “crates” can be enabled/disabled/installed/uninstalled cleanly.
* **Predictable networking**: system-level, headless-safe profiles; MAC-based matching.
* **Multi-tenant by default**: multiple users log in and manage the system through roles and service-scoped permissions.

---

## What ships with CrateOS

### Core platform

* Ubuntu Server LTS base
* OpenSSH (curated login flow)
* systemd (service management under a Crate control plane)
* NetworkManager (policy-enforced, headless-safe)
* nftables firewall (Crate-managed rulesets)

### Developer & ops tooling

* **Node Version Manager** (nvm) and pinned default Node LTS (with opt-in per-project versions)
* **C build toolchain** (build-essential, pkg-config, etc.)
* **Python** (Python 3 + venv tooling + pip)
* **Git**
* **PowerShell**
* **tmux** (primary) + **screen** (optional secondary)
* **Docker Engine** + **docker compose** plugin
* **Nginx** reverse proxy + templates

### Essential platform utilities (robust, low-bloat)

* **chrony** (reliable time sync)
* **unattended-upgrades** + **needrestart** (security patching + restart hygiene)
* **btop** (modern system monitor)
* **ncdu** (disk usage triage)
* **smartmontools** + **nvme-cli** (disk/NVMe health)
* **lm-sensors** + **fancontrol** (temps + fan tuning)
* **hddtemp** (optional legacy HDD temp readings)
* **iperf3** (network throughput testing)
* **ethtool** + **iw** (link diagnostics/tuning)
* **dnsutils** + **mtr-tiny** (network diagnostics)
* **pciutils** + **usbutils** (hardware inventory)
* **lsof** (who owns ports/files)
* **jq** + **yq** (JSON/YAML tooling)
* **rsync** (reliable sync/transfer)
* **dos2unix** (DOS→Unix line ending conversion)
* **logrotate** (keeps exported logs sane)
* **fail2ban** (SSH brute-force noise control)
* **WireGuard** (clean admin VPN)
* **msmtp** (send-only SMTP relay for alerts; not a full mail server)
* **cloudflared** (Cloudflare Tunnel client; managed as a Crate module)

### Optional UX & web admin

* **LightDM** (optional CrateOS-controlled local login/session host)
* **Cockpit** (optional lightweight web admin UI)

---

## UX changes vs stock Ubuntu Server

### SSH and interactive access

* Default SSH entry lands in the **Crate Console** rather than a raw shell.
* Local operator login should land in the same CrateOS-owned control surface rather than a stock Ubuntu shell session.
* Raw shell access is treated as **break-glass** and is only for authenticated admin-authorized use.
* Any local GUI or future virtual desktop surface should land inside a **CrateOS-owned session/workspace**, not a normal distro desktop.
* The console provides:

  * System status dashboard
  * Service enable/disable/start/stop/restart
  * User management + roles/permissions
  * Logs browse/search/export
  * Network setup + repair
  * Firewall templates
  * Updates & backups
  * Module install/uninstall

### “No guessing commands”

* CrateOS exposes common workflows as guided actions:

  * “Enable reverse proxy for service X”
  * “Create site mapping / TLS policy”
  * “Pin LAN/WIFI by MAC and set routing priority”
  * “Install service module Y”
  * “Clean logs/cache”

---

## Canonical filesystem layout

**Single root:** `/srv/crateos` (configurable, but defaults here)

```text
/srv/crateos/
  config/                 # the only human-edited configs
    crateos.yaml           # global platform config
    network.yaml           # LAN/WIFI policies, metrics, allow/deny
    firewall.yaml          # nft templates and toggles
    services.yaml          # enabled modules + options
    users.yaml             # roles and permissions
    reverse-proxy.yaml     # nginx mappings, hostnames, policies

  modules/                # module definitions (like “mod metadata”)
    nginx.module.yaml
    docker.module.yaml
    postgres.module.yaml
    cloudflared.module.yaml

  services/               # each service lives like a crate/mod
    nginx/
      config/
      data/
      logs/
      module.yaml          # resolved module config
    app-foo/
      config/
      data/
      logs/

  state/                  # desired/actual state, snapshots, last-good
    desired.json
    actual.json
    last-good/
    backups/

  logs/                   # exported/curated logs
    boot.log
    net.log
    fw.log
    services.log

  export/                 # curated view (symlink farm) to OS internals
    etc/
    var/

  bin/                    # platform tools
    crateos               # CLI/TUI entry
    crateos-agent         # apply/enforcer service
```

**Rule:** users interact with `/srv/crateos/**` only. OS paths remain accessible but are considered “under the hood.”
Managed config writes made by CrateOS are tracked as monitored changes; edits that appear outside CrateOS write paths are recorded as unmonitored config changes so operator/state surfaces can distinguish external drift from CrateOS-owned mutations.

---
## Installation model

CrateOS is **installed as part of the OS install**. The CrateOS ISO installer is a forced CrateOS framework install (not a generic Ubuntu installer).

* No in-place “upgrade from vanilla Ubuntu” path
* Images are built and installed as **CrateOS-first** systems
* Upgrades are handled by new images/releases, not by in-place conversion or rollback
* ISO install completion now depends on seeded config presence and persistent first-boot unit enablement; post-install verification is executed with `/usr/local/bin/verify-mvp-install`
* ISO media rebuild preserves the base Ubuntu boot model by replaying source ISO boot metadata and refreshing media checksums after mutation
* Fresh installs promote the installer-created machine user into the initial CrateOS admin operator; if operator state is missing later, recover locally with `crateos bootstrap <name>`
* Fresh installs stamp local-console takeover directly into the target rootfs: `tty1` autologins the seeded operator, the seeded operator shell is `/usr/local/bin/crateos-login-shell`, and that login shell `exec`s `crateos console`
* Agent liveness is supervised by both systemd restart policy and `crateos-agent-watchdog.timer`, which logs and retries recovery if the agent or its socket falls out of service after boot
* Machine-readiness now expects the live agent socket plus rendered platform/watchdog state artifacts, not just enabled units and seeded files
* Those runtime state artifacts must also stay fresh enough to prove the control plane is still updating after boot
* Readiness is summarized into a single generated report under `/srv/crateos/state/readiness-report.json` so verification and operator surfaces agree on why a machine is degraded
* `crateos-policy.timer` refreshes that readiness report every 2 minutes after boot, and installed-host verification treats the report as stale after 3 minutes
* The TUI also treats a stale readiness report as degraded, so operator surfaces do not keep showing an old historical `ready` state after policy freshness is lost
* Operator-facing platform adapter status also degrades when `platform-state.json` is stale, so rendered host posture cannot look current after agent updates stop
* Operator-facing service posture also degrades when stored `crate-state.json` snapshots are stale, so service views do not present old agent renders as current runtime truth

---

## Service model (not hosted)

CrateOS is a **self-hosted software package**, not a live managed service. There is **no external SLA**—reliability, uptime, and backups are owned by the operator and expressed through CrateOS tooling and policies.

---

## Permissions & roles (baseline schema)

CrateOS uses **fine-grained permission nodes** (Discord/Minecraft style). Deny by default; allow explicitly.

### Global permissions

* `sys.view`
* `sys.manage`
* `users.view`
* `users.*`
* `roles.view`
* `roles.create`
* `roles.edit`
* `roles.delete`
* `audit.view`
* `logs.view`
* `shell.breakglass`
* `net.status`
* `net.configure`
* `net.*`
* `proxy.*`
* `updates.view`
* `updates.apply`
* `backups.view`
* `backups.run`
* `modules.*`

### Service-scoped permissions

* `svc.<service>.view`
* `svc.<service>.edit`
* `svc.<service>.start`
* `svc.<service>.stop`
* `svc.<service>.restart`
* `svc.<service>.logs.view`
* `svc.<service>.config.view`
* `svc.<service>.config.edit`
* `svc.<service>.plugins.view`
* `svc.<service>.plugins.add`
* `svc.<service>.plugins.remove`
* `svc.<service>.plugins.configure`
* `svc.<service>.data.backup`
* `svc.<service>.data.restore`

### Role templates (starter set)

* **Owner**: `sys.*`, `users.*`, `roles.*`, `audit.view`, `shell.breakglass`, `net.*`, `proxy.*`, `updates.*`, `backups.*`, `modules.*`, all `svc.*`
* **Admin**: `sys.view`, `users.view`, `roles.view`, `net.*`, `proxy.*`, `updates.apply`, `backups.run`, `modules.*`, all `svc.*` (no break-glass)
* **Service Admin (scoped)**: all `svc.<service>.*`
* **Operator**: `svc.<service>.view|start|stop|restart|logs.view`
* **Auditor**: `audit.view`, `logs.view`, `svc.<service>.logs.view`

### Policy rules

* Deny by default
* Break-glass requires explicit permission and logs a security event
* Service-scoped roles never grant global permissions

### Access/session posture

CrateOS treats login surfaces as controlled appliance entry points.

* `crateos.yaml` models this under `access`
* SSH should remain enabled and land in `console`
* Local GUI currently assumes a LightDM-hosted CrateOS session when enabled
* Virtual desktop is intended to host the same CrateOS workspace/panel model rather than a generic distro desktop
* Break-glass remains explicit, permission-gated, and limited to configured entry surfaces

---

## Release & publishing flow (draft)

CrateOS uses **GitHub as the source of truth** for releases.

* Build artifacts: `.deb`, `.iso`, `.qcow2` (later: Pi image)
* GitHub Releases host signed checksums and versioned artifacts
* Update channels can be introduced later (stable/beta/nightly), if desired

---
## Package policy (required vs optional)

CrateOS splits packages into **required** (always installed at OS install time) and **optional/opt-in** (installed when a module/service is enabled).

* Source of truth: `packaging/config/packages.yaml`
* Autoinstall ISO installs **required** only
* Optional packages map to service modules (databases, tunnels, admin UI, etc.)

---

## Control plane & management

### Components

* **crateos**: primary operator interface and control surface
* **crateos-agent**: local daemon (root) that applies desired state idempotently
* **crateos-policy**: periodic/boot-time drift detection and repair
* **optional web panel**: talks to agent via local socket/API

### State model

* `config/*.yaml` → parsed into **Desired State**
* agent probes system → **Actual State**
* agent computes diff → applies
* writes:

  * `/srv/crateos/state/desired.json`
  * `/srv/crateos/state/actual.json`
  * `/srv/crateos/state/last-good/*` snapshot before changes

### Permissions model (Bukkit-style)

* Role/group-based permissions:
  * `sys.view`
  * `sys.manage`
  * `users.*`

  * `net.*`
  * `svc.*`
  * `svc.<service>.view|edit|start|stop`
  * `svc.<service>.plugins.add|remove|configure`
  * `proxy.*`
  * `logs.view`
  * `modules.*`
  * `shell.breakglass`
* Default users land in console with only safe actions.
* Admin users can unlock break-glass shell.
* Permissions are assigned per user and can be scoped per service/module.

---

## System links & integration points

CrateOS intentionally provides a **curated export view** of scattered Linux internals, routing known paths into a clean, uniform structure under the Crate root.

### Export/symlink farm examples

* `/srv/crateos/export/etc/NetworkManager` → `/etc/NetworkManager`
* `/srv/crateos/export/etc/nginx` → `/etc/nginx`
* `/srv/crateos/export/etc/nftables.conf` → `/etc/nftables.conf`
* `/srv/crateos/export/var/log/journal` → `/var/log/journal`

### Curated log exports

CrateOS generates readable flat logs under `/srv/crateos/logs/` from journald.
Examples:

* `net.log` from `journalctl -u NetworkManager -b`
* `services.log` from key units
* `boot.log` from `journalctl -b`

---

## Networking policy (headless-safe)

* Enforce **system-owned** NetworkManager connections (no user keyring dependency).
* Bind NIC profiles by **MAC address**, not `enpX` naming.
* Define route preference (LAN primary, WIFI secondary) via metrics.

Example policy intent:

* Create/repair `LAN` profile pinned to ethernet NIC MAC.
* Create/repair `WIFI` profile pinned to wifi NIC (or interface) and ensure secrets stored for boot.
* Apply metrics: `LAN=100`, `WIFI=600`.

---

## Reverse proxy (nginx)

CrateOS ships nginx plus templates and a managed mapping file.

Features:

* HTTP services registry: route `host/path → target service`.
* Optional TLS integration (future: ACME automation).
* Health checks and port conflict warnings.
* Safe reloads (`nginx -t` before apply).

---

## Services & modules

### Module lifecycle

* **Install**: apt/packages + directories + config skeleton
* **Enable**: systemd enable + start
* **Disable**: stop + disable (keep installed)
* **Uninstall**: stop + purge (optional) + remove Crate directories

### Runtime choices

* Primary: **systemd-native** services where appropriate
* Optional: containerized services using **Docker Compose**

CrateOS supports both, but **exposes a single interface** for the user.

### Managed actors and execution policy

CrateOS should treat hosted services and scheduled jobs as managed workloads with their own execution actors.

* each service/module/job should run under a CrateOS-managed actor identity rather than a human operator account
* uploads should land in a managed intake path first, then be rendered into the runnable service/job working directory
* operators should be able to classify an upload as:
  * a long-lived service
  * a scheduled job
* execution policy should include:
  * start command
  * schedule for jobs
  * timeout and stop-timeout
  * kill behavior on timeout
  * overlap policy for scheduled jobs

This is the basis for “upload files, then run them as a service or invoke them on a schedule” without dumping the operator into raw cron/systemd management.

---

## tmux vs screen

* Both can ship.
* CrateOS uses **tmux internally** for robustness and scripting, and may expose **screen** as optional fallback.
* Console sessions can attach/detach cleanly and survive disconnects.

---

## Data cleanup & maintenance

CrateOS exposes one-button actions:

* Vacuum journald (size/time-based)
* Prune docker images/volumes (policy-driven)
* Rotate/export logs
* Clean caches (apt, pip, npm)
* Service-level “factory reset” (wipe data + reinit) with backup prompt

---

## Storage posture

CrateOS now treats storage as a first-pass platform posture surface for MVP.

* The agent records storage posture under `/srv/crateos/state/storage-state.json`
* The Platform view surfaces whether the machine has only the system disk or also has safer mounted data targets
* Mounts under `/srv/*`, `/mnt/*`, or `/media/*` are treated as candidate data targets for stateful crates
* Module-backed crates surface their canonical/native data paths so operators can see where state actually lives

This is posture and visibility, not full disk orchestration yet.

---

## Roadmap notes

### MVP (first shippable)

* Directory layout + manifests
* TUI: status, diagnostics, logs, services, network
* Agent: apply idempotently (NM + systemd + nginx)
* Docker Compose support for a few modules

### Next

* Web UI panel
* ACME/TLS automation
* Role/permission UI
* Snapshot/rollback improvements
* Signed update channel for CrateOS packages

---

## MVP terminal command-lane acceptance checklist

Use this checklist to validate terminal-first behavior from the operator perspective.

Canonical command set for MVP:

* `help`
* `list <services|users|logs|sources|net|status|diagnostics>` (or `list` for current view)
* `nav <setup|menu|status|diagnostics|services|users|logs|network>`
* `status <system|services|platform|next|prev|select>`
* `svc <list|enable|start|stop|disable|install|uninstall|restart|next|prev|select> [service|service1,service2|all]`
* `<service> <enable|start|stop|disable|install|uninstall|restart>`
* `<enable|start|stop|disable|install|uninstall|restart> <service|service1,service2|all>`
* `user <list|add|rename|set|role|perms|delete|next|prev|select> [name|name1,name2]`
* `log <list|next|prev|select> [service|service1,service2]` and `log source <list|next|prev|select> [source|source1,source2]`
* `net <list|next|prev|select> [interface|iface1,iface2]`
* `diag <summary|verification|ownership|config|actor [target]|focus <target>|next|prev|select>`
* `list actors` while in Diagnostics, or `diag list actors`
* `bootstrap <admin>`
* `system refresh`
* `system dos2unix [config|services|all]`
* `system ftp-complete <path|dir>`
* `back`
* `quit`
* Setup lock: before bootstrap, non-bootstrap control-plane commands should return a setup-lock warning
* Event-driven newline policy: normalize only on write-complete events (FTP upload completion, web-form save, TUI save) for known text/config formats; do not broad-scan random host paths
* Lifecycle hooks: `internal/config/lineendings.go` exposes `OnFTPUploadComplete(path)`, `OnWebFormSave(path)`, and `OnTUISave(path)` to keep newline handling centralized
* FTP upload finalize API: `POST /uploads/ftp/complete` with body `{ "path": "<uploaded-file-or-dir>" }` triggers CRLF normalization on that target (single file or recursive directory walk) and returns status plus normalized/scanned counts

### A) Command mode fundamentals

* Enter command mode with `:`
* Type a command, press `enter`, confirm feedback appears in the command lane
* Press `esc` while in command mode, confirm command mode exits cleanly
* Type unknown input and confirm an explicit error response appears
* Run a chain such as `nav services; svc list` and confirm sequential execution
* Run a failing chain and confirm fail-fast abort occurs at the failing step
* Run a quoted command such as `user add \"ops lead\" admin` and confirm argument parsing remains intact
* Run a chain with quoted args and separators inside quotes to confirm quote-safe splitting

### B) Navigation coverage

* `nav menu`
* `nav status`
* `nav diagnostics`
* `nav services`
* `nav users`
* `nav logs`
* `nav network`
* `list`, `list services`, and `list diagnostics`
* Confirm each command routes to the expected panel

### C) Status module coverage

* `status system`
* `status services`
* `status platform`
* `status next`
* `status prev`
* Confirm section focus changes without leaving the status view

### C.1) Diagnostics coverage

* `diag summary`
* `diag verification`
* `diag ownership`
* `diag config`
* `diag list actors`
* `diag actor <crate>`
* `diag focus <crate|actor|user|id>`
* `diag next`
* `diag prev`
* Confirm the verification section reflects install prerequisites such as agent socket, admin operator presence, readiness state, and storage state rendering
* Confirm actor diagnostics can jump directly to a selected managed workload and that summary/detail windows move together as selection changes

### D) Service command coverage

* `svc next` / `svc prev`
* `svc list`
* `svc select <service>`
* `svc select <service1,service2>` and verify partial/success feedback is explicit
* `svc start <service>`
* `svc stop <service>`
* `svc enable <service>`
* `svc disable <service>`
* `svc restart <service1,service2>` and verify partial/success feedback is explicit
* `svc stop all` and verify all known services receive the action
* Confirm state and selection update in the Services panel

### E) User command coverage

* `user select <name>`
* `user list`
* `user set <name>`
* `user role <name>`
* `user perms <name>`
* `user delete <name>`
* `user role <name1,name2>` and verify partial/success feedback is explicit
* `user perms <name1,name2>` and verify partial/success feedback is explicit
* `user delete <name1,name2>` and verify partial/success feedback is explicit
* Confirm user list/current-user posture updates accordingly

### F) Logs and network coverage

* `log service next` / `log service prev`
* `log list` and `log source list`
* `log service select <service>`
* `log service select <service1,service2>` and verify partial/success feedback is explicit
* `log source select <source1,source2>` and verify partial/success feedback is explicit
* `log source next` / `log source prev`
* `net next` / `net prev`
* `net list`
* `net select <interface>`
* `net select <iface1,iface2>` and verify partial/success feedback is explicit
* Confirm selection focus updates in both views

### G) Global behavior

* `system refresh`
* `system dos2unix config` (or `all`) and verify normalization reports processed file count
* `system ftp-complete <path-to-crlf-text-file>` and verify `normalized` result is reported
* Run `system ftp-complete <same-path>` again and verify `skipped` result is reported when file is already LF-normalized
* `system ftp-complete <ftp-upload-directory>` and verify recursive result includes sensible `normalized`/`scanned` counts
* `help` from at least two different views and confirm help text is context-sensitive
* `quit` exits the session cleanly

## MVP ready criteria (command-lane closure)

Mark command-lane MVP ready only when all are true:

* Setup lock is enforced before bootstrap, with only bootstrap/help/quit routes allowed.
* Every core panel (status/diagnostics/services/users/logs/network) is reachable and operable from commands, not hotkeys alone.
* Batch command targeting works where implemented (services/users/log service+source/network selects), with explicit partial/success/failure feedback.
* FTP finalize lifecycle is event-driven and scoped (`system ftp-complete <path|dir>` / `/uploads/ftp/complete`), with recursive directory handling and reported normalized/scanned counts.
* `system dos2unix` remains scoped to Crate-managed paths (`config|services|all`) and avoids random host scans.
* Command help/usage text matches implemented grammar exactly.
* Diagnostics includes a verification surface that mirrors the installed-host MVP verifier prerequisites.
* Diagnostics includes direct managed-actor targeting and lifecycle inspection without requiring operators to open raw JSON artifacts.
