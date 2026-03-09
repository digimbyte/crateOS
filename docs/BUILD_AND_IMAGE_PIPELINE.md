# BUILD_AND_IMAGE_PIPELINE

## Binaries
- `make build` → `dist/bin/{crateos,crateos-agent,crateos-policy}`
- Version injected via `-ldflags -X ...platform.Version=$(VERSION)` (default `0.1.0+noble1`)

## Debian packages
- `make deb` (Linux/WSL/CI)
- Stages `packaging/deb/*` into `dist/deb-staging/<pkg>`
- Injects VERSION into `DEBIAN/control` and postinst (`CRATEOS_VERSION`)
- Outputs: `dist/<pkg>_<VERSION>_amd64.deb`

Contents:
- Binaries to `/usr/local/bin`
- Systemd units:
  - `crateos-agent.service`
  - `crateos-policy.service` + `.timer`
- SSH forced-command: `/etc/ssh/sshd_config.d/10-crateos.conf`
- Postinst creates `/srv/crateos` tree and writes `state/installed.json`

## qcow2 image
- `make qcow2` (requires `qemu-utils`, `cloud-localds`, `guestfish`)
- Starts from Ubuntu 24.04 cloud image
- Resizes to 20G
- Renders shared seed identity defaults from `images/common/seed-defaults.env`
- Renders required package list from `packaging/config/packages.yaml`
- Builds `seed-<VERSION>.iso` with cloud-init:
  - default user `crate` (password expired during provisioning)
  - installs required base packages plus CrateOS debs from `/var/tmp/crateos-debs`
  - runs shared bootstrap-artifact verification before runtime validation
  - runs `/usr/local/bin/verify-mvp-install`
- Embeds debs into qcow2 via guestfish inspection mode (`-i`) instead of assuming a fixed root partition path
- Outputs:
  - `dist/crateos-<VERSION>.qcow2`
  - `dist/seed-<VERSION>.iso` (attach alongside the qcow2 for this cloud-init lane)

## ISO (autoinstall)
- `make iso` (requires `xorriso`, `p7zip-full`, `wget`)
- Downloads Ubuntu 24.04.2 live-server ISO (cached)
- Renders shared seed identity defaults from `images/common/seed-defaults.env`
- Renders required package list from `packaging/config/packages.yaml`
- Injects autoinstall seed under `nocloud/` (`user-data`, `meta-data`)
- Ensures kernel cmdline has `autoinstall ds=nocloud;s=/cdrom/nocloud/`
- Embeds CrateOS debs under `crateos-debs/` (required for this installer)
- Regenerates `md5sum.txt` after media mutation
- Runs shared bootstrap-artifact verification inside the target rootfs after package install
- Rebuilds ISO via xorriso using boot metadata replayed from the source Ubuntu ISO rather than hard-coded isolinux paths
- Output: `dist/crateos-<VERSION>.iso`

## Release flow (intended)
- Build deb → qcow2 → ISO
- Upload artifacts + checksums to GitHub Releases
- Baseline: Ubuntu 24.04 (noble) `amd64`

## Installable MVP acceptance contract
- `make build` outputs `dist/bin/{crateos,crateos-agent,crateos-policy}`
- `make deb` outputs all three CrateOS `.deb` artifacts in `dist/`
- `make iso` outputs `dist/crateos-<VERSION>.iso`
- `make qcow2` outputs `dist/crateos-<VERSION>.qcow2` and `dist/seed-<VERSION>.iso`
- ISO install must consume the embedded `crateos-debs/` payload and fail if it is missing
- First boot must have:
  - `crateos-agent.service` active/enabled
  - `crateos-policy.timer` active/enabled
  - `/srv/crateos/state/installed.json` present
  - seeded configs under `/srv/crateos/config/`
  - SSH forced into `/usr/local/bin/crateos console`
- Readiness freshness contract:
  - `crateos-policy.timer` refreshes `readiness-report.json` every 2 minutes after boot
  - installed-host verification treats `readiness-report.json` as stale after 3 minutes
- Installed-host proof point: `/usr/local/bin/verify-mvp-install` passes
- First console session must render with local fallback state even before the local agent API is ready
