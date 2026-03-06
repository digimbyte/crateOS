# CrateOS

CrateOS is a curated, appliance-style server experience built on **Ubuntu Server LTS**. It presents a **single control surface** (a retro “Pitboy/DOS choose‑your‑adventure” TUI and optional web panel) and a **single canonical filesystem root** for configuration, services, logs, and state.

**Goal:** provide a robust “it just works” server platform with predictable defaults, self-healing policies, and a clean UX—while still allowing advanced users to “pop the hood” deliberately.

---

## Design principles

* **One root directory** users care about (Minecraft modpack vibe).
* **One supported control plane** (TUI + API). No memorizing thousands of CLI flags.
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

* **LightDM** (optional local desktop/login layer)
* **Cockpit** (optional lightweight web admin UI)

---

## UX changes vs stock Ubuntu Server

### SSH and interactive access

* Default SSH entry lands in the **Crate Console** (TUI) rather than a raw shell.
* Raw shell access is treated as **break-glass** (explicit and permission-gated).
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

---
## Installation model

CrateOS is **installed as part of the OS install** (autoinstall ISO / qcow2 pipeline). It is not an add-on users layer onto an existing Ubuntu host.

* No in-place “upgrade from vanilla Ubuntu” path
* Images are built and installed as **CrateOS-first** systems
* Upgrades are handled by new images/releases, not by in-place conversion or rollback

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
* `users.create`
* `users.edit`
* `users.delete`
* `roles.view`
* `roles.create`
* `roles.edit`
* `roles.delete`
* `audit.view`
* `logs.view`
* `shell.breakglass`
* `network.view`
* `network.edit`
* `proxy.view`
* `proxy.edit`
* `updates.view`
* `updates.apply`
* `backups.view`
* `backups.run`
* `modules.view`
* `modules.install`
* `modules.uninstall`
* `modules.enable`
* `modules.disable`

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

* **Owner**: `sys.manage`, `users.*`, `roles.*`, `audit.view`, `shell.breakglass`, `network.*`, `proxy.*`, `updates.*`, `backups.*`, `modules.*`, all `svc.*`
* **Admin**: `users.view`, `roles.view`, `network.edit`, `proxy.edit`, `updates.apply`, `backups.run`, `modules.*`, all `svc.*` (no break-glass)
* **Service Admin (scoped)**: all `svc.<service>.*`
* **Operator**: `svc.<service>.view|start|stop|restart|logs.view`
* **Auditor**: `audit.view`, `logs.view`, `svc.<service>.logs.view`

### Policy rules

* Deny by default
* Break-glass requires explicit permission and logs a security event
* Service-scoped roles never grant global permissions

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

* **crateos**: interactive TUI + CLI (user-facing)
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

  * `net.*`
  * `svc.*`
  * `svc.<service>.view|edit|start|stop`
  * `svc.<service>.plugins.add|remove|configure`
  * `proxy.*`
  * `logs.view`
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

## Roadmap notes

### MVP (first shippable)

* Directory layout + manifests
* TUI: status, logs, services, network
* Agent: apply idempotently (NM + systemd + nginx)
* Docker Compose support for a few modules

### Next

* Web UI panel
* ACME/TLS automation
* Role/permission UI
* Snapshot/rollback improvements
* Signed update channel for CrateOS packages
