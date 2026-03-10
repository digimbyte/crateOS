# Module Platform Constraints

This document defines how CrateOS modules declare and enforce platform compatibility constraints.

---

## Overview

Every CrateOS module can declare which platforms it supports via the `platforms` field in its `module.yaml` metadata. This allows:

1. **User guidance**: The panel/TUI warns when installing unsupported modules
2. **Agent enforcement**: The agent refuses installation on incompatible platforms
3. **Configuration tuning**: Platform-specific defaults for resource constraints
4. **Future extensibility**: Easy to add new platforms without breaking existing modules

---

## Module Metadata Structure

### Basic Declaration

```yaml
metadata:
  id: nginx
  version: 1.24.0
  
  # Platform support (required)
  platforms:
    - x86              # Supported on x86-64
    - rpi              # Supported on Raspberry Pi 4/5
    - rpi0             # Supported on Raspberry Pi Zero 2 W
```

### With Constraints

```yaml
metadata:
  id: postgres
  version: 16.0.0
  
  # Platform support with notes
  platforms:
    - x86              # Full support
    - rpi              # Limited: acceptable on 4GB+
    - rpi0: "never"    # Explicitly unsupported (512MB RAM)
  
  # Platform-specific configuration defaults
  platformDefaults:
    x86:
      # Full power
      max_connections: 200
      shared_buffers: 1GB
      work_mem: 4MB
      
    rpi:
      # Reduced but functional
      max_connections: 50
      shared_buffers: 256MB
      work_mem: 1MB
      
    # rpi0 is in "never" list, so no defaults needed
```

### Complete Example

```yaml
metadata:
  id: mosquitto
  version: 2.0.15
  
  version: 2.0.15
  description: "Lightweight MQTT message broker"
  
  # Platform support matrix
  platforms:
    - x86              # Full support
    - rpi              # Fully supported
    - rpi0             # Supported but monitor memory
  
  # Platform-specific tuning
  platformDefaults:
    x86:
      max_queued_messages: 1000
      message_size_limit: 0  # unlimited
      
    rpi:
      max_queued_messages: 500
      message_size_limit: 262144  # 256KB
      
    rpi0:
      max_queued_messages: 100
      message_size_limit: 65536   # 64KB
  
  # Resource requirements per platform
  platformRequirements:
    x86:
      min_ram_mb: 256
      min_disk_mb: 500
      
    rpi:
      min_ram_mb: 256
      min_disk_mb: 500
      
    rpi0:
      min_ram_mb: 32
      min_disk_mb: 200
```

---

## Platform Support Syntax

### Implicit Support (Recommended)

List platforms that are supported:

```yaml
platforms:
  - x86
  - rpi
  - rpi0
```

Meaning: Works on all three platforms with default configuration.

### Explicit Constraints

Use string values to indicate constraints:

```yaml
platforms:
  - x86                    # Supported (implicit)
  - rpi                    # Supported (implicit)
  - rpi0: "never"          # NOT supported (explicit rejection)
```

### Support Notes

Future syntax (for documentation):

```yaml
platforms:
  x86: "full"              # Full support
  rpi: "limited"           # Works but resource-constrained
  rpi0: "never"            # Not supported
```

---

## Platform-Specific Configuration

### platformDefaults

Each platform can have default configuration values:

```yaml
platformDefaults:
  x86:
    # These are the default config values for x86
    workers: 8
    cache_size_mb: 512
    
  rpi:
    # Reduced for ARM
    workers: 2
    cache_size_mb: 128
    
  rpi0:
    # Minimal for Pi Zero 2
    workers: 1
    cache_size_mb: 32
```

These defaults are applied at installation time and can be overridden by the user.

### platformRequirements

Declare minimum resource requirements:

```yaml
platformRequirements:
  x86:
    min_ram_mb: 512
    min_disk_mb: 2048
    min_cpu_cores: 1
    
  rpi:
    min_ram_mb: 256
    min_disk_mb: 1024
    min_cpu_cores: 1
    
  rpi0:
    min_ram_mb: 32
    min_disk_mb: 256
    min_cpu_cores: 0  # Single-core equivalent
```

The agent will check these at installation time and warn the user if constraints are not met.

---

## Agent Behavior

### Installation

When a user attempts to install a module on a platform:

```
Platform: rpi0
Module: postgres
platforms: [x86, rpi, rpi0: "never"]

Result: ERROR - Module postgres is not supported on rpi0
```

### Configuration

The agent applies platform-specific defaults:

```go
// Pseudocode
if module.platformDefaults[platform] != nil {
    config = merge(config, module.platformDefaults[platform])
}
```

### Health Checks

Modules can adjust health check logic based on platform:

```go
import "github.com/crateos/crateos/internal/platform"

if platform.IsResourceConstrained() {
    // On RPi Zero 2: check memory usage closely
    healthCheck.memoryThreshold = 70  // 70% of 512MB
} else if platform.IsRaspberryPi() {
    // On RPi 4/5: standard threshold
    healthCheck.memoryThreshold = 80  // 80% of 4GB
} else {
    // On x86: relaxed
    healthCheck.memoryThreshold = 90  // 90% of 2GB+
}
```

---

## Common Module Categories

### Fully Supported on All Platforms

Most core services work on all platforms:

```yaml
platforms:
  - x86
  - rpi
  - rpi0

examples:
  - crateos (core agent)
  - nginx (lightweight proxy)
  - wireguard (VPN)
  - openssh-server (SSH access)
```

### Limited on RPi0

Heavier services work on x86 and RPi 4/5 but are problematic on Zero 2:

```yaml
platforms:
  - x86
  - rpi
  - rpi0: "never"

examples:
  - postgres (database, needs 256MB+ for swap)
  - docker (container runtime)
  - lightdm (GUI, 500MB+ installation)
```

### x86 Only

Virtualization and advanced tools:

```yaml
platforms:
  - x86

examples:
  - qemu-system-x86 (virtualization)
  - libvirt (VM management)
  - cockpit (web UI)
  - ffmpeg (media processing)
```

---

## Recommended Module Categories

### Tier 1: Universal (All Platforms)

Essential services that must work everywhere:

- **Core CrateOS** (agent, policy, CLI)
- **Networking** (SSH, WireGuard, nginx)
- **Storage** (rsync, basic backup)
- **Monitoring** (lightweight metrics)

Configuration pattern:

```yaml
platforms:
  - x86
  - rpi
  - rpi0

platformDefaults:
  x86: { ... full settings ... }
  rpi: { ... reduced settings ... }
  rpi0: { ... minimal settings ... }
```

### Tier 2: ARM-Capable (x86 + RPi 4/5)

Services that work on powerful ARM but not Pi Zero 2:

- **Databases** (postgres, mysql, mariadb)
- **Containers** (docker, podman)
- **Message Brokers** (redis, mosquitto)
- **Web Servers** (nginx, apache)

Configuration pattern:

```yaml
platforms:
  - x86
  - rpi
  - rpi0: "never"

platformDefaults:
  x86: { ... full settings ... }
  rpi: { ... resource-constrained ... }
```

### Tier 3: x86 Only

Advanced tools requiring significant resources:

- **Virtualization** (qemu, libvirt, virt-manager)
- **Desktop UI** (lightdm, cockpit, gnome)
- **Media Tools** (ffmpeg, imagemagick)
- **Development** (gcc, llvm, python development)

Configuration pattern:

```yaml
platforms:
  - x86

platformDefaults:
  x86: { ... full power ... }
```

---

## Runtime Platform Detection

Modules can detect their platform at runtime:

```go
package mymodule

import "github.com/crateos/crateos/internal/platform"

func Init(ctx context.Context) error {
    // Detect platform
    if platform.IsResourceConstrained() {
        // Pi Zero 2: use minimal configuration
        workers = 1
        bufferSize = 32 * 1024 * 1024  // 32MB
    } else if platform.IsRaspberryPi() {
        // Pi 4/5: moderate configuration
        workers = 2
        bufferSize = 256 * 1024 * 1024  // 256MB
    } else {
        // x86: full power
        workers = runtime.NumCPU()
        bufferSize = 1024 * 1024 * 1024  // 1GB
    }
    
    return startService(ctx)
}
```

### Available Functions

```go
// In internal/platform/platform.go

func IsX86() bool                       // x86-64
func IsRaspberryPi() bool               // RPi 4/5
func IsRaspberryPiZero() bool           // RPi Zero 2 W
func IsARM64() bool                     // Any ARM64 (RPi or RPi0)
func IsResourceConstrained() bool       // Only RPi Zero 2 (<2GB RAM)
```

---

## Testing Modules on All Platforms

### Cross-Platform Testing Strategy

1. **Unit Tests**: Run on all platforms (CI)
2. **Integration Tests**: Run in target environment
3. **Resource Tests**: Verify constraints are reasonable

### Example Test Configuration

```yaml
# tests/platform-matrix.yaml
test_matrix:
  - platform: x86
    resources: { ram: 2GB, disk: 10GB }
    stress: high
    
  - platform: rpi
    resources: { ram: 4GB, disk: 8GB }
    stress: medium
    
  - platform: rpi0
    resources: { ram: 512MB, disk: 4GB }
    stress: low
```

---

## Future Extensions

Reserved for future platform expansion:

```yaml
# Example: Future NVIDIA Jetson support
platforms:
  - x86
  - rpi
  - rpi0: "never"
  - jetson              # Future: NVIDIA Jetson (ARM with GPU)
  - ppc64: "never"      # Future: PowerPC 64-bit
  - arm-generic         # Future: Generic ARMv7/v8

platformDefaults:
  jetson:
    # GPU-accelerated settings
    use_cuda: true
    gpu_memory_mb: 2048
```

---

## Migration Path

If a module needs to drop support for a platform:

### Before (Supports all)
```yaml
platforms:
  - x86
  - rpi
  - rpi0
```

### After (Drop RPi0 support)
```yaml
platforms:
  - x86
  - rpi
  - rpi0: "never"

deprecated:
  - platform: rpi0
    reason: "Memory requirements increased to 1GB in v2.0"
    last_supported_version: "1.5.0"
    migration: "Upgrade to x86 or RPi 4/5 device"
```

The agent will show migration guidance to users on unsupported platforms.

---

## Schema Reference

Complete module.yaml schema with platform support:

```yaml
metadata:
  id: string                          # Module identifier
  version: string                     # Semantic version
  description: string                 # Human description
  
  # Platform support (NEW)
  platforms:
    - string                          # Platform IDs: x86, rpi, rpi0
  
  # Platform-specific configuration (NEW)
  platformDefaults:
    x86:
      key: value                      # Default config for x86
    rpi:
      key: value                      # Default config for RPi 4/5
    rpi0:
      key: value                      # Default config for RPi Zero 2
  
  # Resource requirements (NEW)
  platformRequirements:
    x86:
      min_ram_mb: integer
      min_disk_mb: integer
      min_cpu_cores: integer
    rpi:
      min_ram_mb: integer
      min_disk_mb: integer
      min_cpu_cores: integer
    rpi0:
      min_ram_mb: integer
      min_disk_mb: integer
      min_cpu_cores: integer
  
  # Deprecation info (NEW, optional)
  deprecated:
    - platform: string
      reason: string
      last_supported_version: string
      migration: string
```

---

## Examples

See `docs/PLATFORM_SUPPORT.md` for the module availability matrix and deployment recommendations.
