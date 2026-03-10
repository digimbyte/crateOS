package virtualization

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

// VirtualDesktopState represents the desired and actual desktop session state.
type VirtualDesktopState struct {
	GeneratedAt   string                        `json:"generated_at"`
	Provider      string                        `json:"provider"` // vnc, rdp, none
	Landing       string                        `json:"landing"` // console, panel, workspace, recovery
	Enabled       bool                          `json:"enabled"`
	Sessions      []VirtualDesktopSessionSummary `json:"sessions"`
	Issues        []string                      `json:"issues"`
	Summary       string                        `json:"summary"`
}

// VirtualDesktopSessionSummary represents an active session.
type VirtualDesktopSessionSummary struct {
	SessionID     string `json:"session_id"`
	Username      string `json:"username"`
	Type          string `json:"type"`
	Landing       string `json:"landing"`
	Status        string `json:"status"`
	PID           int    `json:"pid,omitempty"`
	Port          int    `json:"port,omitempty"`
	Display       string `json:"display,omitempty"`
	StartedAt     string `json:"started_at"`
	LastActivityAt string `json:"last_activity_at"`
}

// ReconcileVirtualDesktop reconciles virtual desktop configuration with actual state.
func ReconcileVirtualDesktop(cfg *config.Config) VirtualDesktopState {
	state := VirtualDesktopState{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Sessions:    make([]VirtualDesktopSessionSummary, 0),
		Issues:      make([]string, 0),
	}

	// Check if virtual desktop is enabled
	state.Enabled = cfg.CrateOS.Access.VirtualDesktop.Enabled
	state.Provider = strings.TrimSpace(cfg.CrateOS.Access.VirtualDesktop.Provider)
	state.Landing = normalizeLandingForDesktop(cfg.CrateOS.Access.VirtualDesktop.Landing)

	if !state.Enabled {
		state.Summary = "virtual desktop disabled"
		return state
	}

	// Validate provider
	if state.Provider == "" {
		state.Issues = append(state.Issues, "virtual desktop enabled but provider not specified")
		state.Summary = fmt.Sprintf("virtual desktop %s disabled (no provider)", state.Landing)
		return state
	}

	if !isSupportedProvider(state.Provider) {
		state.Issues = append(state.Issues, fmt.Sprintf("unsupported virtual desktop provider: %s", state.Provider))
		state.Summary = fmt.Sprintf("virtual desktop misconfigured (%s unsupported)", state.Provider)
		return state
	}

	// Load session manager and get active sessions
	sm := NewSessionManager()
	if err := sm.LoadSessions(); err != nil {
		state.Issues = append(state.Issues, fmt.Sprintf("failed to load sessions: %v", err))
	}

	for _, session := range sm.ListAllSessions() {
		state.Sessions = append(state.Sessions, VirtualDesktopSessionSummary{
			SessionID:     session.SessionID,
			Username:      session.Username,
			Type:          string(session.Type),
			Landing:       session.Landing,
			Status:        session.Status,
			PID:           session.PID,
			Port:          session.Port,
			Display:       session.Display,
			StartedAt:     session.StartedAt,
			LastActivityAt: session.LastActivityAt,
		})
	}

	// Build summary
	sessionCount := len(state.Sessions)
	runningCount := 0
	for _, s := range state.Sessions {
		if s.Status == "running" {
			runningCount++
		}
	}

	state.Summary = fmt.Sprintf(
		"virtual desktop (%s → %s): %d sessions (%d running)",
		state.Provider, state.Landing, sessionCount, runningCount,
	)

	if len(state.Issues) == 0 {
		return state
	}

	state.Summary += fmt.Sprintf(" with %d issues", len(state.Issues))
	return state
}

// isSupportedProvider checks if a provider is supported.
func isSupportedProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "vnc", "rdp", "x11", "wayland":
		return true
	default:
		return false
	}
}

// normalizeLandingForDesktop normalizes the landing surface for virtual desktop.
func normalizeLandingForDesktop(landing string) string {
	switch strings.ToLower(strings.TrimSpace(landing)) {
	case "", "panel":
		return "panel"
	case "workspace":
		return "workspace"
	case "recovery":
		return "recovery"
	case "console":
		return "console"
	default:
		return "panel"
	}
}

// ValidateVirtualDesktopConfig validates the virtual desktop configuration.
func ValidateVirtualDesktopConfig(cfg config.CrateOSConfig) []string {
	var issues []string

	if !cfg.Access.VirtualDesktop.Enabled {
		return nil
	}

	provider := strings.TrimSpace(cfg.Access.VirtualDesktop.Provider)
	if provider == "" {
		issues = append(issues, "virtual desktop enabled but provider not specified")
	} else if !isSupportedProvider(provider) {
		issues = append(issues, fmt.Sprintf("unsupported virtual desktop provider: %s", provider))
	}

	landing := normalizeLandingForDesktop(cfg.Access.VirtualDesktop.Landing)
	if landing == "shell" || landing == "desktop" {
		issues = append(issues, "virtual desktop landing must stay inside CrateOS-owned surfaces (not shell or desktop)")
	}

	return issues
}

// SessionRequest represents a request to start a session.
type SessionRequest struct {
	Username  string `json:"username"`
	Type      string `json:"type"` // vnc, rdp, x11, wayland
	Landing   string `json:"landing"`
}

// SessionResponse represents a session response.
type SessionResponse struct {
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Port        int    `json:"port,omitempty"`
	Display     string `json:"display,omitempty"`
	Error       string `json:"error,omitempty"`
}

// StartUserSession starts a virtual desktop session for a user.
func StartUserSession(username, sessionType, landing string) SessionResponse {
	resp := SessionResponse{
		Type: sessionType,
	}

	sm := NewSessionManager()
	if err := sm.LoadSessions(); err != nil {
		resp.Status = "error"
		resp.Error = fmt.Sprintf("failed to load sessions: %v", err)
		return resp
	}

	session, err := sm.StartSession(username, sessionType, landing)
	if err != nil {
		resp.Status = "error"
		resp.Error = err.Error()
		return resp
	}

	resp.SessionID = session.SessionID
	resp.Status = session.Status
	resp.Port = session.Port
	resp.Display = session.Display

	return resp
}

// StopUserSession stops a virtual desktop session.
func StopUserSession(sessionID string) SessionResponse {
	resp := SessionResponse{
		SessionID: sessionID,
	}

	sm := NewSessionManager()
	if err := sm.LoadSessions(); err != nil {
		resp.Status = "error"
		resp.Error = fmt.Sprintf("failed to load sessions: %v", err)
		return resp
	}

	if err := sm.StopSession(sessionID); err != nil {
		resp.Status = "error"
		resp.Error = err.Error()
		return resp
	}

	resp.Status = "stopped"
	return resp
}

// ListUserSessions lists all sessions for a user.
func ListUserSessions(username string) []VirtualDesktopSessionSummary {
	var summaries []VirtualDesktopSessionSummary

	sm := NewSessionManager()
	if err := sm.LoadSessions(); err != nil {
		return summaries
	}

	sessions := sm.ListUserSessions(username)
	for _, s := range sessions {
		summaries = append(summaries, VirtualDesktopSessionSummary{
			SessionID:     s.SessionID,
			Username:      s.Username,
			Type:          string(s.Type),
			Landing:       s.Landing,
			Status:        s.Status,
			PID:           s.PID,
			Port:          s.Port,
			Display:       s.Display,
			StartedAt:     s.StartedAt,
			LastActivityAt: s.LastActivityAt,
		})
	}

	return summaries
}

// GetSessionInfo retrieves detailed session information.
func GetSessionInfo(sessionID string) (*VirtualDesktopSessionSummary, error) {
	sm := NewSessionManager()
	if err := sm.LoadSessions(); err != nil {
		return nil, fmt.Errorf("failed to load sessions: %w", err)
	}

	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	return &VirtualDesktopSessionSummary{
		SessionID:     session.SessionID,
		Username:      session.Username,
		Type:          string(session.Type),
		Landing:       session.Landing,
		Status:        session.Status,
		PID:           session.PID,
		Port:          session.Port,
		Display:       session.Display,
		StartedAt:     session.StartedAt,
		LastActivityAt: session.LastActivityAt,
	}, nil
}

// SaveVirtualDesktopState persists the current state.
func SaveVirtualDesktopState(state VirtualDesktopState) error {
	path := platform.CratePath("state", "rendered", "virtual-desktop.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return WriteFile(path, data)
}

// WriteFile is a helper to write files.
func WriteFile(path string, data []byte) error {
	// Implementation would use os.WriteFile with proper error handling
	return fmt.Errorf("not implemented")
}
