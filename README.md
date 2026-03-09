# CrateOS

<img src="assets/CrateOS.png" alt="CrateOS" width="360" />

A curated, appliance-style server platform built on **Ubuntu Server LTS**.

CrateOS replaces shell-first administration with a **Pitboy/DOS choose-your-adventure console** (SSH/TUI) and an optional web panel. Everything users care about lives under a single canonical root: **`/srv/crateos`**.

> **Thesis:** Linux is powerful but scattered. CrateOS turns it into a cohesive “vehicle as a service”: one control plane, one directory model, and predictable defaults.

---

## What CrateOS is

* **A platform layer** on top of Ubuntu Server LTS (not a distro fork).
* **A control plane** that owns the supported workflows:

  * networking, firewall, services, reverse proxy, logs, updates, backups
* **A modular service system** (“crates”) with clean lifecycle management.
* **A modpack-style filesystem layout** so configs/logs/state don’t get scattered.
* **A multi-tenant server framework** with a cPanel-like UX: multiple users log in and manage services/modules through roles and permissions.

## What CrateOS is not

* A security product that tries to stop determined root users.
* A replacement kernel or a new Linux distribution.
* An app store clone without opinionated operational policy.

---

## Key features

* **TUI-first administration**: SSH lands in a guided console instead of a raw shell.
* **Controlled session surfaces**: local GUI and future virtual desktop entry points are intended to host CrateOS-owned sessions, not a normal distro desktop.
* **Single canonical root**: configs, logs, state, modules, and service data live under `/srv/crateos`.
* **Idempotent apply**: declarative configuration → desired state → agent applies safely.
* **Self-healing networking**: headless-safe NetworkManager profiles with MAC-based matching.
* **Managed reverse proxy**: nginx templates and a single mapping model.
* **Modular services**: enable/disable/install/uninstall crates without guessing commands.
* **Curated logging**: exported, readable logs (plus optional deep links to OS internals).
* **Clean maintenance**: one-button cleanup (journald vacuum, docker prune, cache cleanup).

---

## Canonical filesystem layout

CrateOS makes the OS layout an implementation detail. Users interact with **one root**:

```text
/srv/crateos/
  config/                 # the only human-edited configs
    crateos.yaml
    network.yaml
    firewall.yaml
    services.yaml
    reverse-proxy.yaml
    users.yaml

  modules/                # module definitions (like “mod metadata”)
  services/               # each service lives like a crate/mod
  state/                  # desired/actual, snapshots, last-good
  logs/                   # exported/curated logs
  export/                 # curated view (symlink farm) to OS internals
  bin/                    # platform tools (crateos, agent)
```

---

## What ships with CrateOS

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

## How it works

### Components

* **`crateos`**: interactive TUI + CLI
* **`crateos-agent`**: local daemon (root) that applies desired state idempotently
* **`crateos-policy`**: boot/periodic drift detection and repair
* **(optional) web panel**: speaks to the agent via a local socket/API

### State model

1. `config/*.yaml` defines **Desired State**
2. the agent probes the system for **Actual State**
3. the agent computes a diff and applies changes safely
4. `state/last-good/` snapshots enable rollback when needed

---

## Roadmap

### MVP

* Root layout + manifests
* TUI: status, logs, services, network
* Agent: apply (NetworkManager + systemd + nginx)
* Docker Compose support for selected modules

### Next

* Web panel
* TLS/ACME automation
* Role/permissions UI
* Signed update channel for CrateOS packages
* Better snapshots/rollback

---

## MVP machine readiness (ISO install + test/update lanes)

Use this flow to produce image artifacts where the CrateOS ISO is a forced framework installer, and non-ISO artifacts are for test/update workflows.

### 1) Build artifacts
Build host prerequisites for image targets:

* ISO: `wget`, `7z`, `xorriso`, `sed`, `grep`
* qcow2: `wget`, `qemu-img`, `cloud-localds`, `guestfish`

```bash
make build
make deb
make iso
make qcow2
```
`make iso` requires `.deb` artifacts and embeds them into the install media; this ISO always installs CrateOS.

Expected outputs in `dist/`:

* `dist/bin/crateos`
* `dist/bin/crateos-agent`
* `dist/bin/crateos-policy`
* `dist/crateos_<version>_amd64.deb`
* `dist/crateos-agent_<version>_amd64.deb`
* `dist/crateos-policy_<version>_amd64.deb`
* `dist/crateos-<version>.iso`
* `dist/crateos-<version>.qcow2`
* `dist/seed-<version>.iso`
`.deb` outputs are image-pipeline build artifacts used for provisioning, not a standalone operator install surface.
If operator state is ever lost after install, recover it with `crateos bootstrap <name>` from the local machine.

### 2) Install path vs test/update paths

Primary framework install path:

* `dist/crateos-<version>.iso` (forced CrateOS install path)

Testing/update lanes:

* `dist/crateos-<version>.qcow2` + `dist/seed-<version>.iso` (paired test/update workflow)

### 3) Verify service/timer activation on first boot

```bash
systemctl status crateos-agent.service --no-pager
systemctl status crateos-policy.timer --no-pager
```
Default first-login credential for ISO seed user:

* user: `crate`
* password: `crateos` (expired in the target system during install; change required on first login)

### 4) Verify CrateOS root bootstrap

```bash
ls -la /srv/crateos
cat /srv/crateos/state/installed.json
```

Also verify default config seed files exist on first install:

```bash
ls -la /srv/crateos/config
```

### 5) Verify SSH force-command landing

Confirm `/etc/ssh/sshd_config.d/10-crateos.conf` exists and includes:

* `ForceCommand /usr/local/bin/crateos console`

### 6) Framework install expectations

* CrateOS framework install is ISO-based and forced by this installer; non-ISO artifacts are test/update lanes.
* Existing `/srv/crateos/config/*.yaml` is preserved; default configs are only seeded when missing on install.
* ISO late-commands install embedded CrateOS `.deb` files and run dependency repair (`apt-get -f install`) if needed.
* ISO autoinstall fails fast if embedded `.deb` payload is missing, if expected CrateOS binaries are not present in target rootfs after package install, or if seeded config files / persistent unit enablement links are missing.
* ISO rebuild now replays the source Ubuntu ISO boot metadata and refreshes `md5sum.txt` after media mutation instead of assuming older hard-coded isolinux paths.
* ISO and qcow2 now share the same installed bootstrap-artifact verification path before runtime validation, reducing lane drift in machine-readiness checks.
* ISO and qcow2 both derive their required base package list from `packaging/config/packages.yaml` instead of maintaining separate copied package blocks.
* ISO and qcow2 both derive shared seed identity defaults (hostname, default user, password hash) from `images/common/seed-defaults.env`.
* `crateos-policy.timer` now refreshes the canonical readiness report on a 2-minute cadence after boot; installed-host verification treats that report as stale after 3 minutes.
* Run one-command verification on installed host:

```bash
/usr/local/bin/verify-mvp-install
```

### 7) Installable MVP acceptance contract
Treat the installable MVP foundation as complete only when all of the following are true:

* `make build` produces `dist/bin/crateos`, `dist/bin/crateos-agent`, and `dist/bin/crateos-policy`.
* `make deb` produces the three expected CrateOS `.deb` artifacts in `dist/`.
* The staged Debian metadata version matches the build version used for the binaries and emitted artifact filenames.
* `make iso` produces `dist/crateos-<version>.iso`.
* `make qcow2` produces `dist/crateos-<version>.qcow2` and `dist/seed-<version>.iso`.
* Installing from the CrateOS ISO completes without bypassing the embedded CrateOS payload.
* First boot has `crateos-agent.service` active/enabled, `crateos-agent-watchdog.timer` active/enabled, and `crateos-policy.timer` active/enabled.
* `/srv/crateos`, `/srv/crateos/config`, `/srv/crateos/state/installed.json`, `/srv/crateos/state/platform-state.json`, `/srv/crateos/state/agent-watchdog.json`, `/srv/crateos/state/readiness-report.json`, `/srv/crateos/state/storage-state.json`, and the seeded default config files exist.
* SSH lands in `crateos console` via `ForceCommand /usr/local/bin/crateos console`.
* `/usr/local/bin/verify-mvp-install` passes on the installed host.
* `/srv/crateos/runtime/agent.sock` exists as a live Unix socket after boot.
* Platform, storage, and watchdog state artifacts are fresh enough to show the control plane is still updating, not just historically present.
* `readiness-report.json` is refreshed on the expected policy cadence and remains fresh under the installed-host verifier window instead of aging into a false degraded state.
* The canonical readiness report says the machine is `ready`, not merely partially present.
* The first interactive console session renders even if the agent socket is not yet ready, using local fallback state where needed.
* Installed operator state includes at least one configured admin so local CLI/TUI control paths are usable after first boot.
* Agent liveness recovery attempts are logged on-host so post-boot crashes do not become silent control-plane loss.

---

## Contributing

CrateOS is opinionated by design. Contributions that **reduce ambiguity** and **improve determinism** are preferred.

* Open an issue describing:

  * the user-facing problem
  * the desired “one true path” behavior
  * how it fits the `/srv/crateos` model

---

## License
Apache-2.0. See `LICENSE`.
TBD
