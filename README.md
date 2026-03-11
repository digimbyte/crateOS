# CrateOS

<img src="assets/CrateOS.png" alt="CrateOS" width="360" />

CrateOS is a curated, modular **Ubuntu Server LTS control plane** built to feel more like an **Ubuntu cPanel** than a traditional Linux box.

Its purpose is simple: **remove the nerd from Linux** for day-to-day server management without stripping power away from trusted admins. CrateOS replaces scattered shell-first workflows with a guided control surface, a canonical filesystem model, and predictable operational paths for the things people actually want to manage.

That means no forcing operators to remember random CLI incantations, no routine bash-prompt babysitting, less permission/path trivia, and less damage from common operator mistakes like bad uploads, line-ending problems, or config drift spread across the host.

At the same time, CrateOS is **not** trying to imprison the machine from authorized users. If an admin has `sudo`, that access is intentional. CrateOS provides the sane default path, the clean management path, and the modular path—not fake protection from the people who are already trusted to own the box.

> **Thesis:** Ubuntu is a strong base, but stock Linux administration is too fragmented, too manual, and too easy to derail with trivia. CrateOS turns it into a cohesive control plane with one root, one management model, and one supported operator experience.

---

## What CrateOS is

* **A platform layer** on top of Ubuntu Server LTS, not a distro fork.
* **A cPanel-style management experience for Ubuntu** focused on self-hosted services, system operations, and modular software management.
* **A control plane** for common operational workflows:

  * networking, firewall, reverse proxy, services, logs, updates, backups, users, cron, and custom software
* **A modular service and workload system** (“crates”) with clean install/enable/start/stop/disable/uninstall flows.
* **A canonical filesystem and state model** so configs, logs, exports, runtime state, and service data are not scattered across the host.
* **A multi-user operational framework** where people manage curated and custom modules through roles, permissions, and guided surfaces instead of raw host trivia.

## What CrateOS is not

* A product that tries to block or “outsmart” admins who already have `sudo`.
* A replacement kernel or a new Linux distribution.
* A generic app store with no opinionated operational model.
* A promise that the shell never exists; it is a promise that the shell should not be the normal path for routine management.

---

## Core intent

CrateOS exists to make Ubuntu server administration easier to deploy, easier to understand, and easier to recover without falling back into Linux tribal knowledge.

The project is aimed at operators who want to:

* drop in a machine and get to a usable managed state quickly
* install and manage curated modules without hand-building every workflow
* run custom software without reinventing service layout, policy, and maintenance every time
* manage cron and scheduled workloads from a structured platform surface
* avoid permission/path/newline/config mistakes caused by ad hoc CLI and FTP habits
* keep the host customizable and extensible without turning it back into unmanaged Linux chaos

---

## Key features

* **CrateOS-first administration**: the machine lands in the CrateOS control surface instead of treating a raw shell as the normal operator interface.
* **Optional web-panel posture**: the platform supports an appliance-style management surface beyond the terminal.
* **Single canonical root**: configs, logs, state, modules, and service data live under `/srv/crateos`.
* **Idempotent apply model**: declarative configuration → desired state → agent applies safely.
* **Guided system management**: networking, reverse proxy, services, maintenance, and platform posture are exposed as supported workflows instead of “figure out the right command.”
* **Modular services and software**: install, enable, disable, remove, and manage curated or custom modules with predictable lifecycle behavior.
* **Cron and managed workload direction**: scheduled jobs and custom software are part of the intended platform scope, not an afterthought.
* **Curated logging and exports**: readable logs and normalized system views instead of hunting through native paths by memory.
* **Reduced operator footguns**: the platform is designed to minimize file-permission churn, path sprawl, config drift, and newline/transfer mistakes that commonly come from manual host handling.
* **Admin escape hatches by intent**: authorized users can still access the underlying OS when needed; CrateOS just makes that the exception rather than the daily workflow.

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

* **`crateos`**: the primary operator interface and control surface
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
If operator state is ever lost after install, the local console should route into the CrateOS primer; `crateos bootstrap <name>` remains the manual local recovery path when users are missing.

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
Default first-install seeded operator identity:

* user: the installer-seeded first operator from `images/common/seed-defaults.env` (default: `crate`)
* password: `crateos`
* local `tty1` autologins that seeded operator and lands in `crateos console` through `/usr/local/bin/crateos-login-shell`

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
* ISO autoinstall fails fast if embedded `.deb` payload is missing, if expected CrateOS binaries are not present in target rootfs after package install, or if the seeded operator takeover files (`crateos-login-shell`, `tty1` override, operator shell assignment) are missing.
* ISO rebuild now replays the source Ubuntu ISO boot metadata and refreshes `md5sum.txt` after media mutation instead of assuming older hard-coded isolinux paths.
* ISO and qcow2 now share the same installed bootstrap-artifact verification path before runtime validation, reducing lane drift in machine-readiness checks.
* ISO and qcow2 both derive their required base package list from `packaging/config/packages.yaml` instead of maintaining separate copied package blocks.
* ISO and qcow2 both derive shared seed identity defaults (hostname, default user, password hash) from `images/common/seed-defaults.env`.
* The installer identity user is promoted into `/srv/crateos/config/users.yaml` as the initial CrateOS admin rather than relying on a separate hardcoded framework account.
* Local console takeover is an image contract: `tty1` autologins the seeded operator, that operator uses `/usr/local/bin/crateos-login-shell`, and the login shell `exec`s `crateos console` instead of landing in raw bash.
* When local first-use/runtime state is incomplete, `crateos console` stays inside a locked CrateOS primer manager instead of failing out to a shell or presenting the normal menu as if the machine were ready; that primer now persists machine identity in `crateos.yaml`, repairs local takeover artifacts, and provisions the first admin locally.
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
* `tty1` autologins the seeded operator through `getty@tty1` override and that operator shell is `/usr/local/bin/crateos-login-shell`.
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
