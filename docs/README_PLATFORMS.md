# CrateOS Multi-Platform Build System Documentation

This directory contains comprehensive documentation for CrateOS's platform-specific build architecture supporting x86-64, Raspberry Pi 4/5, and Raspberry Pi Zero 2 W.

---

## Quick Navigation

### For New Users

Start here:

1. **[PLATFORM_BUILD_QUICKSTART.md](../PLATFORM_BUILD_QUICKSTART.md)** (5 min read)
   - Quick reference for building each platform
   - Build commands for Windows and Linux
   - File structure overview
   - Deployment instructions

### For Developers

Understand the architecture:

2. **[ARCHITECTURE_PLATFORM_BUILDS.md](../ARCHITECTURE_PLATFORM_BUILDS.md)** (15 min read)
   - Complete system design and implementation
   - Platform specifications and constraints
   - Build system details (Makefile and build.ps1)
   - Runtime platform detection
   - Configuration inheritance model
   - Future extensibility

### For Module Authors

Learn about platform support:

3. **[MODULE_PLATFORM_CONSTRAINTS.md](MODULE_PLATFORM_CONSTRAINTS.md)** (10 min read)
   - Module metadata schema with platform support
   - How to declare platform compatibility in module.yaml
   - Platform-specific configuration defaults
   - Resource requirements per platform
   - Agent behavior and installation constraints
   - Examples for different module categories

### For System Administrators

Reference material:

4. **[PLATFORM_SUPPORT.md](PLATFORM_SUPPORT.md)** (10 min read)
   - Detailed platform specifications
   - Module availability matrix
   - Packaging strategy per platform
   - Deployment recommendations
   - Image distribution and naming

---

## Document Map

### Top-Level (in root)

| File | Purpose | Audience |
| ---- | ------- | --------- |
| `PLATFORM_BUILD_QUICKSTART.md` | Quick start guide | All users |
| `ARCHITECTURE_PLATFORM_BUILDS.md` | Complete architecture reference | Developers, architects |

### In docs/

| File | Purpose | Audience |
| ---- | ------- | --------- |
| `PLATFORM_SUPPORT.md` | Platform specifications & capabilities | DevOps, module authors |
| `MODULE_PLATFORM_CONSTRAINTS.md` | Module platform metadata schema | Module authors, developers |
| `README_PLATFORMS.md` | This file - documentation index | All users |

### Related Documentation

| File | Purpose |
| ---- | ------- |
| `BUILD_SYSTEM.md` | Original build system (now deprecated, see ARCHITECTURE_PLATFORM_BUILDS.md) |
| `MODULE_SPEC.md` | Base module specification |
| `MODULE_REGISTRY.md` | Module registry model |

---

## Key Concepts

### Platform Identifier

A single `PLATFORM` key controls the entire build:

```bash
PLATFORM ∈ {x86, rpi, rpi0}
```

Determines:
- Go cross-compilation target (GOOS/GOARCH)
- Base operating system (Ubuntu 24.04 vs RPi OS)
- Package set and dependencies
- Version suffix for distribution
- Runtime feature availability

### Build Commands

**Windows** (via WSL2):
```powershell
.\build.ps1 build-{x86|rpi|rpi0}  # Compile binaries
.\build.ps1 deb-{x86|rpi|rpi0}    # Create packages
.\build.ps1 image-{x86|rpi|rpi0}  # Build images
```

**Linux/WSL**:
```bash
make PLATFORM={x86|rpi|rpi0} build    # Compile binaries
make PLATFORM={x86|rpi|rpi0} deb      # Create packages
make PLATFORM={x86|rpi|rpi0} image-*  # Build images
```

### Supported Platforms

1. **x86-64** (Ubuntu 24.04, full features)
   - Servers, VMs, desktops
   - Full toolchain, virtualization, diagnostics
   - 98 packages

2. **Raspberry Pi 4/5** (RPi OS, moderate features)
   - Edge devices, IoT hubs, home automation
   - GPIO/I2C/SPI support, reduced diagnostics
   - 72 packages

3. **Raspberry Pi Zero 2 W** (RPi OS Lite, minimal)
   - Bastion hosts, VPN endpoints, sensors
   - SSH, WireGuard, GPIO support only
   - 28 packages, 512MB RAM with swap

---

## Getting Started

### Step 1: Choose Your Platform

- **x86**: Full-featured server/VM distribution
- **RPi 4/5**: Edge computing, multiple services
- **RPi Zero 2**: Single service, minimal resources

### Step 2: Build

See `PLATFORM_BUILD_QUICKSTART.md` for exact commands.

### Step 3: Deploy

- **x86**: Boot ISO in VM or physical hardware
- **RPi 4/5**: Flash .img to microSD card
- **RPi Zero 2**: Flash .img to 4GB+ microSD card

### Step 4: Configure Modules

Use the CrateOS panel/TUI to install and configure services. The system respects platform constraints automatically.

---

## Module Development

### Declaring Platform Support

Add to your `module.yaml`:

```yaml
metadata:
  id: my-service
  version: 1.0.0
  
  # Platform support
  platforms:
    - x86              # Full support
    - rpi              # Limited support
    - rpi0: "never"    # Not supported (512MB RAM)
  
  # Platform-specific defaults
  platformDefaults:
    x86:
      workers: 8
      cache_mb: 512
    rpi:
      workers: 2
      cache_mb: 128
```

See `MODULE_PLATFORM_CONSTRAINTS.md` for complete schema.

### Detecting Platform at Runtime

In Go code:

```go
import "github.com/crateos/crateos/internal/platform"

if platform.IsResourceConstrained() {
    // Pi Zero 2: minimal mode
} else if platform.IsRaspberryPi() {
    // Pi 4/5: moderate mode
} else {
    // x86: full power
}
```

---

## Frequently Asked Questions

### Q: Can I cross-compile on Windows?

**A**: Yes! WSL2 handles cross-compilation automatically. You can build ARM binaries on x86 Windows via `build.ps1`.

### Q: Which platform should I target first?

**A**: Start with x86 (full feature set), then test on RPi 4/5 (moderate constraints), finally RPi Zero 2 (extreme constraints).

### Q: Can I deploy a service to multiple platforms?

**A**: Yes! Use platform-specific defaults in `module.yaml` to tune configuration per platform.

### Q: What if my service isn't suitable for RPi Zero 2?

**A**: Declare `rpi0: "never"` in the platforms list. The agent will prevent installation and guide users to upgrade.

### Q: How do I handle hardware-specific features (GPIO, etc.)?

**A**: Use `platform.IsARM64()` to detect ARM devices and enable GPIO support conditionally.

---

## Architecture Highlights

### Layered Configuration

Platform-specific settings are inherited and overridden:

```
seed-defaults.env (x86 base)
    ↓
    seed-defaults-rpi.env (RPi 4/5 overrides)
    ↓
    seed-defaults-rpi0.env (Pi Zero 2 overrides)
```

### Package Layering

Each platform has tailored dependencies:

```
packages.yaml (baseline, not used directly)
    ├── packages-x86.yaml (full toolchain)
    ├── packages-rpi.yaml (reduced diagnostics)
    └── packages-rpi0.yaml (minimal only)
```

### Build-Time Constants

Platform is injected at compile time and accessible at runtime:

```go
const BuildTarget = "x86"  // or "rpi", "rpi0"
```

This enables conditional logic without runtime detection overhead.

---

## File Organization Summary

```
CrateOS/
├── PLATFORM_BUILD_QUICKSTART.md         ← Start here for quick reference
├── ARCHITECTURE_PLATFORM_BUILDS.md      ← Complete system design
├── Makefile                             ← Build orchestration
├── build.ps1                            ← Windows driver
│
├── images/
│   ├── common/                          ← Shared utilities
│   │   ├── seed-defaults.env
│   │   ├── seed-defaults-rpi.env
│   │   └── seed-defaults-rpi0.env
│   ├── x86/                             ← x86 ISO builder
│   ├── rpi/                             ← RPi 4/5 image builder
│   └── rpi0/                            ← RPi Zero 2 image builder
│
├── packaging/config/
│   ├── packages.yaml
│   ├── packages-x86.yaml
│   ├── packages-rpi.yaml
│   └── packages-rpi0.yaml
│
├── internal/platform/
│   └── platform.go                      ← BuildTarget constant + helpers
│
└── docs/
    ├── README_PLATFORMS.md              ← This file
    ├── PLATFORM_SUPPORT.md              ← Platform specifications
    └── MODULE_PLATFORM_CONSTRAINTS.md   ← Module metadata schema
```

---

## Implementation Checklist

✅ Platform key abstraction (x86, rpi, rpi0)
✅ Makefile with platform-specific targets
✅ build.ps1 with cross-compile support
✅ Platform-specific image builders
✅ Layered configuration (seed defaults)
✅ Platform-specific package sets
✅ Module platform metadata schema
✅ Runtime platform detection functions
✅ Build-time constant injection
✅ Comprehensive documentation

---

## Next Steps

1. **Try it**: See `PLATFORM_BUILD_QUICKSTART.md`
2. **Understand it**: Read `ARCHITECTURE_PLATFORM_BUILDS.md`
3. **Deploy it**: Choose your target platform and build
4. **Develop for it**: Use `MODULE_PLATFORM_CONSTRAINTS.md` to make modules platform-aware
5. **Extend it**: Add new platforms following the pattern

---

## Support and Questions

For implementation details, refer to the specific documentation files above.
For general CrateOS questions, see the main README and other docs.

---

**Status**: ✅ Production-ready multi-platform build system

**Last Updated**: 2026-03-10
**Scope**: x86-64, Raspberry Pi 4/5, Raspberry Pi Zero 2 W
