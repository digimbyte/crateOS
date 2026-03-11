package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

func (m model) executeSystemCommand(mod string, params []string) (tea.Model, tea.Cmd) {
	switch mod {
	case "refresh", "reload":
		m.refreshOverview()
		m.refreshUsers()
		m.refreshPrimerState()
		if m.primerRequired {
			m.currentView = viewSetup
			m.setCommandWarn("primer still blocking normal console access")
			return m, nil
		}
		m.setCommandOK("state refreshed")
		return m, nil
	case "dos2unix", "normalize", "lineendings":
		scope := "config"
		if len(params) > 0 {
			scope = strings.ToLower(strings.TrimSpace(params[0]))
		}
		count, err := runDos2UnixHealth(scope)
		if err != nil {
			m.setCommandError("dos2unix failed: " + err.Error())
			return m, nil
		}
		m.setCommandOK(fmt.Sprintf("dos2unix normalized files: %d (scope:%s)", count, scope))
		return m, nil
	case "ftp-complete", "upload-complete":
		target := strings.TrimSpace(strings.Join(params, " "))
		if target == "" {
			m.setCommandWarn("usage: system ftp-complete <path>")
			return m, nil
		}
		m.setCommandWarn("ftp-complete: use agent reconciliation instead")
		return m, nil
	default:
		m.setCommandWarn("usage: system <refresh|dos2unix [config|services|all]|ftp-complete <path>>")
		return m, nil
	}
}

func (m *model) requireLiveControlPlane(action string) bool {
	if m.controlPlaneOnline {
		return true
	}
	if strings.TrimSpace(action) == "" {
		action = "write action"
	}
	m.setCommandWarn(action + " unavailable: control plane offline, console is using fallback state")
	return false
}

func runDos2UnixHealth(scope string) (int, error) {
	if runtime.GOOS != "linux" {
		return 0, fmt.Errorf("dos2unix health check is only supported on linux hosts")
	}
	paths := []string{}
	switch scope {
	case "config":
		paths = []string{platform.CratePath("config")}
	case "services":
		paths = []string{platform.CratePath("services")}
	case "all":
		paths = []string{platform.CratePath("config"), platform.CratePath("services"), platform.CratePath("modules")}
	default:
		return 0, fmt.Errorf("unsupported scope: %s", scope)
	}
	targets := []string{}
	for _, root := range paths {
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			normalized, normalizeErr := config.NormalizeFileIfNeeded(path)
			if normalizeErr != nil {
				return normalizeErr
			}
			if normalized {
				targets = append(targets, path)
			}
			return nil
		}); err != nil {
			return len(targets), err
		}
	}
	return len(targets), nil
}

func (m model) executeNavigationCommand(target string) (tea.Model, tea.Cmd) {
	if m.primerRequired && target != "setup" {
		m.currentView = viewSetup
		m.setCommandWarn("primer completion required before navigation")
		return m, nil
	}
	switch target {
	case "setup":
		m.currentView = viewSetup
		m.setCommandOK("route: setup")
	case "menu", "home":
		m.enterMenuView()
		m.setCommandOK("route: menu")
	case "status":
		m.enterStatusView()
		m.setCommandOK("route: status")
	case "diagnostics", "diag":
		m.enterDiagnosticsView()
		m.setCommandOK("route: diagnostics")
	case "services", "service", "svc":
		m.enterServicesView()
		m.setCommandOK("route: services")
	case "users", "userdir":
		m.enterUsersView()
		m.setCommandOK("route: users")
	case "logs", "log":
		m.enterLogsView()
		m.setCommandOK("route: logs")
	case "network", "net":
		m.enterNetworkView()
		m.setCommandOK("route: network")
	default:
		m.setCommandWarn("usage: nav <setup|menu|status|diagnostics|services|users|logs|network>")
	}
	return m, nil
}
