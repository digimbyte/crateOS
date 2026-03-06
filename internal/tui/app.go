package tui

import (
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crateos/crateos/internal/sysinfo"
)

// ── View state ──────────────────────────────────────────────────────

type viewID int

const (
	viewMenu viewID = iota
	viewStatus
	viewServices
	viewLogs
	viewNetwork
)

// ── Service info ────────────────────────────────────────────────────

// ServiceInfo describes a managed service and its runtime status.
type ServiceInfo struct {
	Name   string
	Status string // "active", "inactive", "failed", "unknown"
	Type   string // "systemd" or "docker"
}

// ── Model ───────────────────────────────────────────────────────────

type model struct {
	currentView viewID
	cursor      int
	width       int
	height      int
	info        sysinfo.Info
	interfaces  []sysinfo.NetIface
	services    []ServiceInfo
	quitting    bool
}

var menuItems = []string{
	"System Status",
	"Services",
	"Logs",
	"Network",
	"Exit",
}

// NewModel creates the initial TUI model with fresh system info.
func NewModel() model {
	return model{
		currentView: viewMenu,
		info:        sysinfo.Gather(),
		interfaces:  sysinfo.NetworkInterfaces(),
		services:    gatherServices(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch m.currentView {
	case viewMenu:
		return m.updateMenu(msg)
	case viewStatus:
		return m.updateGeneric(msg)
	case viewServices:
		return m.updateGeneric(msg)
	case viewLogs:
		return m.updateGeneric(msg)
	case viewNetwork:
		return m.updateGeneric(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.currentView {
	case viewMenu:
		return m.viewMenu()
	case viewStatus:
		return m.viewStatus()
	case viewServices:
		return m.viewServices()
	case viewLogs:
		return m.viewLogs()
	case viewNetwork:
		return m.viewNetwork()
	}
	return ""
}

// updateGeneric handles ESC/backspace to return to the menu from any sub-view.
func (m model) updateGeneric(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "backspace", "q":
			m.currentView = viewMenu
			m.cursor = 0
		}
	}
	return m, nil
}

// Run starts the TUI in alt-screen mode.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ── Helpers ─────────────────────────────────────────────────────────

func gatherServices() []ServiceInfo {
	known := []string{"crateos-agent", "crateos-policy"}
	var svcs []ServiceInfo
	for _, name := range known {
		s := ServiceInfo{Name: name, Type: "systemd", Status: "unknown"}
		if runtime.GOOS == "linux" {
			out, err := exec.Command("systemctl", "is-active", name).Output()
			if err == nil {
				s.Status = strings.TrimSpace(string(out))
			} else {
				s.Status = "inactive"
			}
		}
		svcs = append(svcs, s)
	}
	return svcs
}
