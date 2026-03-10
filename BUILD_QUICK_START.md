# CrateOS ISO Build — Quick Start

## Prerequisites

**On Windows**:
- Go 1.20+ installed
- WSL2 installed with Ubuntu distribution
- PowerShell 5.0+ (included on Windows 10/11)

**On Linux**:
- Go 1.20+ installed
- `make`, `dpkg`, `7z`, `xorriso`, `wget`

## Build the ISO (Windows)

```powershell
cd P:\CrateOS

# 1. Build binaries
.\build.ps1 build

# 2. Create .deb packages (delegates to WSL)
.\build.ps1 deb

# 3. Create ISO (delegates to WSL)
.\build.ps1 iso

# Done! ISO is at: dist/crateos-0.1.0-dev.iso
```

**Time**: ~5-10 minutes total

**Output**: `dist/crateos-0.1.0-dev.iso`

## Build the ISO (Linux)

```bash
cd /path/to/crateos

# 1. Build binaries and packages
make deb

# 2. Create ISO
make iso

# Done! ISO is at: dist/crateos-0.1.0+noble1.iso
```

**Time**: ~5-10 minutes total

**Output**: `dist/crateos-0.1.0+noble1.iso`

## What Happens

1. **Go binaries** compiled with embedded version
2. **.deb packages** created:
   - `crateos` — CLI/TUI console
   - `crateos-agent` — Platform state daemon + watchdog
   - `crateos-policy` — Drift detection/repair
3. **Ubuntu 24.04 ISO** downloaded (cached)
4. **CrateOS packages embedded** on ISO media
5. **Autoinstall config injected**:
   - Cloud-init user-data (hostname, users, packages)
   - Kernel cmdline patched for automatic install
   - Bootstrap verification hook in late-commands
6. **ISO rebuilt** with xorriso preserving boot metadata

## Boot and Verify

1. Boot VM or physical machine from ISO
2. Automatic Ubuntu installation starts (~3-5 min)
3. CrateOS packages install during `late-commands`
4. System reboots
5. Agent auto-starts, reconciles state
6. Ready to use

## Troubleshooting

**"WSL is not installed"**
```powershell
# Install WSL2
wsl --install

# Then verify
wsl.exe --version
```

**"Missing tools in WSL"**
```bash
# In WSL Ubuntu terminal
sudo apt-get update
sudo apt-get install -y xorriso p7zip-full wget
```

**"No .deb files found"**
- Ensure `.\build.ps1 deb` completed successfully
- Check `dist/` contains `*.deb` files

**Manual WSL build**
```powershell
wsl.exe bash -c "cd /mnt/p/crateos && make iso"
```

## Custom Build Options

**Set version**:
```powershell
# Windows
$env:VERSION = "1.2.3"; .\build.ps1 deb; .\build.ps1 iso

# Linux
VERSION=1.2.3 make iso
```

**Custom Ubuntu ISO**:
```bash
UBUNTU_ISO_URL="https://..." make iso
```

**Clean everything**:
```powershell
.\build.ps1 clean     # Windows
make clean            # Linux
```

## Output Artifacts

After successful build:

```
dist/
├── bin/
│   ├── crateos.exe           (Windows) or crateos (Linux)
│   ├── crateos-agent.exe     (Windows) or crateos-agent (Linux)
│   └── crateos-policy.exe    (Windows) or crateos-policy (Linux)
├── crateos-agent_0.1.0-dev_amd64.deb
├── crateos-policy_0.1.0-dev_amd64.deb
├── crateos_0.1.0-dev_amd64.deb
├── crateos-0.1.0-dev.iso            ← Boot this!
└── cache/
    └── ubuntu-24.04.2-live-server-amd64.iso
```

## What's on the ISO

**Ubuntu 24.04 LTS** base with:
- OpenSSH, NetworkManager, nftables
- Development tools (Go, Python, build-essential)
- System utilities (curl, wget, jq, tmux, btop, etc.)
- CrateOS agents and CLI
- Systemd timers for reconciliation
- SSH ForceCommand for TUI access
- Auto-login and password reset on first boot

**Storage**: ~2 GB ISO

**Boot**: UEFI/BIOS compatible (xorriso preserves metadata)

## Next Steps

1. **Distribute ISO** to target machines
2. **Boot from ISO** (VM or USB)
3. **Automatic installation** (no user interaction)
4. **Agent starts**, reconciles platform state
5. **SSH into machine** → lands in CrateOS TUI
