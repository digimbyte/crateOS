package logs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/crateos/crateos/internal/platform"
)

// ExportSpec defines a curated log file to generate from journald.
type ExportSpec struct {
	Name string   // output filename (e.g. "boot.log")
	Args []string // journalctl arguments
}

// DefaultSpecs returns the standard set of curated log exports.
func DefaultSpecs() []ExportSpec {
	return []ExportSpec{
		{
			Name: "boot.log",
			Args: []string{"-b", "--no-pager", "-o", "short-iso"},
		},
		{
			Name: "net.log",
			Args: []string{"-u", "NetworkManager", "-b", "--no-pager", "-o", "short-iso"},
		},
		{
			Name: "services.log",
			Args: []string{
				"-u", "crateos-agent",
				"-u", "crateos-policy",
				"-u", "nginx",
				"-u", "docker",
				"-b", "--no-pager", "-o", "short-iso",
			},
		},
		{
			Name: "fw.log",
			Args: []string{"-u", "nftables", "-b", "--no-pager", "-o", "short-iso"},
		},
	}
}

// ExportAll generates all curated log files into /srv/crateos/logs/.
// This is a no-op on non-Linux systems.
func ExportAll() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	logDir := platform.CratePath("logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	for _, spec := range DefaultSpecs() {
		if err := exportOne(logDir, spec); err != nil {
			log.Printf("log export %s: %v", spec.Name, err)
			// Continue exporting other logs even if one fails.
		}
	}
	return nil
}

func exportOne(dir string, spec ExportSpec) error {
	outPath := fmt.Sprintf("%s/%s", dir, spec.Name)

	out, err := exec.Command("journalctl", spec.Args...).Output()
	if err != nil {
		return fmt.Errorf("journalctl: %w", err)
	}

	return os.WriteFile(outPath, out, 0644)
}
