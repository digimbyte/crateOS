package takeover

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ContractCheck struct {
	Label   string
	OK      bool
	Details string
}

func shellMatches(username string) bool {
	passwdData, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(passwdData), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 7 && strings.TrimSpace(fields[0]) == username {
			return strings.TrimSpace(fields[6]) == LoginShellPath
		}
	}
	return false
}

const (
	LoginShellPath = "/usr/local/bin/crateos-login-shell"
	loginShellBody = "#!/bin/bash\nCURRENT_USER=\"${USER:-$(id -un 2>/dev/null || printf '%s' 'crate')}\"\nCURRENT_HOME=\"${HOME:-$(getent passwd \"${CURRENT_USER}\" 2>/dev/null | cut -d: -f6)}\"\n\nexport TERM=\"${TERM:-linux}\"\nexport USER=\"${CURRENT_USER}\"\nexport LOGNAME=\"${LOGNAME:-${CURRENT_USER}}\"\nexport HOME=\"${CURRENT_HOME:-/home/${CURRENT_USER}}\"\n\nexec /usr/local/bin/crateos console\n"
	ttyOverridePath = "/etc/systemd/system/getty@tty1.service.d/override.conf"
	sshDropInPath   = "/etc/ssh/sshd_config.d/10-crateos.conf"
	sshDropInBody   = "# CrateOS: force SSH sessions into the CrateOS console.\n# Users land in the TUI instead of a raw shell.\n# Admin break-glass access is handled via the console's escape hatch.\nForceCommand /usr/local/bin/crateos console\n"
	issuePath       = "/etc/issue"
	issueBody       = "CrateOS - Ubuntu-derived framework appliance \\n \\l\n"
	issueNetPath    = "/etc/issue.net"
	issueNetBody    = "CrateOS - Ubuntu-derived framework appliance\n"
	osReleasePath   = "/etc/os-release"
	osReleaseBody   = "NAME=\"CrateOS\"\nPRETTY_NAME=\"CrateOS (Ubuntu noble derivative)\"\nID=crateos\nID_LIKE=ubuntu debian\nVERSION=\"0.1.0+noble1\"\nVERSION_ID=\"0.1.0\"\nVERSION_CODENAME=noble\nHOME_URL=\"https://crateos.local\"\nSUPPORT_URL=\"https://crateos.local/support\"\nBUG_REPORT_URL=\"https://crateos.local/issues\"\n"
	lsbReleasePath  = "/etc/lsb-release"
	lsbReleaseBody  = "DISTRIB_ID=CrateOS\nDISTRIB_RELEASE=0.1.0\nDISTRIB_CODENAME=noble\nDISTRIB_DESCRIPTION=\"CrateOS (Ubuntu noble derivative)\"\n"
)

func EnsureLoginShell() error {
	if err := os.MkdirAll(filepath.Dir(LoginShellPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(LoginShellPath, []byte(loginShellBody), 0755); err != nil {
		return err
	}
	return os.Chmod(LoginShellPath, 0755)
}

func EnsureUserShell(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	if shellMatches(username) {
		return nil
	}
	if err := exec.Command("usermod", "-s", LoginShellPath, username).Run(); err == nil {
		if shellMatches(username) {
			return nil
		}
	}
	if _, err := exec.LookPath("chsh"); err == nil {
		if err := exec.Command("chsh", "-s", LoginShellPath, username).Run(); err == nil {
			if shellMatches(username) {
				return nil
			}
		}
	}
	return fmt.Errorf("failed to enforce %s as login shell for %s", LoginShellPath, username)
}

func TTYOverridePath() string {
	return ttyOverridePath
}

func RenderTTYOverride(_ string) string {
	return fmt.Sprintf("[Service]\nExecStart=\nExecStart=-/sbin/agetty --noclear %%I $TERM\nType=idle\n")
}

func EnsureTTYOverride(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ttyOverridePath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(ttyOverridePath, []byte(RenderTTYOverride(username)), 0644); err != nil {
		return err
	}
	return os.Chmod(ttyOverridePath, 0644)
}

func EnsureSSHDropIn() error {
	if err := os.MkdirAll(filepath.Dir(sshDropInPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(sshDropInPath, []byte(sshDropInBody), 0644); err != nil {
		return err
	}
	return os.Chmod(sshDropInPath, 0644)
}

func EnsureIdentityFiles() error {
	managed := []struct {
		path string
		body string
		mode os.FileMode
	}{
		{path: issuePath, body: issueBody, mode: 0644},
		{path: issueNetPath, body: issueNetBody, mode: 0644},
		{path: osReleasePath, body: osReleaseBody, mode: 0644},
		{path: lsbReleasePath, body: lsbReleaseBody, mode: 0644},
	}
	for _, file := range managed {
		if err := os.MkdirAll(filepath.Dir(file.path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(file.path, []byte(file.body), file.mode); err != nil {
			return err
		}
		if err := os.Chmod(file.path, file.mode); err != nil {
			return err
		}
	}
	return nil
}
func RepairLocalConsoleContract(username string) error {
	if err := EnsureLoginShell(); err != nil {
		return err
	}
	username = strings.TrimSpace(username)
	if username != "" {
		if err := EnsureUserShell(username); err != nil {
			return err
		}
		if err := EnsureTTYOverride(username); err != nil {
			return err
		}
		ReloadSystemdBestEffort()
	}
	return nil
}

func RepairLocalInstallContract(username string) error {
	if err := RepairLocalConsoleContract(username); err != nil {
		return err
	}
	if err := EnsureSSHDropIn(); err != nil {
		return err
	}
	if err := EnsureIdentityFiles(); err != nil {
		return err
	}
	return nil
}

func SSHDropInPath() string {
	return sshDropInPath
}

func OSReleasePath() string {
	return osReleasePath
}

func EvaluateLocalInstallContract(username string) []ContractCheck {
	checks := make([]ContractCheck, 0, 5)
	if _, err := os.Stat(LoginShellPath); err == nil {
		checks = append(checks, ContractCheck{Label: "Console takeover shell", OK: true, Details: LoginShellPath})
	} else {
		checks = append(checks, ContractCheck{Label: "Console takeover shell", OK: false, Details: "missing " + LoginShellPath})
	}
	if _, err := os.Stat(TTYOverridePath()); err == nil {
		checks = append(checks, ContractCheck{Label: "TTY1 takeover override", OK: true, Details: TTYOverridePath()})
	} else {
		checks = append(checks, ContractCheck{Label: "TTY1 takeover override", OK: false, Details: "missing " + TTYOverridePath()})
	}
	if _, err := os.Stat(SSHDropInPath()); err == nil {
		checks = append(checks, ContractCheck{Label: "SSH console landing", OK: true, Details: SSHDropInPath()})
	} else {
		checks = append(checks, ContractCheck{Label: "SSH console landing", OK: false, Details: "missing " + SSHDropInPath()})
	}
	if _, err := os.Stat(OSReleasePath()); err == nil {
		checks = append(checks, ContractCheck{Label: "System identity files", OK: true, Details: OSReleasePath()})
	} else {
		checks = append(checks, ContractCheck{Label: "System identity files", OK: false, Details: "missing " + OSReleasePath()})
	}
	username = strings.TrimSpace(username)
	if username != "" {
		provisioned := shellMatches(username)
		if provisioned {
			checks = append(checks, ContractCheck{Label: "Provisioned operator account", OK: true, Details: username + " uses crateos-login-shell"})
		} else {
			checks = append(checks, ContractCheck{Label: "Provisioned operator account", OK: false, Details: "system account missing or shell mismatch for " + username})
		}
	}
	return checks
}

func ReloadSystemdBestEffort() {
	_ = exec.Command("systemctl", "daemon-reload").Run()
}
