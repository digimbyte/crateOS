# CrateOS Platform Support Matrix

This document defines the platform-specific build architecture, feature availability, and module compatibility across supported hardware targets.

---

## Supported Platforms

### x86-64 (Server/Desktop/VM)
- **Base OS**: Ubuntu 24.04 LTS (Noble Numbat)
- **Architecture**: AMD64 (amd64)
- **Image Format**: ISO (autoinstall) or QCOW2 (VM)
- **Minimum specs**: 1GB RAM, 10GB disk
- **Typical specs**: 2GB+ RAM, SSD, quad-core CPU
- **Build identifier**: `x86`
- **Version suffix**: `0.1.0+noble1`

**Capabilities**:
- Full development toolchain (build-essential, python3-pip, etc.)
- Virtualization support (QEMU, libvirt)
- Heavy diagnostic tools (smartmontools, nvme-cli, etc.)
- All optional modules supported
- Docker containers with full performance
- PostgreSQL and heavy databases
- Media processing (ffmpeg, imagemagick)

### Raspberry Pi 4/5
- **Base OS**: Raspberry Pi OS 64-bit (Bookworm)
- **Architecture**: ARM64 (arm64)
- **Image Format**: IMG (microSD image)
- **Minimum specs**: 2GB RAM, 8GB microSD
- **Typical specs**: 4GB RAM, Class 10 microSD, 1.8GHz quad-core
- **Build identifier**: `rpi`
- **Version suffix**: `0.1.0+rpi1`

**Capabilities**:
- Lightweight diagnostics (btop, ncdu, ethtool)
- GPIO/SPI/I2C hardware interfaces
- Docker containers (resource-constrained)
- PostgreSQL (acceptable performance)
- Network services (nginx, SSH)
- Home automation, IoT applications
- Camera module support (optional)

**Restrictions**:
- No virtualization (qemu-system-x86, libvirt)
- No media processing (ffmpeg, imagemagick)
- No x86 thermal tools

### Raspberry Pi Zero 2 W
- **Base OS**: Raspberry Pi OS Lite 64-bit (Bookworm)
- **Architecture**: ARM64 (arm64)
- **Image Format**: IMG (microSD image, lite variant)
- **Minimum specs**: 512MB RAM, 4GB microSD
- **Typical specs**: 512MB RAM, Class 10 microSD, 1GHz equivalent
- **Build identifier**: `rpi0`
- **Version suffix**: `0.1.0+rpi0-1`
- **Form factor**: Credit-card sized, passive cooling

**Capabilities**:
- Minimal core services (SSH, CrateOS agent)
- GPIO/SPI/I2C hardware interfaces
- Lightweight networking (SSH, WireGuard)
- Single lightweight service + agent

**Severe Restrictions**:
- 512MB RAM requires careful workload planning
- No development tools (build-essential prohibited)
- No docker (RAM impossible)
- No python3-pip (too heavy)
- No powershell (resource killer)
- No network-manager (use dhcpcd only)
- Swap to microSD mandatory (5-10 second first-boot overhead)
- Maximum 1-2 application services + agent
- No heavy diagnostics

**Recommended Uses**:
- SSH gateway/bastion host
- WireGuard VPN endpoint
- GPIO sensor reader (light polling only)
- Network monitoring probe
- Message broker client (MQTT, lightweight only)
- Remote syslog forwarder

---

## Build System

### Build Targets by Platform

```bash
# x86 builds (Ubuntu/ISO)
make PLATFORM=x86 build-x86       # Compile x86 binaries
make PLATFORM=x86 deb-x86         # Create .deb packages (amd64)
make PLATFORM=x86 image-x86       # Build ISO image
make PLATFORM=x86 qcow2           # Build QCOW2 VM image

# RPi 4/5 builds (Raspberry Pi OS)
make PLATFORM=rpi build-rpi       # Compile ARM64 binaries
make PLATFORM=rpi deb-rpi         # Create .deb packages (arm64)
make PLATFORM=rpi image-rpi       # Build RPi OS image

# RPi Zero 2 builds (minimal)
make PLATFORM=rpi0 build-rpi0     # Compile ARM64 binaries
make PLATFORM=rpi0 deb-rpi0       # Create .deb packages (arm64)
make PLATFORM=rpi0 image-rpi0     # Build minimal RPi image
```

### Environment Variables

The Makefile automatically sets GOOS/GOARCH based on PLATFORM:

| PLATFORM | GOOS  | GOARCH |
| -------- | ----- | ------ |
| x86      | linux | amd64  |
| rpi      | linux | arm64  |
| rpi0     | linux | arm64  |

### Version Tagging

Each platform has a distinct version suffix for ISO/image labels:

| PLATFORM | Version Pattern    | Example          |
| -------- | ------------------ | ---------------- |
| x86      | 0.1.0+noble1       | Ubuntu release   |
| rpi      | 0.1.0+rpi1         | RPi release      |
| rpi0     | 0.1.0+rpi0-1       | RPi Zero release |

---

## Module Platform Constraints

### Module Metadata

Each CrateOS module declares platform support via `module.yaml`:

```yaml
metadata:
  id: postgres
  version: 16.0.0
  platforms:
    - x86              # Full support: 4GB+ RAM, SSD, full toolchain
    - rpi              # Limited: acceptable on 4GB+, slower I/O
    - rpi0: "never"    # Explicitly unsupported due to RAM constraints
  
  platformDefaults:
    x86:
      max_connections: 200
      shared_buffers: 1GB
    rpi:
      max_connections: 50
      shared_buffers: 256MB
```

### Module Availability Matrix

| Module      | x86 | RPi 4/5 | RPi Zero 2 | Notes                          |
| ----------- | --- | ------- | ---------- | ------------------------------ |
| crateos     | ✓   | ✓       | ✓          | Core agent on all platforms    |
| nginx       | ✓   | ✓       | ✓          | Lightweight reverse proxy      |
| postgres    | ✓   | ✓ (dim) | ✗          | Too heavy for Zero 2           |
| docker      | ✓   | ✓ (dim) | ✗          | Impossible on 512MB RAM        |
| redis       | ✓   | ✓       | ~          | Possible but swap-heavy        |
| mosquitto   | ✓   | ✓       | ✓          | Lightweight MQTT broker        |
| wireguard   | ✓   | ✓       | ✓          | VPN endpoint                   |
| cockpit     | ✓   | ✗       | ✗          | Web UI too heavy for ARM       |
| lightdm     | ✓   | ✓ (dim) | ✗          | GUI needs 4GB+, not on Zero    |
| gcc         | ✓   | ✓ (dim) | ✗          | Dev toolchain excluded on Zero |

Legend:
- ✓ Full support, no caveats
- ✓ (dim) Works but resource-constrained
- ~ Possible but not recommended
- ✗ Unsupported due to platform limitations

---

## Packaging Strategy

### x86 (Full)
- **Config**: `packaging/config/packages-x86.yaml`
- **Size**: ~2GB installed
- **Includes**: All x86-specific tools, diagnostics, virtualization

### RPi 4/5 (Reduced)
- **Config**: `packaging/config/packages-rpi.yaml`
- **Size**: ~1.2GB installed
- **Excludes**: x86-only tools, heavy diagnostics, virtualization
- **Includes**: GPIO libraries, camera support, lighter diagnostics

### RPi Zero 2 W (Minimal)
- **Config**: `packaging/config/packages-rpi0.yaml`
- **Size**: ~600MB installed
- **Excludes**: All development tools, heavy packages, diagnostics
- **Includes**: Core SSH, WireGuard, minimal GPIO support
- **Critical**: Swap to microSD card (configured at build time)

---

## Configuration Inheritance

Platform-specific seed defaults are layered:

```
images/common/
├── seed-defaults.env          # Base (x86) defaults
├── seed-defaults-rpi.env      # RPi 4/5 overrides
└── seed-defaults-rpi0.env     # RPi Zero 2 stripped

Each file sources the previous and adds/overrides values.
```

### Example Variables

**seed-defaults.env** (x86 base):
```env
HOSTNAME=crateos
DEFAULT_USER=crate
```

**seed-defaults-rpi.env** (RPi 4/5):
```env
HOSTNAME=crateos-rpi
ARM_FREQ=1800
GPU_MEMORY_MB=128
```

**seed-defaults-rpi0.env** (RPi Zero 2):
```env
HOSTNAME=crateos-zero
ARM_FREQ=1000
GPU_MEMORY_MB=64
SWAP_SIZE_MB=512
```

---

## Build-Time Constants

The `PLATFORM` identifier is injected at compile time:

```go
// internal/platform/BUILD_TARGET.go
var BuildTarget = "${PLATFORM}"  // Injected via -ldflags

// Usage in code:
if platform.BuildTarget == "rpi0" {
    // Enable minimal mode
    agent.MaxWorkers = 1
    diagnostics.Disable()
}
```

---

## Deployment Recommendations

### x86 (Server/VM)
- Bare metal: Dell, Lenovo, HP servers
- VMs: KVM, Proxmox, ESXi, AWS, Azure
- Multiple services, high workload
- Full CrateOS feature set

### RPi 4/5
- Home automation server
- Kubernetes edge node
- Monitoring probe
- Development/testing
- Multiple services
- Most CrateOS modules available

### RPi Zero 2 W
- Remote SSH gateway
- VPN endpoint
- Sensor data collector
- Message broker client
- Single application per device
- Minimal feature set

---

## Image Distribution

Each platform produces distinct images:

- **x86**: `crateos-0.1.0+noble1.iso` (2GB, bootable)
- **RPi 4/5**: `crateos-rpi-0.1.0+rpi1.img` (500MB-1GB)
- **RPi Zero 2**: `crateos-rpi0-0.1.0+rpi0-1.img` (350-500MB, lite)

All images are distributed via the same release channel with platform indicated in filename.

---

## Future Platforms

Reserved identifiers for future expansion:

- `arm-generic` — generic ARMv7/ARMv8 boards
- `jetson` — NVIDIA Jetson (ML/AI edge)
- `ppc64` — PowerPC 64-bit
- `s390x` — IBM System z mainframe

The PLATFORM key will evolve to support these without breaking existing builds.
