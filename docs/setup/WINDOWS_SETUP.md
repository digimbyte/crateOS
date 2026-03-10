# CrateOS Windows Build Setup

Your system has WSL2 installed but **no Ubuntu distro** is available. You need to install Ubuntu to build CrateOS packages and ISO.

## Quick Setup (3 minutes)

### Option 1: Install Ubuntu from Microsoft Store (Easiest)

1. **Open Microsoft Store** and search for "Ubuntu"
2. **Click "Ubuntu 24.04 LTS"** (or latest version)
3. **Click "Install"**
4. **Wait for installation** to complete (~2-3 minutes)
5. **Launch Ubuntu** from Start menu
6. **Create initial user** when prompted
7. **Run this to verify**:
   ```powershell
   wsl.exe --list --verbose
   # Should show: Ubuntu ... Stopped ... 2
   ```

### Option 2: Install Ubuntu via Command Line

```powershell
# List available distros
wsl.exe --list --online

# Install Ubuntu (latest)
wsl.exe --install -d Ubuntu

# Or specific version
wsl.exe --install -d Ubuntu-24.04

# Launch Ubuntu
wsl.exe -d Ubuntu
```

### Option 3: Use WSL in Docker Desktop (If You Already Have It)

If Docker Desktop is running, you can use its Ubuntu distro:

```powershell
# Check Docker Desktop has a usable distro
docker run --rm ubuntu:24.04 sh -c "apt-get update && apt-get install -y xorriso p7zip-full wget"
```

But it's better to have a standalone Ubuntu for CrateOS builds.

---

## Verify Installation

After installing Ubuntu, run this to check everything is ready:

```powershell
cd P:\CrateOS
.\wsl-init.ps1
```

Expected output:
```
==> CrateOS WSL2 Initialization Check

1. Checking WSL2 installation...
[OK] wsl.exe found
2. Checking WSL version...
[OK] WSL version: 2.x.x
3. Checking installed distros...
[OK] Found distro: Ubuntu
4. Testing WSL VM startup...
[OK] WSL started successfully
5. Checking drive access in WSL...
[OK] P: drive mounted as: /mnt/p
6. Checking WSL build tools...
[OK] wget installed
[OK] 7z installed
[OK] xorriso installed
[OK] dpkg installed
7. Checking Go installation...
[OK] go version go1.x...

==> Summary
✓ All checks passed. Ready to build!
```

---

## Install Build Tools in Ubuntu

Once Ubuntu is installed and running, you may need to install the build tools:

```bash
# In WSL (Ubuntu terminal)
sudo apt-get update
sudo apt-get install -y \
    wget \
    p7zip-full \
    xorriso \
    dpkg \
    make
```

---

## Now Build CrateOS

Once Ubuntu is set up and verified:

```powershell
cd P:\CrateOS

# Step 1: Compile binaries (native on Windows)
.\build.ps1 build

# Step 2: Create .deb packages (via WSL)
.\build.ps1 deb

# Step 3: Create ISO (via WSL)
.\build.ps1 iso

# Result: dist/crateos-0.1.0-dev.iso
```

---

## Troubleshooting

### "WSL failed to start"

```powershell
# Restart WSL VM
wsl.exe --shutdown

# Then try again
.\build.ps1 deb
```

### "P: drive not accessible in WSL"

If P: is a network drive, mount it in WSL:

```bash
# In WSL terminal
sudo mkdir -p /mnt/network-p
sudo mount -t drvfs 'P:' /mnt/network-p

# Then update build.ps1 or set WSL default mount location
# (Usually not needed - WSL auto-mounts local drives)
```

### "Build tools missing"

If you see warnings about missing tools (wget, 7z, xorriso):

```bash
sudo apt-get update
sudo apt-get install -y \
    wget \
    p7zip-full \
    xorriso
```

### "WSL version too old"

If you see `[WARN] Could not detect WSL version`, update:

```powershell
wsl.exe --update
wsl.exe --shutdown
```

---

## Current System State

**Current Status**:
- ✓ WSL2 installed
- ✓ Go installed
- ✗ Ubuntu distro NOT installed
- ✗ Build tools NOT available

**To Fix**: Install Ubuntu (see "Quick Setup" above)

---

## Next Steps

1. Install Ubuntu using one of the methods above
2. Run `.\wsl-init.ps1` to verify everything works
3. Run `.\build.ps1 build` to compile binaries
4. Run `.\build.ps1 deb` to create packages
5. Run `.\build.ps1 iso` to create the ISO

Total time: ~5-10 minutes after Ubuntu is installed.
