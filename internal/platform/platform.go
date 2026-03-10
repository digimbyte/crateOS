package platform

import "path/filepath"

const (
	Version   = "0.1.0-dev"
	BuildTarget = "x86"  // Injected at compile time via -ldflags
	AppName   = "CrateOS"
	CrateRoot = "/srv/crateos"
	AgentSocket = "/srv/crateos/runtime/agent.sock"
	DefaultConfigRoot = "/usr/share/crateos/defaults/config"
)

// RequiredDirs lists subdirectories that must exist under CrateRoot.
var RequiredDirs = []string{
	"config",
	"modules",
	"services",
	"state",
	"state/last-good",
	"state/rendered",
	"state/backups",
	"logs",
	"export",
	"registry",
	"runtime",
	"cache",
	"backups",
	"bin",
}

// IsX86 returns true if built for x86-64 architecture.
func IsX86() bool {
	return BuildTarget == "x86"
}

// IsRaspberryPi returns true if built for Raspberry Pi 4/5.
func IsRaspberryPi() bool {
	return BuildTarget == "rpi"
}

// IsRaspberryPiZero returns true if built for Raspberry Pi Zero 2 W.
func IsRaspberryPiZero() bool {
	return BuildTarget == "rpi0"
}

// IsARM64 returns true if built for ARM64 architecture (RPi or RPi0).
func IsARM64() bool {
	return BuildTarget == "rpi" || BuildTarget == "rpi0"
}

// IsResourceConstrained returns true for platforms with <2GB RAM (RPi0).
func IsResourceConstrained() bool {
	return BuildTarget == "rpi0"
}

// CratePath joins path segments under CrateRoot.
func CratePath(parts ...string) string {
	args := append([]string{CrateRoot}, parts...)
	return filepath.Join(args...)
}
