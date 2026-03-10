# CrateOS Windows Build - Setup Guide

Your system is **partially ready** for CrateOS builds. Use this guide to complete the setup.

## Current Status

```
✓ Go 1.24.4 installed
✓ PowerShell 7.5.4 present
✓ WSL2 installed
✗ WSL2 VM cannot start
✗ No Ubuntu distro found
✗ Build tools unavailable
```

## Step-by-Step Setup

### Step 1: Install Ubuntu in WSL2

Ubuntu must be installed and running for CrateOS packaging.

**Option A: Command Line (Recommended)**
```powershell
# Open PowerShell as Administrator and run:
wsl.exe --install -d Ubuntu-24.04

# This will:
# - Download Ubuntu 24.04 LTS
# - Install in WSL2
# - Prompt for username/password
# - Ready to use

# Verify installation
wsl.exe --list --verbose
# Should show: Ubuntu-24.04 ... Stopped ... 2
```

**Option B: Microsoft Store**
1. Open Microsoft Store app
2. Search: `Ubuntu 24.04 LTS`
3. Click "Install"
4. Wait 2-3 minutes
5. Launch from Start menu

**Option C: Manual Installation**
```powershell
# List available distros
wsl.exe --list --online

# Find Ubuntu-24.04 in the list, then install
wsl.exe --install -d Ubuntu-24.04
```

### Step 2: Start Ubuntu and Verify

```powershell
# Start Ubuntu VM
wsl.exe -d Ubuntu

# Inside Ubuntu terminal, update packages
sudo apt-get update
sudo apt-get upgrade -y
```

Then exit Ubuntu (type `exit`).

### Step 3: Install Build Tools in Ubuntu

The build system needs specific tools. Install them:

```powershell
# Install all required build tools
wsl.exe -d Ubuntu -- sudo apt-get install -y \
  wget \
  p7zip-full \
  xorriso \
  make \
  dpkg

# This takes 1-2 minutes
```

Or if you prefer to do it interactively:
```powershell
wsl.exe -d Ubuntu
# Now in Ubuntu terminal:
sudo apt-get update
sudo apt-get install -y wget p7zip-full xorriso make dpkg
exit
```

### Step 4: Verify Prerequisites

Run the build system's prerequisite checker:

```powershell
cd P:\CrateOS
.\build.ps1 check
```

**Expected output**:
```
CrateOS Build Prerequisite Check

Checking Windows prerequisites...
  ✓ go version go1.24.4 windows/amd64
  ✓ PowerShell 7.5.4
Checking WSL prerequisites...
  ✓ WSL2 installed
  ✓ WSL2 VM starts successfully
  ✓ Found Ubuntu distro: Ubuntu-24.04
  ✓ All required build tools present (wget, 7z, xorriso, dpkg, make)

✓ All prerequisites satisfied. Ready to build.
```

### Step 5: Build CrateOS

Once prerequisites are verified:

```powershell
cd P:\CrateOS

# Step 1: Compile Go binaries (30 seconds)
.\build.ps1 build

# Step 2: Create .deb packages (1-2 minutes)
.\build.ps1 deb

# Step 3: Create bootable ISO (2-5 minutes)
.\build.ps1 iso

# Done! ISO is ready at: dist/crateos-0.1.0-dev.iso
```

## Troubleshooting

### "WSL2 VM cannot start"

```powershell
# Restart the WSL2 VM
wsl.exe --shutdown

# Wait a few seconds, then try again
.\build.ps1 check
```

### "No Ubuntu distro found"

Ubuntu is not installed. Install it:

```powershell
# Install Ubuntu 24.04
wsl.exe --install -d Ubuntu-24.04

# Wait for installation to complete
# Then verify
wsl.exe --list --verbose
```

### "Missing build tools" error

One or more required tools are not installed in Ubuntu. Install them:

```powershell
# Install all tools at once
wsl.exe -d Ubuntu -- sudo apt-get install -y wget p7zip-full xorriso make dpkg

# Or specific tool
wsl.exe -d Ubuntu -- sudo apt-get install -y xorriso

# Then verify
.\build.ps1 check
```

### WSL Hangs During Check

```powershell
# Kill the stuck WSL process
wsl.exe --shutdown

# Wait 5 seconds, then try again
.\build.ps1 check
```

### "Permission denied" in Ubuntu

Make sure to use `sudo` for apt-get commands:

```powershell
# WRONG - will fail:
wsl.exe -d Ubuntu -- apt-get install xorriso

# RIGHT - will work:
wsl.exe -d Ubuntu -- sudo apt-get install -y xorriso
```

## Build Workflow

Once everything is set up:

```powershell
# Check everything is ready
.\build.ps1 check

# Build binaries (Windows native, fast)
.\build.ps1 build

# Create packages (delegated to WSL, medium)
.\build.ps1 deb

# Create ISO (delegated to WSL, slow)
.\build.ps1 iso

# Your bootable ISO is ready!
Get-Item dist/crateos-*.iso
```

## What Each Build Target Does

| Target | What It Does | Where | Time |
|--------|-------------|-------|------|
| `check` | Verify all prerequisites | Windows | 10s |
| `build` | Compile Go binaries | Windows | 30s |
| `deb` | Create .deb packages | WSL | 1-2 min |
| `iso` | Create bootable ISO | WSL | 2-5 min |
| `qcow2` | Create QEMU image | WSL | 2-5 min |
| `clean` | Remove dist/ | Windows | 5s |

## After Building

Once the ISO is created, you can:

1. **Boot in VM**:
   ```powershell
   # QEMU example:
   qemu-system-x86_64 -cdrom dist/crateos-0.1.0-dev.iso -m 2048 -enable-kvm
   ```

2. **Boot on USB** (on Linux):
   ```bash
   sudo dd if=dist/crateos-0.1.0-dev.iso of=/dev/sdX bs=4M
   sudo sync
   ```

3. **Boot on USB** (on Windows):
   - Use [Balena Etcher](https://www.balena.io/etcher/)
   - Or [Rufus](https://rufus.ie/)

## Quick Reference Commands

```powershell
# Install Ubuntu
wsl.exe --install -d Ubuntu-24.04

# Start Ubuntu
wsl.exe -d Ubuntu

# Install build tools
wsl.exe -d Ubuntu -- sudo apt-get install -y wget p7zip-full xorriso make dpkg

# Check prerequisites
cd P:\CrateOS && .\build.ps1 check

# Build everything
.\build.ps1 build && .\build.ps1 deb && .\build.ps1 iso

# Check the ISO
Get-Item dist/crateos-*.iso
```

## System Requirements Met

After completing this setup:

**Windows**:
- ✓ Go 1.20+
- ✓ PowerShell 5.0+
- ✓ WSL2 enabled

**WSL2 Ubuntu**:
- ✓ Ubuntu 24.04 LTS installed
- ✓ wget (for downloads)
- ✓ p7zip-full (for ISO extraction)
- ✓ xorriso (for ISO rebuilding)
- ✓ make (for build orchestration)
- ✓ dpkg (for package creation)

## Need Help?

If something goes wrong:

1. Run `.\build.ps1 check` - this will show you exactly what's missing
2. Follow the error messages - they have specific fixes
3. See "Troubleshooting" section above
4. All commands can be re-run safely (idempotent)

## Success!

When `.\build.ps1 check` shows all green checkmarks, you're ready:

```
✓ All prerequisites satisfied. Ready to build.
```

Then build the ISO:
```powershell
.\build.ps1 build
.\build.ps1 deb
.\build.ps1 iso
```

Your bootable ISO will be at `dist/crateos-0.1.0-dev.iso`.
