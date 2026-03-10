package virtualization

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

// SessionType represents the type of virtual desktop session.
type SessionType string

const (
	SessionTypeVNC    SessionType = "vnc"
	SessionTypeRDP    SessionType = "rdp"
	SessionTypeX11    SessionType = "x11"
	SessionTypeWayland SessionType = "wayland"
)

// UserSession represents an active user desktop session.
type UserSession struct {
	SessionID     string    `json:"session_id"`
	Username      string    `json:"username"`
	Type          SessionType `json:"type"`
	Landing       string    `json:"landing"` // console, panel, workspace, recovery
	Status        string    `json:"status"` // running, stopped, crashed
	PID           int       `json:"pid,omitempty"`
	Display       string    `json:"display,omitempty"` // :0, :1, etc.
	Port          int       `json:"port,omitempty"` // VNC port
	StartedAt     string    `json:"started_at"`
	LastActivityAt string  `json:"last_activity_at"`
	Error         string    `json:"error,omitempty"`
}

// SessionManager manages user desktop sessions.
type SessionManager struct {
	sessions map[string]*UserSession
	stateDir string
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*UserSession),
		stateDir: platform.CratePath("state", "virtualization"),
	}
}

// StartSession starts a new user session.
func (sm *SessionManager) StartSession(username, sessionType, landing string) (*UserSession, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("virtual desktop sessions not supported on non-Linux")
	}

	if err := os.MkdirAll(sm.stateDir, 0755); err != nil {
		return nil, err
	}

	session := &UserSession{
		SessionID:     generateSessionID(username),
		Username:      username,
		Type:          SessionType(sessionType),
		Landing:       landing,
		Status:        "initializing",
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		LastActivityAt: time.Now().UTC().Format(time.RFC3339),
	}

	switch SessionType(sessionType) {
	case SessionTypeVNC:
		if err := sm.startVNCSession(session); err != nil {
			session.Status = "crashed"
			session.Error = err.Error()
			return session, err
		}
	case SessionTypeX11:
		if err := sm.startX11Session(session); err != nil {
			session.Status = "crashed"
			session.Error = err.Error()
			return session, err
		}
	case SessionTypeWayland:
		if err := sm.startWaylandSession(session); err != nil {
			session.Status = "crashed"
			session.Error = err.Error()
			return session, err
		}
	default:
		return nil, fmt.Errorf("unsupported session type: %s", sessionType)
	}

	session.Status = "running"
	sm.sessions[session.SessionID] = session

	// Persist session
	if err := sm.persistSession(session); err != nil {
		return session, fmt.Errorf("failed to persist session: %w", err)
	}

	return session, nil
}

// startVNCSession initializes a VNC session.
func (sm *SessionManager) startVNCSession(session *UserSession) error {
	// Find available VNC port (starting from 5900)
	port := 5900
	for i := 0; i < 100; i++ {
		port = 5900 + i
		if !isPortInUse(port) {
			break
		}
	}

	session.Port = port
	session.Display = fmt.Sprintf(":%d", port-5900)

	// Start Xvfb (X virtual framebuffer)
	cmd := exec.Command("Xvfb", session.Display, "-screen", "0", "1920x1080x24")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xvfb: %w", err)
	}

	session.PID = cmd.Process.Pid

	// Wait a moment for Xvfb to stabilize
	time.Sleep(500 * time.Millisecond)

	// Start VNC server
	vncCmd := exec.Command("vncserver", session.Display, "-geometry", "1920x1080", "-depth", "24")
	vncCmd.Env = append(os.Environ(),
		fmt.Sprintf("DISPLAY=%s", session.Display),
		fmt.Sprintf("USER=%s", session.Username),
	)

	if err := vncCmd.Start(); err != nil {
		return fmt.Errorf("failed to start VNC server: %w", err)
	}

	// Start window manager/desktop in background
	wmCmd := exec.Command("startxfce4")
	wmCmd.Env = append(os.Environ(), fmt.Sprintf("DISPLAY=%s", session.Display))
	if err := wmCmd.Start(); err != nil {
		// Non-fatal; proceed without WM
		fmt.Fprintf(os.Stderr, "warning: failed to start window manager: %v\n", err)
	}

	return nil
}

// startX11Session initializes an X11 session.
func (sm *SessionManager) startX11Session(session *UserSession) error {
	// Find available display
	display := ":10"
	for i := 10; i < 100; i++ {
		display = fmt.Sprintf(":%d", i)
		if !displayExists(display) {
			break
		}
	}

	session.Display = display

	// Start Xvfb
	cmd := exec.Command("Xvfb", display, "-screen", "0", "1920x1080x24")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xvfb: %w", err)
	}

	session.PID = cmd.Process.Pid
	time.Sleep(500 * time.Millisecond)

	// Start window manager
	wmCmd := exec.Command("startxfce4")
	wmCmd.Env = append(os.Environ(), fmt.Sprintf("DISPLAY=%s", display))
	if err := wmCmd.Start(); err != nil {
		return fmt.Errorf("failed to start window manager: %w", err)
	}

	return nil
}

// startWaylandSession initializes a Wayland session.
func (sm *SessionManager) startWaylandSession(session *UserSession) error {
	// Start Weston compositor (Wayland reference implementation)
	cmd := exec.Command("weston", "--backend=headless-backend.so", "--width=1920", "--height=1080")
	cmd.Env = append(os.Environ(), fmt.Sprintf("USER=%s", session.Username))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Weston: %w", err)
	}

	session.PID = cmd.Process.Pid
	return nil
}

// StopSession stops a running session.
func (sm *SessionManager) StopSession(sessionID string) error {
	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.PID > 0 {
		proc, err := os.FindProcess(session.PID)
		if err == nil {
			_ = proc.Kill()
		}
	}

	session.Status = "stopped"
	session.LastActivityAt = time.Now().UTC().Format(time.RFC3339)

	// Update persisted state
	_ = sm.persistSession(session)

	delete(sm.sessions, sessionID)
	return nil
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*UserSession, error) {
	if session, ok := sm.sessions[sessionID]; ok {
		return session, nil
	}

	// Try to load from disk
	path := filepath.Join(sm.stateDir, sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	var session UserSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// ListUserSessions returns all sessions for a user.
func (sm *SessionManager) ListUserSessions(username string) []*UserSession {
	var sessions []*UserSession
	for _, s := range sm.sessions {
		if s.Username == username {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// ListAllSessions returns all active sessions.
func (sm *SessionManager) ListAllSessions() []*UserSession {
	sessions := make([]*UserSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// UpdateActivity updates the last activity timestamp.
func (sm *SessionManager) UpdateActivity(sessionID string) error {
	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastActivityAt = time.Now().UTC().Format(time.RFC3339)
	return sm.persistSession(session)
}

// persistSession writes session state to disk.
func (sm *SessionManager) persistSession(session *UserSession) error {
	path := filepath.Join(sm.stateDir, session.SessionID+".json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// LoadSessions loads all persisted sessions from disk.
func (sm *SessionManager) LoadSessions() error {
	entries, err := os.ReadDir(sm.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No sessions yet
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(sm.stateDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var session UserSession
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			sm.sessions[session.SessionID] = &session
		}
	}

	return nil
}

// Helper functions

func generateSessionID(username string) string {
	return fmt.Sprintf("%s-%d", username, time.Now().UnixNano())
}

func isPortInUse(port int) bool {
	cmd := exec.Command("netstat", "-tuln")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), fmt.Sprintf(":%d", port))
}

func displayExists(display string) bool {
	cmd := exec.Command("test", "-e", fmt.Sprintf("/tmp/.X11-unix/X%s", strings.TrimPrefix(display, ":")))
	return cmd.Run() == nil
}
