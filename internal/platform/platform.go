package platform

import "path/filepath"

const (
	Version   = "0.1.0-dev"
	AppName   = "CrateOS"
	CrateRoot = "/srv/crateos"
)

// RequiredDirs lists subdirectories that must exist under CrateRoot.
var RequiredDirs = []string{
	"config",
	"modules",
	"services",
	"state",
	"state/last-good",
	"state/backups",
	"logs",
	"export",
	"bin",
}

// CratePath joins path segments under CrateRoot.
func CratePath(parts ...string) string {
	args := append([]string{CrateRoot}, parts...)
	return filepath.Join(args...)
}
