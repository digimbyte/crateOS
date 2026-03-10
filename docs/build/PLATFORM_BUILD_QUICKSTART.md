# CrateOS Platform-Specific Build - Quick Start

This document provides a quick reference for building CrateOS for different hardware platforms.

---

## Supported Platforms

| Platform      | Build ID | Arch  | Image Format | Specs                    |
| ------------- | -------- | ----- | ------------ | ------------------------ |
| x86-64        | `x86`    | amd64 | ISO/QCOW2    | Ubuntu 24.04 LTS, 1GB+   |
| Raspberry Pi 4/5 | `rpi` | arm64 | IMG          | Raspberry Pi OS, 2GB+    |
| Raspberry Pi Zero 2 W | `rpi0` | arm64 | IMG (lite) | Raspberry Pi OS Lite, 512MB |

---

## Build from Windows (PowerShell)

### Build x86 ISO
```powershell
cd P:\CrateOS
..\build.ps1 build-x86        # Compile x86 binaries
..\build.ps1 deb-x86          # Package into .deb files
..\build.ps1 image-x86        # Build ISO image
# Output: dist/crateos-0.1.0+noble1.iso
```

### Build Raspberry Pi 4/5 Image
```powershell
..\build.ps1 build-rpi        # Compile ARM64 binaries
..\build.ps1 deb-rpi          # Package into .deb files (ARM64)
..\build.ps1 image-rpi        # Build RPi OS image
# Output: dist/crateos-rpi-0.1.0+rpi1.img
```

### Build Raspberry Pi Zero 2 W Image
```powershell
..\build.ps1 build-rpi0       # Compile ARM64 binaries
..\build.ps1 deb-rpi0         # Package into .deb files (ARM64)
..\build.ps1 image-rpi0       # Build minimal RPi image
# Output: dist/crateos-rpi0-0.1.0+rpi0-1.img
```

### Build All Platforms
```powershell
..\build.ps1 build-x86        # x86 binaries
..\build.ps1 build-rpi        # RPi 4/5 binaries
..\build.ps1 build-rpi0       # RPi Zero 2 binaries
..\build.ps1 deb-x86          # x86 packages
..\build.ps1 deb-rpi          # RPi 4/5 packages
..\build.ps1 deb-rpi0         # RPi Zero 2 packages
..\build.ps1 image-x86        # x86 ISO
..\build.ps1 image-rpi        # RPi 4/5 image
..\build.ps1 image-rpi0       # RPi Zero 2 image
```

---

## Build from Linux (Make)

### Build x86 ISO
```bash
cd /path/to/CrateOS
make PLATFORM=x86 build-x86   # Compile x86 binaries
make PLATFORM=x86 deb-x86     # Package into .deb files
make PLATFORM=x86 image-x86   # Build ISO image
# Output: dist/crateos-0.1.0+noble1.iso
```

### Build Raspberry Pi 4/5 Image
```bash
make PLATFORM=rpi build-rpi   # Compile ARM64 binaries
make PLATFORM=rpi deb-rpi     # Package into .deb files (ARM64)
make PLATFORM=rpi image-rpi   # Build RPi OS image
# Output: dist/crateos-rpi-0.1.0+rpi1.img
```

### Build Raspberry Pi Zero 2 W Image
```bash
make PLATFORM=rpi0 build-rpi0 # Compile ARM64 binaries
make PLATFORM=rpi0 deb-rpi0   # Package into .deb files (ARM64)
make PLATFORM=rpi0 image-rpi0 # Build minimal RPi image
# Output: dist/crateos-rpi0-0.1.0+rpi0-1.img
```

### View All Targets
```bash
make help                     # Display all available targets
```

---

## File Structure

The build system is organized by platform:

```
CrateOS/
├── Makefile                          # Main build orchestration
├── build.ps1                         # Windows PowerShell driver
│
├── images/
│   ├── common/                       # Shared build utilities
│   │   ├── seed-defaults.env         # x86 base defaults
│   │   ├── seed-defaults-rpi.env     # RPi 4/5 overrides
│   │   ├── seed-defaults-rpi0.env    # RPi Zero 2 overrides
│   │   └── render-required-packages.sh
│   │
│   ├── x86/                          # x86-64 builds
│   │   ├── build.sh                  # ISO builder
│   │   └── autoinstall/
│   │       ├── user-data.template
│   │       └── meta-data
│   │
│   ├── rpi/                          # Raspberry Pi 4/5 builds
│   │   ├── build.sh                  # RPi OS image builder
│   │   └── config/
│   │
│   └── rpi0/                         # Raspberry Pi Zero 2 W builds
│       ├── build.sh                  # Minimal RPi OS builder
│       └── config/
│
├── packaging/config/
│   ├── packages.yaml                 # Base packages (Ubuntu)
│   ├── packages-x86.yaml             # x86-specific packages
│   ├── packages-rpi.yaml             # RPi 4/5 packages (reduced)
│   └── packages-rpi0.yaml            # RPi Zero 2 packages (minimal)
│
├── docs/
│   └── PLATFORM_SUPPORT.md           # Comprehensive platform documentation
│
└── internal/platform/
    └── BUILD_TARGET.go               # Injected at build time (PLATFORM constant)
```

---

## Build Output

Each platform produces distinct artifacts:

### x86
- **Binary**: `dist/bin/crateos.exe` (amd64)
- **Packages**: `dist/crateos_0.1.0+noble1_amd64.deb`
- **Image**: `dist/crateos-0.1.0+noble1.iso` (2GB, bootable)

### Raspberry Pi 4/5
- **Binary**: `dist/bin/crateos.exe` (arm64)
- **Packages**: `dist/crateos_0.1.0+rpi1_arm64.deb`
- **Image**: `dist/crateos-rpi-0.1.0+rpi1.img` (500MB-1GB)

### Raspberry Pi Zero 2 W
- **Binary**: `dist/bin/crateos.exe` (arm64)
- **Packages**: `dist/crateos_0.1.0+rpi0-1_arm64.deb`
- **Image**: `dist/crateos-rpi0-0.1.0+rpi0-1.img` (350-500MB, lite)

---

## Environment Variables

### Build-Time Constants
The `PLATFORM` identifier is injected via `-ldflags`:

```bash
# Example: Build x86 with custom version
VERSION=1.2.3 make PLATFORM=x86 build-x86
```

### Platform Detection at Runtime
Code can detect build platform:

```go
import "github.com/crateos/crateos/internal/platform"

if platform.BuildTarget == "rpi0" {
    // Enable minimal mode for Pi Zero 2
    agent.MaxWorkers = 1
    diagnostics.Disable()
}
```

---

## Deployment

### x86 (Ubuntu 24.04 ISO)
Flash to USB or boot in VM:
```bash
# USB
sudo dd if=crateos-0.1.0+noble1.iso of=/dev/sdX bs=4M && sync

# QEMU
qemu-system-x86_64 -enable-kvm -m 2G -cdrom crateos-0.1.0+noble1.iso
```

### Raspberry Pi 4/5
Flash to microSD:
```bash
# Balena Etcher (recommended)
etcher --flash-to-device /dev/sdX crateos-rpi-0.1.0+rpi1.img

# or dd (careful with device!)
xzcat crateos-rpi-0.1.0+rpi1.img.xz | dd of=/dev/sdX bs=4M && sync
```

### Raspberry Pi Zero 2 W
Flash to microSD (lite variant):
```bash
# Etcher (recommended, handles compression)
etcher --flash-to-device /dev/sdX crateos-rpi0-0.1.0+rpi0-1.img

# or xz + dd
xzcat crateos-rpi0-0.1.0+rpi0-1.img.xz | dd of=/dev/sdX bs=4M && sync
```

First boot takes 5-10 minutes on RPi Zero 2 due to package installation from swap.

---

## Troubleshooting

### x86 Build
```bash
# Check prerequisites
make help

# Verbose build
make PLATFORM=x86 build-x86 -v

# Clean rebuild
make clean && make PLATFORM=x86 build-x86 deb-x86 image-x86
```

### RPi Builds
```bash
# Cross-compile check
file dist/bin/crateos.exe        # Should show "ELF 64-bit ARM"

# Verify image
file dist/crateos-rpi-*.img      # Should show "DOS/MBR boot sector"

# Check package sizes
ls -lh dist/crateos*.deb
```

---

## Advanced: Custom Versions

Build with custom version tags:

```bash
# Windows
..\build.ps1 -Version "1.2.3-custom" build-rpi0

# Linux
VERSION=1.2.3-custom make PLATFORM=rpi0 build-rpi0 deb-rpi0 image-rpi0

# Result: crateos-rpi0-1.2.3-custom_arm64.deb
```

---

## Advanced: Module Platform Constraints

Modules declare platform support in `module.yaml`:

```yaml
metadata:
  id: postgres
  version: 16.0.0
  platforms:
    - x86              # Full support
    - rpi              # Limited (2GB+ RAM, slower I/O)
    - rpi0: "never"    # Explicitly unsupported
  
  platformDefaults:
    x86:
      max_connections: 200
      shared_buffers: 1GB
    rpi:
      max_connections: 50
      shared_buffers: 256MB
```

The agent respects these constraints during installation.

---

## Next Steps

1. **Read** `docs/PLATFORM_SUPPORT.md` for detailed platform capabilities
2. **Build** your target platform using the commands above
3. **Deploy** the resulting image to hardware
4. **Configure** modules via the CrateOS panel/TUI

For advanced module development with platform-specific behavior, see:
- `docs/MODULE_SPEC.md`
- `docs/MODULE_REGISTRY.md`
- `packaging/config/packages-{x86,rpi,rpi0}.yaml`
