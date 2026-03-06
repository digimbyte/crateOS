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

* **LightDM** (optional local desktop/login layer)
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

## Contributing

CrateOS is opinionated by design. Contributions that **reduce ambiguity** and **improve determinism** are preferred.

* Open an issue describing:

  * the user-facing problem
  * the desired “one true path” behavior
  * how it fits the `/srv/crateos` model

---

## License

TBD
