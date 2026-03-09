package platform

import "path/filepath"

const (
	Version   = "0.1.0-dev"
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

// CratePath joins path segments under CrateRoot.
func CratePath(parts ...string) string {
	args := append([]string{CrateRoot}, parts...)
	return filepath.Join(args...)
}
