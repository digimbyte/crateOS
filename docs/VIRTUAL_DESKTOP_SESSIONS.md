# CrateOS Virtual Desktop Session Management

## Overview

This document describes the virtual desktop session management system for CrateOS, enabling users to access desktop environments via VNC, RDP, X11, or Wayland with controlled landing surfaces and comprehensive session tracking.

## Architecture

### Components

```
crateos.yaml (Config)
    ↓
config.Load()
    ↓
virtualization.ReconcileVirtualDesktop()
    ├─ virtualization/sessions.go: Session lifecycle management
    ├─ virtualization/reconcile.go: State reconciliation
    └─ internal/api/virtualization.go: HTTP API endpoints
    ↓
state/virtualization/*.json (Session state)
state/rendered/virtual-desktop.json (Rendered state)
```

### Session Types Supported

- **VNC** (Virtual Network Computing)
  - Via Xvfb (X virtual framebuffer) + TightVNC/TigerVNC
  - Accessible on dynamically allocated port (5900+)
  - Supports 1920x1080x24 resolution
  
- **RDP** (Remote Desktop Protocol)
  - Future implementation via xrdp
  - Windows Remote Desktop compatible
  
- **X11**
  - Direct X11 display server
  - XFCE4 window manager
  - Supports display forwarding
  
- **Wayland**
  - Modern display protocol via Weston compositor
  - Better security model than X11

## Configuration

### Enable Virtual Desktop

In `crateos.yaml`:

```yaml
access:
  virtual_desktop:
    enabled: true
    provider: "vnc"           # vnc, rdp, x11, wayland
    landing: "workspace"      # console, panel, workspace, recovery
```

### Landing Surfaces

- **console** - CrateOS console (TUI)
- **panel** - Desktop panel/taskbar environment
- **workspace** - Full desktop workspace
- **recovery** - Recovery/diagnostic environment

## Session Lifecycle

### Starting a Session

```bash
curl -X POST http://unix/virtualization/sessions/start \
  -H "X-CrateOS-User: alice" \
  -d '{
    "type": "vnc",
    "landing": "workspace"
  }'
```

Response:
```json
{
  "session_id": "alice-1678390472000000000",
  "status": "running",
  "type": "vnc",
  "port": 5900,
  "display": ":0"
}
```

Session Details:
- `session_id`: Unique session identifier
- `status`: initializing → running → stopped/crashed
- `port`: VNC port (if applicable)
- `display`: X11 display number

### Listing Sessions

**Current user's sessions:**
```bash
curl http://unix/virtualization/sessions/list \
  -H "X-CrateOS-User: alice"
```

**Admin viewing all sessions:**
```bash
curl "http://unix/virtualization/sessions/list?user=*" \
  -H "X-CrateOS-User: admin"
```

Response:
```json
{
  "user": "alice",
  "sessions": [
    {
      "session_id": "alice-1678390472000000000",
      "username": "alice",
      "type": "vnc",
      "landing": "workspace",
      "status": "running",
      "pid": 12345,
      "port": 5900,
      "display": ":0",
      "started_at": "2026-03-09T20:23:48Z",
      "last_activity_at": "2026-03-09T20:24:30Z"
    }
  ],
  "count": 1
}
```

### Getting Session Info

```bash
curl "http://unix/virtualization/sessions/info?session_id=alice-1678390472000000000" \
  -H "X-CrateOS-User: alice"
```

### Stopping a Session

```bash
curl -X POST http://unix/virtualization/sessions/stop \
  -H "X-CrateOS-User: alice" \
  -d '{
    "session_id": "alice-1678390472000000000"
  }'
```

Response:
```json
{
  "session_id": "alice-1678390472000000000",
  "status": "stopped"
}
```

## Session State Management

### Persisted Session Data

Sessions are persisted to:
```
/srv/crateos/state/virtualization/<session_id>.json
```

Example:
```json
{
  "session_id": "alice-1678390472000000000",
  "username": "alice",
  "type": "vnc",
  "landing": "workspace",
  "status": "running",
  "pid": 12345,
  "display": ":0",
  "port": 5900,
  "started_at": "2026-03-09T20:23:48Z",
  "last_activity_at": "2026-03-09T20:24:30Z"
}
```

### Platform State Integration

Rendered state:
```
/srv/crateos/state/rendered/virtual-desktop.json
```

Includes:
- Overall provider and landing configuration
- All active sessions with status
- Session counts (running, stopped, crashed)
- Validation issues

## Permissions

### Session Permissions

- `virtualization.manage` - Full virtualization control (admin)
- `sys.manage` - System-wide admin (implicit virtualization.manage)

### Session Access Rules

| Action | Self | Other User | Admin |
|--------|------|-----------|-------|
| Start session | ✓ | ✗ | ✓ |
| Stop own session | ✓ | ✗ | ✓ |
| List own sessions | ✓ | ✗ | ✓ |
| View own session | ✓ | ✗ | ✓ |
| View all sessions | ✗ | ✗ | ✓ |
| View system status | ✗ | ✗ | ✓ |

## VNC Access

### Connecting to a VNC Session

Once a session is started on port 5900:

```bash
# Using vncviewer
vncviewer localhost:5900

# Using ssh tunnel
ssh -L 5900:localhost:5900 user@crateos-host
vncviewer localhost:5900
```

### VNC Features

- 1920×1080 resolution
- 24-bit color depth
- XFCE4 desktop environment
- Automatic display allocation
- Port auto-discovery (5900, 5901, 5902, etc.)

## Session Activity Tracking

### Updating Activity

The `last_activity_at` timestamp is updated:
- When session is created
- When user interacts with session (mouse/keyboard)
- When session is explicitly updated

### Activity API

```bash
curl -X POST http://unix/virtualization/sessions/activity \
  -H "X-CrateOS-User: alice" \
  -d '{"session_id": "alice-1678390472000000000"}'
```

## Display Management

### Available Displays

CrateOS manages X11 displays:
- `:0` - Primary display (if local GUI enabled)
- `:10-:99` - VNC/virtual sessions

### Display Detection

The system automatically:
1. Checks for existing displays in `/tmp/.X11-unix/`
2. Allocates next available display number
3. Starts Xvfb with allocated display
4. Starts window manager on that display

## Environment Variables in Sessions

Sessions automatically set:
```bash
DISPLAY=:N        # Assigned display number
USER=username     # Session username
HOME=/home/username
LOGNAME=username
UID=NNNN
GID=NNNN
```

## Window Manager Integration

### XFCE4 Desktop Environment

Each VNC session automatically launches XFCE4:
- Desktop
- Panel with taskbar
- Application menu
- File manager
- Terminal

### Custom Landing Surfaces

Landing configuration allows pinning to:
- **workspace** - Full desktop with applications
- **panel** - Just taskbar/panel (minimal)
- **console** - Return to CrateOS console
- **recovery** - Diagnostic/recovery mode

## Security Considerations

### Session Isolation

- Each session runs with user's UID/GID
- Home directory isolation
- Display server bound to localhost
- VNC access requires SSH tunnel or local network

### Process Management

- Sessions tracked by PID
- Killing process terminates session
- Session state persisted for recovery
- Orphaned sessions auto-cleanup

### Network Security

- VNC ports restricted to localhost (no remote access)
- Access requires SSH tunnel to host
- Forward through SSH provides encryption
- Port allocation prevents conflicts

## Troubleshooting

### Session Fails to Start

Check:
1. Xvfb installed? `which Xvfb`
2. Window manager available? `which startxfce4`
3. Display server resources? `ps aux | grep Xvfb`
4. Port conflicts? `netstat -tuln | grep 59`

### VNC Connection Refused

Check:
1. Session status: `/virtualization/sessions/info`
2. VNC server running: `ps aux | grep vnc`
3. SSH tunnel active: `ssh -L 5900:localhost:5900`
4. Firewall rules allow 5900

### Session Stuck/Unresponsive

Actions:
1. Stop session via API
2. Check for orphaned processes: `ps aux | grep [username]`
3. Manual cleanup: `kill -9 [PID]`
4. Review state file in `/srv/crateos/state/virtualization/`

## Performance Considerations

### Resource Usage per Session

Typical VNC session:
- Memory: 200-500 MB
- CPU: 5-15% idle
- Display: Xvfb consumes minimal resources
- Network: ~2-5 Mbps for typical interaction

### Session Limits

Recommended limits:
- Max sessions per user: 5
- Max total sessions: 20
- Session idle timeout: 2 hours
- Display cleanup: Auto-remove stopped sessions

## Future Enhancements

- [ ] RDP protocol support via xrdp
- [ ] Session idle timeout/auto-cleanup
- [ ] Per-user session limits
- [ ] Session recording for audit
- [ ] Clipboard sharing over VNC
- [ ] Multi-display support
- [ ] GPU acceleration for 3D
- [ ] WebRTC/HTML5 VNC client
- [ ] Session suspension/resumption
- [ ] Desktop environment customization per user
