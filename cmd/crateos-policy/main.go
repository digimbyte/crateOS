package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"github.com/crateos/crateos/internal/config"

	"github.com/crateos/crateos/internal/platform"
)

const (
	maxPlatformStateAge = 20 * time.Minute
	maxWatchdogStateAge = 3 * time.Minute
	maxReadinessReportAge = 3 * time.Minute
)

const (
	crateOSHeaderScript = "#!/bin/sh\nprintf 'CrateOS\\n'\nprintf 'Ubuntu-derived framework appliance\\n'\n"
	crateOSHelpScript   = "#!/bin/sh\nprintf 'Primary interface: CrateOS control surface\\n'\nprintf 'Base platform: Ubuntu noble derivative\\n'\nprintf 'Admin shell access is break-glass only.\\n'\n"
	crateOSSSHDropIn    = "# CrateOS: force SSH sessions into the CrateOS console.\n# Users land in the TUI instead of a raw shell.\n# Admin break-glass access is handled via the console's escape hatch.\nForceCommand /usr/local/bin/crateos console\n"
	crateOSIssue        = "CrateOS - Ubuntu-derived framework appliance \\n \\l\n"
	crateOSIssueNet     = "CrateOS - Ubuntu-derived framework appliance\n"
	crateOSRelease      = "NAME=\"CrateOS\"\nPRETTY_NAME=\"CrateOS (Ubuntu noble derivative)\"\nID=crateos\nID_LIKE=ubuntu debian\nVERSION=\"0.1.0+noble1\"\nVERSION_ID=\"0.1.0\"\nVERSION_CODENAME=noble\nHOME_URL=\"https://crateos.local\"\nSUPPORT_URL=\"https://crateos.local/support\"\nBUG_REPORT_URL=\"https://crateos.local/issues\"\n"
	crateOSLSBRelease   = "DISTRIB_ID=CrateOS\nDISTRIB_RELEASE=0.1.0\nDISTRIB_CODENAME=noble\nDISTRIB_DESCRIPTION=\"CrateOS (Ubuntu noble derivative)\"\n"
	crateOSGrubBranding = "GRUB_DISTRIBUTOR=\"CrateOS\"\n"
)

var stockMOTDScripts = []string{
	"/etc/update-motd.d/50-landscape-sysinfo",
	"/etc/update-motd.d/50-motd-news",
	"/etc/update-motd.d/80-livepatch",
	"/etc/update-motd.d/88-esm-announce",
	"/etc/update-motd.d/91-contract-ua-esm-status",
	"/etc/update-motd.d/91-release-upgrade",
	"/etc/update-motd.d/92-unattended-upgrades",
	"/etc/update-motd.d/95-hwe-eol",
	"/etc/update-motd.d/97-overlayroot",
	"/etc/update-motd.d/98-fsck-at-reboot",
	"/etc/update-motd.d/98-reboot-required",
}

type readinessReport struct {
	CheckedAt string   `json:"checked_at"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Failures  []string `json:"failures,omitempty"`
}

// checks is the list of policy assertions to verify.
var checks = []struct {
	name string
	fn   func() error
}{
	{"crate root exists", checkCrateRoot},
	{"installed marker present", checkInstalledMarker},
	{"required dirs present", checkRequiredDirs},
	{"agent socket present", checkAgentSocket},
	{"platform state present", checkPlatformState},
	{"watchdog state present", checkWatchdogState},
}

func main() {
	log.SetPrefix("crateos-policy: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	log.Printf("policy check v%s", platform.Version)
	if err := reconcileManagedState(); err != nil {
		log.Printf("repair warning: %v", err)
	}

	failed := 0
	failures := []string{}
	for _, c := range checks {
		if err := c.fn(); err != nil {
			log.Printf("FAIL  %s: %v", c.name, err)
			failed++
			failures = append(failures, fmt.Sprintf("%s: %v", c.name, err))
		} else {
			log.Printf("OK    %s", c.name)
		}
	}
	if err := writeReadinessReport(failures); err != nil {
		log.Printf("FAIL  readiness report write: %v", err)
		failed++
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("policy check: %d/%d checks failed\n", failed, len(checks))
		os.Exit(1)
	}
	fmt.Printf("policy check: all %d checks passed\n", len(checks))
}

func reconcileManagedState() error {
	var errs []string

	cfg, err := config.Load()
	if err != nil {
		errs = append(errs, fmt.Sprintf("load config: %v", err))
	}

	repairs := []func(*config.Config) error{
		ensureCrateDirectories,
		ensureSSHDropIn,
		ensureShellWrapperPermissions,
		ensureMOTDState,
		ensureIdentityFiles,
		ensureGrubBranding,
		ensureTTYOverride,
	}

	for _, repair := range repairs {
		if err := repair(cfg); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func ensureCrateDirectories(_ *config.Config) error {
	if err := os.MkdirAll("/etc/ssh/sshd_config.d", 0755); err != nil {
		return fmt.Errorf("ensure ssh drop-in dir: %w", err)
	}
	if err := os.MkdirAll("/etc/default/grub.d", 0755); err != nil {
		return fmt.Errorf("ensure grub branding dir: %w", err)
	}
	if err := os.MkdirAll("/etc/update-motd.d", 0755); err != nil {
		return fmt.Errorf("ensure motd dir: %w", err)
	}
	return nil
}

func ensureSSHDropIn(_ *config.Config) error {
	if err := writeManagedFile("/etc/ssh/sshd_config.d/10-crateos.conf", crateOSSSHDropIn, 0644); err != nil {
		return fmt.Errorf("ensure ssh drop-in: %w", err)
	}
	if err := restartServiceIfActive("sshd", "ssh"); err != nil {
		return fmt.Errorf("restart ssh service: %w", err)
	}
	return nil
}

func ensureShellWrapperPermissions(_ *config.Config) error {
	if err := os.Chmod("/usr/local/bin/crateos-shell-wrapper", 0755); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("chmod shell wrapper: %w", err)
	}
	return nil
}

func ensureMOTDState(_ *config.Config) error {
	if err := writeManagedFile("/etc/update-motd.d/00-crateos-header", crateOSHeaderScript, 0755); err != nil {
		return fmt.Errorf("ensure CrateOS MOTD header: %w", err)
	}
	if err := writeManagedFile("/etc/update-motd.d/10-help-text", crateOSHelpScript, 0755); err != nil {
		return fmt.Errorf("ensure CrateOS MOTD help text: %w", err)
	}
	if err := writeManagedFile("/etc/default/motd-news", "ENABLED=0\n", 0644); err != nil {
		return fmt.Errorf("disable motd-news: %w", err)
	}
	for _, path := range stockMOTDScripts {
		if err := chmodIfExists(path, 0644); err != nil {
			return fmt.Errorf("disable stock motd script %s: %w", path, err)
		}
	}
	return nil
}

func ensureIdentityFiles(_ *config.Config) error {
	managed := []struct {
		path string
		body string
		mode os.FileMode
	}{
		{path: "/etc/issue", body: crateOSIssue, mode: 0644},
		{path: "/etc/issue.net", body: crateOSIssueNet, mode: 0644},
		{path: "/etc/os-release", body: crateOSRelease, mode: 0644},
		{path: "/etc/lsb-release", body: crateOSLSBRelease, mode: 0644},
	}
	for _, file := range managed {
		if err := writeManagedFile(file.path, file.body, file.mode); err != nil {
			return fmt.Errorf("ensure identity file %s: %w", file.path, err)
		}
	}
	return nil
}

func ensureGrubBranding(_ *config.Config) error {
	path := "/etc/default/grub.d/10-crateos-branding.cfg"
	before, _ := os.ReadFile(path)
	if err := writeManagedFile(path, crateOSGrubBranding, 0644); err != nil {
		return fmt.Errorf("ensure grub branding: %w", err)
	}
	if string(before) != crateOSGrubBranding {
		if err := runBestEffort("update-grub"); err != nil {
			if err := runBestEffort("grub-mkconfig", "-o", "/boot/grub/grub.cfg"); err != nil {
				return fmt.Errorf("refresh grub config: %w", err)
			}
		}
	}
	return nil
}

func ensureTTYOverride(cfg *config.Config) error {
	if cfg == nil || len(cfg.Users.Users) == 0 {
		return nil
	}

	username := strings.TrimSpace(cfg.Users.Users[0].Name)
	if username == "" {
		return nil
	}

	override := fmt.Sprintf("[Service]\nExecStart=\nExecStart=-/sbin/agetty --autologin %s --noclear %%I $TERM\nType=idle\n", username)
	path := "/etc/systemd/system/getty@tty1.service.d/override.conf"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("ensure tty1 override dir: %w", err)
	}
	before, _ := os.ReadFile(path)
	if err := writeManagedFile(path, override, 0644); err != nil {
		return fmt.Errorf("ensure tty1 override: %w", err)
	}
	if string(before) != override {
		if err := runSystemctlBestEffort("daemon-reload"); err != nil {
			return fmt.Errorf("reload systemd for tty1 override: %w", err)
		}
	}
	return nil
}

func writeManagedFile(path, content string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return err
	}
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return nil
}

func chmodIfExists(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Chmod(path, mode)
}

func restartServiceIfActive(names ...string) error {
	if !systemdIsLive() {
		return nil
	}
	for _, name := range names {
		if err := exec.Command("systemctl", "is-active", "--quiet", name).Run(); err == nil {
			return exec.Command("systemctl", "restart", name).Run()
		}
	}
	return nil
}

func runBestEffort(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	return exec.Command(name, args...).Run()
}

func runSystemctlBestEffort(args ...string) error {
	if !systemdIsLive() {
		return nil
	}
	return runBestEffort("systemctl", args...)
}

func systemdIsLive() bool {
	info, err := os.Stat("/run/systemd/system")
	return err == nil && info.IsDir()
}

func checkCrateRoot() error {
	info, err := os.Stat(platform.CrateRoot)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", platform.CrateRoot)
	}
	return nil
}

func checkAgentSocket() error {
	info, err := os.Stat(platform.AgentSocket)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s is not a socket", platform.AgentSocket)
	}
	return nil
}

func checkPlatformState() error {
	p := platform.CratePath("state", "platform-state.json")
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if time.Since(info.ModTime()) > maxPlatformStateAge {
		return fmt.Errorf("stale: older than %s", maxPlatformStateAge)
	}
	return nil
}

func checkWatchdogState() error {
	p := platform.CratePath("state", "agent-watchdog.json")
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	if time.Since(info.ModTime()) > maxWatchdogStateAge {
		return fmt.Errorf("stale: older than %s", maxWatchdogStateAge)
	}
	return nil
}

func checkInstalledMarker() error {
	p := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	return nil
}

func checkRequiredDirs() error {
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("%s missing: %w", d, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", d)
		}
	}
	return nil
}

func writeReadinessReport(failures []string) error {
	report := readinessReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "ready",
		Summary:   fmt.Sprintf("control plane ready (policy cadence <= %s, report freshness <= %s)", 2*time.Minute, maxReadinessReportAge),
	}
	if len(failures) > 0 {
		report.Status = "degraded"
		report.Summary = failures[0]
		report.Failures = failures
	}
	path := platform.CratePath("state", "readiness-report.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
