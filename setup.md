Here’s a **step-by-step setup plan** assuming:

* We’re on **Windows**
* We just `git clone`’d the repo
* We have two docs in place: `README.md` and `GUIDE.md`
* We want a pipeline that can output **qcow2**, **autoinstall ISO**, and later **Raspberry Pi image**

I’m going to give you a straight action list you can follow in order.
**Important model note:** CrateOS is installed **as part of the OS install** (ISO/qcow2 image). There is **no in-place upgrade path from vanilla Ubuntu**.

---

## Phase 0 — Decide the “MVP contract” (30 minutes)

**Outcome:** one bootable target that proves the product loop.

1. Define MVP acceptance tests (write these into `GUIDE.md`):

   * After install/boot, `/srv/crateos` exists with the expected subfolders.
   * `crateos-agent` is running as a systemd service.
   * SSH login drops into CrateOS console (not a raw shell).
   * `crateos` console can show: OS version, uptime, CPU/RAM, disks, network status.
   * Network is up pre-login (LAN at minimum; Wi-Fi optional for MVP).

2. Pick the first target to make green:

   * **qcow2 VM image** first (fast iteration), then ISO.
   * Pi comes later once amd64 pipeline is stable.

---

## Phase 1 — Get a build environment on Windows (WSL2) (30–60 min)

**Outcome:** you can run Linux build tooling without having “Ubuntu OS locally.”

1. Install WSL2 Ubuntu:

   * Windows Features: enable **Windows Subsystem for Linux** + **Virtual Machine Platform**
   * Install “Ubuntu” from Microsoft Store
2. In WSL:

   ```bash
   sudo apt update
   sudo apt install -y git make build-essential curl xz-utils
   ```
3. Install Go (recommended) inside WSL (use Ubuntu’s or upstream):

   ```bash
   sudo apt install -y golang
   go version
   ```

---

## Phase 2 — Create the repo skeleton that supports *all* targets (same day)

**Outcome:** the project can build core binaries and has a predictable layout.

1. Add folders (commit these as empty with `.gitkeep` where needed):

   ```
   cmd/crateos/
   cmd/crateos-agent/
   cmd/crateos-policy/
   internal/
   modules/
   packaging/deb/
   images/iso/
   images/qcow2/
   images/rpi/
   scripts/
   .github/workflows/
   ```

2. Initialize Go module at repo root:

   ```bash
   go mod init github.com/<you>/<crateos>
   ```

3. Create **minimal** binaries (so the pipeline has something to build):

   * `crateos`: prints “CrateOS console MVP” + basic status
   * `crateos-agent`: runs forever, logs “agent alive”
   * `crateos-policy`: oneshot prints “policy check ok”

4. Add a top-level `Makefile` with **non-negotiable** targets:

   * `make build` (build binaries)
   * `make deb` (build .deb packages)
   * `make iso` (build autoinstall ISO)
   * `make qcow2` (build VM image)
   * `make rpi` (stub target for now)

---

## Phase 3 — Hardwire the CrateOS root + systemd services (MVP functionality)

**Outcome:** installing CrateOS creates the filesystem and starts services.

1. Define canonical root constant: `/srv/crateos`

2. In `crateos-agent` first boot / startup:

   * Ensure directories exist:

     * `/srv/crateos/{config,modules,services,state,logs,export,bin}`
   * Write a marker file:

     * `/srv/crateos/state/installed.json` (version, install time)

3. Create systemd units (these ship in your deb):

   * `crateos-agent.service` (enabled)
   * `crateos-policy.service` + `crateos-policy.timer` (enabled)

4. SSH “forced console” config (ship in your deb):

   * `/etc/ssh/sshd_config.d/10-crateos.conf`:

     * `ForceCommand /usr/local/bin/crateos console` (or wherever you install it)
   * restart ssh in install scripts

---

## Phase 4 — Debian packaging (so images install cleanly)

**Outcome:** `make deb` produces installable packages.

1. Create packages:

   * `crateos` (installs `/usr/local/bin/crateos`)
   * `crateos-agent` (installs agent + systemd unit)
   * `crateos-policy` (installs policy + timer)
   * optional meta-package later: `crateos-base` depends on your tooling list

2. `postinst` scripts should:

   * create `/srv/crateos` structure (idempotent)
   * enable/start `crateos-agent`
   * enable timer `crateos-policy.timer`
   * drop sshd forced-command file and restart ssh

3. Verify locally in WSL (basic):

   * build deb
   * install into a throwaway container or VM (later)

---

## Phase 5 — ISO pipeline (Ubuntu autoinstall)

**Outcome:** `make iso` outputs a bootable installer that installs Ubuntu Server and then CrateOS.

1. Under `images/iso/` add:

   * `autoinstall/user-data`
   * `autoinstall/meta-data`
   * `build.sh`

2. `user-data` should:

   * install base packages (openssh-server, network-manager, etc.)
   * in `late-commands`:

     * copy your `.deb` files (bundled) and `dpkg -i` them
     * or curl them from GitHub Releases (online mode)

3. `build.sh`:

   * downloads the official Ubuntu Server ISO (version pinned)
   * injects autoinstall seed
   * outputs `dist/crateos-<ver>.iso`

**Rule:** ISO build supports two modes:

* `ISO_MODE=online` (pull deps from Ubuntu repos)
* `ISO_MODE=offline-lite` (embed CrateOS debs; still pulls Ubuntu deps)

---

## Phase 6 — qcow2 pipeline (fast iteration)

**Outcome:** `make qcow2` outputs a VM image you can boot in VirtualBox/VMware/QEMU.

1. Use a known-good method:

   * Start from Ubuntu cloud image
   * Use cloud-init to install CrateOS debs on first boot
2. Output: `dist/crateos-<ver>.qcow2`

This becomes your “tight loop” for development.

---

## Phase 7 — GitHub Actions matrix build (makes the project mobile)

**Outcome:** push a tag → CI builds all artifacts → attaches to Release.
**Source of truth:** GitHub Releases.

1. Workflow jobs:

* `build-deb` (amd64)
* `build-iso` (amd64)
* `build-qcow2` (amd64)
* `build-rpi` (arm64) — stub initially

2. Upload artifacts:

* `.deb` packages
* `.iso`
* `.qcow2`
* checksum files

3. Release policy (draft):

* Tag version → build → attach artifacts
* Publish checksums (and optional signatures)
* This is the distribution mechanism; no external update service required

---

## Phase 8 — Raspberry Pi track (after amd64 is stable)

**Outcome:** `make rpi` builds an image that boots and installs CrateOS.

* Use a Pi-friendly base (Ubuntu Server for Pi) and first-boot provisioning to install CrateOS debs.
* Keep it a separate lane; don’t let Pi complexity block amd64.

---

## What you do *today* (exact next actions)

1. Setup WSL2 Ubuntu + Go
2. Create repo folder skeleton + `go mod init`
3. Implement minimal `crateos`, `crateos-agent`, `crateos-policy` binaries
4. Add `Makefile` with `build` and `deb` working
5. Ship systemd units + `/srv/crateos` creation in `postinst`
6. Only then: ISO builder

---

If you want, I can now turn this into a **checklist-style `GUIDE.md` section** (with headings + commands), but I didn’t touch your canvases yet because you asked for the plan first.
