# CrateOS Build System

## Overview

The CrateOS build system is **fully cross-platform** and works on both Windows and Linux:

- **Windows**: Native Go builds via PowerShell + WSL2 for ISO/deb packaging
- **Linux**: Native builds via Makefile and bash scripts

## Windows Build (build.ps1)

### Requirements
- **Go 1.20+** (for native binary compilation)
- **WSL2** with Ubuntu installed (for packaging/ISO generation)
- **PowerShell 5.0+** (included on Windows 10/11)

### Targets

```powershell
.\build.ps1 build      # Compile Go binaries (dist/bin/*.exe)
.\build.ps1 deb        # Build .deb packages via WSL2 → dist/*.deb
.\build.ps1 iso        # Build autoinstall ISO via WSL2 → dist/crateos-*.iso
.\build.ps1 qcow2      # Build VM image via WSL2 → dist/crateos-*.qcow2
.\build.ps1 clean      # Remove dist/ directory
```

### Build Flow

1. **Windows (PowerShell)**:
   - `.\build.ps1 build` compiles Go binaries natively
   - Outputs: `dist/bin/crateos.exe`, `dist/bin/crateos-agent.exe`, `dist/bin/crateos-policy.exe`

2. **WSL2 Delegation** (for packaging):
   - `.\build.ps1 deb` → `wsl.exe bash -c "make deb"`
   - `.\build.ps1 iso` → `wsl.exe bash -c "make iso"`
   - Path conversion: `P:\CrateOS` → `/mnt/p/CrateOS` (automatic)

3. **In WSL2 (Linux)**:
   - Makefile runs standard Linux tools: `dpkg-deb`, `7z`, `xorriso`, `wget`, etc.
   - Creates `.deb` packages with systemd services, scripts, configs
   - Embeds `.deb` files into Ubuntu 24.04 ISO
   - Injects autoinstall configuration with late-commands

### Example: Build ISO on Windows

```powershell
cd P:\CrateOS
.\build.ps1 build      # Compile binaries (30 sec)
.\build.ps1 deb        # Package debs in WSL (1-2 min)
.\build.ps1 iso        # Create ISO in WSL (2-5 min)
# Output: dist/crateos-0.1.0-dev.iso
```

## Linux Build (make)

### Requirements
- **Go 1.20+**
- **GNU Make**
- **Standard tools**: dpkg, 7z, xorriso, wget, sed, awk, grep

### Targets

```bash
make build    # Compile binaries → dist/bin/crateos, etc.
make deb      # Build .deb packages → dist/*.deb
make iso      # Build autoinstall ISO → dist/crateos-*.iso
make qcow2    # Build VM image → dist/crateos-*.qcow2
make clean    # Remove dist/
```

### Example: Build ISO on Linux

```bash
cd /path/to/CrateOS
make deb      # Package debs (1-2 min)
make iso      # Create ISO (2-5 min)
# Output: dist/crateos-0.1.0+noble1.iso
```

## ISO Build Details

### Base Image
- **Ubuntu 24.04 LTS** (Noble Numbat) - live server edition
- Resolved from `https://releases.ubuntu.com/noble/` to the latest `ubuntu-24.04.x-live-server-amd64.iso`
- Cached in `dist/cache/` for subsequent builds

### Patch Application
The ISO build injects CrateOS into the Ubuntu installer via:

1. **Extract** base ISO (7-Zip)
2. **Render** autoinstall config from `images/iso/autoinstall/user-data.template`
   - Substitutes hostname, users, passwords, required packages
3. **Embed** CrateOS `.deb` files to `/crateos-debs/` on ISO
4. **Inject** cloud-init config to `/nocloud/` directory
5. **Patch** kernel cmdline to enable `autoinstall ds=nocloud;s=/cdrom/nocloud/`
6. **Refresh** MD5 checksums for modified media
7. **Rebuild** ISO with xorriso preserving boot metadata

### Autoinstall Flow (in installer)

```yaml
late-commands:
  - Install CrateOS .deb files from media
  - Stamp crateos-login-shell + tty1 override into /target
  - Run verify-bootstrap-artifacts (from crateos-agent package)
```

### Result
- **Bootable ISO** with forced CrateOS installation
- All dependencies pre-installed
- Agent auto-starts on first boot
- `tty1` autologins the seeded operator into `crateos console`
- Ready to boot in VM/physical machine

## Dependencies

### Debian Packages (auto-installed during OS installation)

From `packaging/config/packages.yaml`:

**Core**:
- openssh-server, network-manager, nftables, openssl, ca-certificates

**Utilities**:
- curl, wget, jq, yq, git, rsync, dos2unix, lsof

**Maintenance**:
- chrony, unattended-upgrades, needrestart, logrotate

**Diagnostics**:
- btop, ncdu, smartmontools, nvme-cli, lm-sensors, fancontrol, hddtemp
- iperf3, ethtool, iw, dnsutils, mtr-tiny, pciutils, usbutils

**Security**:
- fail2ban, wireguard

**Dev Tools**:
- tmux, build-essential, pkg-config, python3, python3-venv, python3-pip, powershell

### CrateOS Packages

1. **crateos**: CLI/TUI console, SSH ForceCommand
2. **crateos-agent**: Platform state enforcer daemon + watchdog
3. **crateos-policy**: Periodic drift detection/repair

### Optional Packages (installed via user config)

- lightdm, cockpit (UI)
- nginx (web)
- docker.io, docker-compose-plugin (containers)
- postgres (database)
- msmtp (email)
- cloudflared (tunnels)
- screen (terminal extras)

## Troubleshooting

### Windows: "WSL is not installed"
- Install WSL2: `wsl --install` or from Microsoft Store
- Ensure Ubuntu distribution is installed

### Windows: Path conversion fails
- PowerShell escaping issue — verify WSL is running
- Try manual path: `wsl.exe bash -c "cd /path && make iso"`

### WSL: Missing tools
- Run in WSL: `sudo apt-get update && sudo apt-get install -y xorriso p7zip-full wget`

### ISO: "no .deb files found"
- Run `.\build.ps1 deb` first to create packages
- Check `dist/` contains `.deb` files

### ISO: Autoinstall fails
- Ubuntu installer requires: user-data, meta-data, kernel cmdline patch
- Verify `images/iso/autoinstall/` files exist
- Check late-commands in rendered user-data match `/cdrom/crateos-debs/` layout

## Version Handling

Version injected at build time:
- Default: `0.1.0-dev` (Windows) or `0.1.0+noble1` (Linux)
- Override: `VERSION=1.2.3 .\build.ps1 deb`

Version embedded in:
- Deb package control files
- deb postinst (CRATEOS_VERSION placeholder)
- ISO label: `CrateOS 1.2.3`

## Cross-Platform Notes

**Line Endings**: All bash scripts converted to LF (not CRLF) for WSL compatibility.

**Path Separators**: 
- Windows build script uses PowerShell Path.Combine() / -Path operators
- WSL path conversion handled by wslpath
- Makefile uses Unix-style forward slashes (works in WSL)

**Binaries**:
- Windows: compiled as `*.exe` 
- Linux: compiled without extension
- Both use same Go source code

## Verification

After build, verify:

```bash
# On target system (Ubuntu 24.04 boot from ISO)
curl http://localhost/status                    # Agent API responding
systemctl status crateos-agent.service          # Agent running
systemctl status crateos-agent-watchdog.timer   # Watchdog timer active
ls -la /srv/crateos/state/platform-state.json  # Platform state file
```

Or run the verification script:
```bash
bash /srv/crateos/scripts/verify-bootstrap-artifacts
bash /srv/crateos/scripts/verify-mvp-install.sh
```
