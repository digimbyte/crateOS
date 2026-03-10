# CrateOS Build System - Safety & Validation

**Date**: March 9, 2026  
**Status**: ✅ Build system now has comprehensive prerequisite validation

---

## What Changed

The build system now **validates all prerequisites before attempting any build operations**. This prevents broken build pipelines and failed ISO creation.

## New Features

### 1. Prerequisite Validation (`.\build.ps1 check`)

A new command that checks your entire system:

```powershell
cd P:\CrateOS
.\build.ps1 check
```

**Validates**:
- ✓ Go 1.20+ installed on Windows
- ✓ PowerShell 5.0+ available
- ✓ WSL2 installed and running
- ✓ Ubuntu distro installed in WSL2
- ✓ All required build tools available:
  - wget (download)
  - 7z (extraction)
  - xorriso (ISO rebuild)
  - dpkg (packaging)
  - make (orchestration)

### 2. Pre-Flight Checks on Every Build Target

Every build command now validates prerequisites before starting:

```powershell
.\build.ps1 build   # Checks Windows prerequisites
.\build.ps1 deb     # Checks Windows + WSL prerequisites
.\build.ps1 iso     # Checks Windows + WSL prerequisites
.\build.ps1 qcow2   # Checks Windows + WSL prerequisites
```

If any prerequisite is missing, the build **fails immediately** with:
- Clear error message
- Exact missing requirement
- How to fix it

### 3. Improved Error Messages

Old error:
```
Failed to convert repo path for WSL: P:\CrateOS
```

New errors are specific and actionable:
```
ERROR: WSL2 VM cannot start. Try: wsl --shutdown

ERROR: No Ubuntu distro found in WSL. Install with: wsl --install -d Ubuntu-24.04
Available distros: docker-desktop-data, docker-desktop

ERROR: Missing build tools in WSL: xorriso, 7z
Install with: wsl -d Ubuntu -- sudo apt-get install -y xorriso p7zip-full
```

## Build Safety Guarantees

### ✅ Prerequisite Validation
- All checks run **before** any build operations
- Comprehensive validation of Windows and WSL environments
- No half-built artifacts on failure

### ✅ Fail-Fast Design
- Build stops immediately if any prerequisite is missing
- Clear explanation of what's wrong
- No confusing intermediate errors

### ✅ Idempotent Prerequisites
- Check command can be run multiple times safely
- Each check is independent (no side effects)
- Can verify after installation to confirm readiness

### ✅ Cross-Platform Validation
- Windows tools checked on Windows
- WSL tools checked in WSL VM
- Separate validation for each platform

### ✅ Actionable Guidance
- Every error includes the fix
- Exact commands to run to resolve issues
- Links to documentation and resources

## Example: Safe Build Flow

```powershell
# User runs build without prerequisites
cd P:\CrateOS
.\build.ps1 iso

# System validates:
# ✓ Go installed
# ✓ PowerShell present
# ✗ WSL2 VM cannot start
# ✗ No Ubuntu distro

# BUILD FAILS IMMEDIATELY with:
# "No Ubuntu distro found in WSL. Install with: wsl --install -d Ubuntu-24.04"
# "WSL2 VM cannot start. Try: wsl --shutdown"

# User follows the guidance, then:
wsl.exe --install -d Ubuntu-24.04
wsl.exe --shutdown

# User tries again
.\build.ps1 iso

# System validates again:
# ✓ Go installed
# ✓ PowerShell present
# ✓ WSL2 installed
# ✓ WSL2 VM starts
# ✓ Ubuntu 24.04 found
# ✓ All build tools present

# BUILD PROCEEDS to create ISO
```

## Validation Checks by Build Target

### `.\build.ps1 check`
Validates everything and reports status.

### `.\build.ps1 build`
Windows only (compile Go binaries):
- ✓ Go installed
- ✓ PowerShell available

### `.\build.ps1 deb`
Windows + WSL2:
- ✓ Go installed
- ✓ PowerShell available
- ✓ WSL2 installed
- ✓ WSL2 VM running
- ✓ Ubuntu distro installed
- ✓ Build tools: wget, 7z, xorriso, dpkg, make

### `.\build.ps1 iso`
Windows + WSL2 (same as deb):
- ✓ Go installed
- ✓ PowerShell available
- ✓ WSL2 installed
- ✓ WSL2 VM running
- ✓ Ubuntu distro installed
- ✓ Build tools: wget, 7z, xorriso, dpkg, make

### `.\build.ps1 qcow2`
Windows + WSL2 (same as iso):
- ✓ Go installed
- ✓ PowerShell available
- ✓ WSL2 installed
- ✓ WSL2 VM running
- ✓ Ubuntu distro installed
- ✓ Build tools: wget, 7z, xorriso, dpkg, make

## Implementation Details

### Windows Prerequisites Function

```powershell
# Checks for:
# - Go command available
# - PowerShell version
# - WSL2 executable

# Fails immediately if any missing
if (-not (Get-Command go)) {
    Write-Error "Go is NOT installed. Install from https://go.dev/dl/"
    exit 1
}
```

### WSL Prerequisites Function

```powershell
# Checks:
# - WSL2 executable
# - WSL2 VM can start
# - Ubuntu distro installed
# - All build tools present in Ubuntu

# Each check validates independently
# Clear error message if any fails
```

### Prerequisite Levels

```
Test-Prerequisites -Level windows
# Only Windows tools (fast, for .\build.ps1 build)

Test-Prerequisites -Level wsl
# Only WSL tools (for custom WSL operations)

Test-Prerequisites -Level all
# Everything (for packaging targets)
```

## Documentation

New guides created:

| File | Purpose |
|------|---------|
| `docs/setup/SETUP_GUIDE.md` | Step-by-step setup with troubleshooting |
| `docs/build/BUILD_SAFETY.md` | This file - safety architecture |
| `wsl-init.ps1` | Standalone diagnostic script |
| Enhanced `build.ps1` | With comprehensive validation |

## Running the Checks

### Quick Status Check
```powershell
.\build.ps1 check
```

Tells you exactly what's installed and what's missing.

### Before Building
```powershell
# Always safe to check first
.\build.ps1 check

# Build only runs if check passes
.\build.ps1 iso
```

### Manual Debugging
```powershell
# If you want to see detailed validation output
Set-ExecutionPolicy Bypass -Scope Process
& "P:\CrateOS\build.ps1" check
```

## No Broken Builds

With these safety improvements, you cannot:

- ❌ Start a build without required tools
- ❌ Get confused by cryptic path errors
- ❌ Partially build and leave broken artifacts
- ❌ Waste time on builds that will fail
- ❌ Have to guess what's missing

Instead:

- ✅ Know exactly what's needed upfront
- ✅ Get clear instructions to fix any issues
- ✅ Build only when prerequisites are met
- ✅ Get specific errors with solutions
- ✅ Know the system is in a good state

## Verification Commands

```powershell
# Check if ready to build
.\build.ps1 check

# See what Go is installed
go version

# See what WSL distros are available
wsl.exe --list --verbose

# See what Ubuntu packages are available
wsl.exe -d Ubuntu -- apt-cache search xorriso

# Manually verify a tool exists in WSL
wsl.exe -d Ubuntu -- command -v xorriso
```

## Success Indicators

When everything is ready, you should see:

```
✓ All prerequisites satisfied. Ready to build.
```

Then you can confidently run:

```powershell
.\build.ps1 build
.\build.ps1 deb
.\build.ps1 iso
```

And the ISO will be created at `dist/crateos-0.1.0-dev.iso`.

---

## Summary

The CrateOS build system now has **production-grade validation**:

1. **Comprehensive checks** before any build operation
2. **Clear error messages** with specific fixes
3. **Fail-fast behavior** to prevent wasted time
4. **No broken pipelines** - validates everything upfront
5. **Cross-platform support** for Windows and Linux

**No more broken builds!**
