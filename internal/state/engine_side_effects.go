package state

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/crateos/crateos/internal/platform"
)

// ensureExports creates the symlink farm under /srv/crateos/export/.
func ensureExports() []Action {
	if runtime.GOOS != "linux" {
		return nil
	}

	var actions []Action
	links := map[string]string{
		"etc/NetworkManager": "/etc/NetworkManager",
		"etc/nginx":          "/etc/nginx",
		"etc/nftables.conf":  "/etc/nftables.conf",
		"etc/ssh":            "/etc/ssh",
		"var/log/journal":    "/var/log/journal",
	}

	exportBase := platform.CratePath("export")
	_ = os.MkdirAll(exportBase, 0755)
	for rel, target := range links {
		linkPath := filepath.Join(exportBase, rel)

		if _, err := os.Stat(target); err != nil {
			continue
		}
		if existing, err := os.Readlink(linkPath); err == nil && existing == target {
			continue
		}

		parent := filepath.Dir(linkPath)
		_ = os.MkdirAll(parent, 0755)
		os.Remove(linkPath)

		if err := os.Symlink(target, linkPath); err != nil {
			log.Printf("warning: symlink %s -> %s: %v", linkPath, target, err)
			continue
		}

		actions = append(actions, Action{
			Description: fmt.Sprintf("linked %s -> %s", linkPath, target),
			Component:   "symlink",
			Target:      linkPath,
		})
	}

	return actions
}

func systemctl(action string, unit ...string) error {
	args := []string{action}
	args = append(args, unit...)
	return exec.Command("systemctl", args...).Run()
}

func packageInstalled(pkg string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return exec.Command("dpkg-query", "-W", "-f=${Status}", pkg).Run() == nil
}

func installPackage(pkg string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("package install unsupported on %s", runtime.GOOS)
	}
	return exec.Command("apt-get", "install", "-y", pkg).Run()
}
