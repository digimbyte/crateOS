#!/usr/bin/env bash
set -euo pipefail

pass() { printf '[PASS] %s\n' "$1"; }
fail() { printf '[FAIL] %s\n' "$1"; exit 1; }
warn() { printf '[WARN] %s\n' "$1"; }

MAX_PLATFORM_STATE_AGE_SECONDS=1200
MAX_WATCHDOG_STATE_AGE_SECONDS=180
MAX_READINESS_REPORT_AGE_SECONDS=180

require_file() {
  local p="$1"
  local label="$2"
  [[ -f "$p" ]] && pass "$label" || fail "$label (missing: $p)"
}

require_dir() {
  local p="$1"
  local label="$2"
  [[ -d "$p" ]] && pass "$label" || fail "$label (missing: $p)"
}

require_systemd_active() {
  local unit="$1"
  if systemctl is-active --quiet "$unit"; then
    pass "unit active: $unit"
  else
    fail "unit not active: $unit"
  fi
}

require_systemd_enabled() {
  local unit="$1"
  if systemctl is-enabled --quiet "$unit"; then
    pass "unit enabled: $unit"
  else
    fail "unit not enabled: $unit"
  fi
}

require_recent_file() {
  local p="$1"
  local max_age="$2"
  local label="$3"
  local now
  local mtime
  now="$(date -u +%s)"
  mtime="$(stat -c %Y "$p" 2>/dev/null || true)"
  [[ -n "$mtime" ]] || fail "$label (mtime unavailable: $p)"
  if (( now - mtime <= max_age )); then
    pass "$label"
  else
    fail "$label (stale: $((now - mtime))s > ${max_age}s)"
  fi
}

echo "==> CrateOS MVP install verification"

if [[ "$(uname -s)" != "Linux" ]]; then
  fail "this verification script is Linux-only"
fi

require_systemd_active "crateos-agent.service"
require_systemd_enabled "crateos-agent.service"
require_systemd_active "crateos-agent-watchdog.timer"
require_systemd_enabled "crateos-agent-watchdog.timer"
require_systemd_active "crateos-policy.timer"
require_systemd_enabled "crateos-policy.timer"

require_dir "/srv/crateos" "crate root exists"
require_dir "/srv/crateos/config" "crate config directory exists"
require_file "/srv/crateos/state/installed.json" "installed marker exists"
require_file "/srv/crateos/state/platform-state.json" "platform state exists"
require_file "/srv/crateos/state/agent-watchdog.json" "agent watchdog state exists"
require_file "/srv/crateos/state/readiness-report.json" "readiness report exists"
require_file "/srv/crateos/state/storage-state.json" "storage state exists"
require_recent_file "/srv/crateos/state/platform-state.json" "$MAX_PLATFORM_STATE_AGE_SECONDS" "platform state is fresh"
require_recent_file "/srv/crateos/state/agent-watchdog.json" "$MAX_WATCHDOG_STATE_AGE_SECONDS" "agent watchdog state is fresh"
require_recent_file "/srv/crateos/state/readiness-report.json" "$MAX_READINESS_REPORT_AGE_SECONDS" "readiness report is fresh"
require_recent_file "/srv/crateos/state/storage-state.json" "$MAX_PLATFORM_STATE_AGE_SECONDS" "storage state is fresh"

if [[ -S "/srv/crateos/runtime/agent.sock" ]]; then
  pass "agent socket exists"
else
  fail "agent socket missing or not a socket: /srv/crateos/runtime/agent.sock"
fi

if grep -q '"status":[[:space:]]*"ready"' /srv/crateos/state/readiness-report.json; then
  pass "readiness report status is ready"
else
  fail "readiness report is not ready"
fi

for cfg in crateos.yaml network.yaml firewall.yaml services.yaml users.yaml reverse-proxy.yaml; do
  require_file "/srv/crateos/config/${cfg}" "seeded config exists: ${cfg}"
done

require_file "/etc/ssh/sshd_config.d/10-crateos.conf" "ssh force-command config exists"
if grep -q '^ForceCommand /usr/local/bin/crateos console$' /etc/ssh/sshd_config.d/10-crateos.conf; then
  pass "ssh force-command configured for crateos console"
else
  fail "ssh force-command line missing or mismatched in /etc/ssh/sshd_config.d/10-crateos.conf"
fi

if grep -Eq '^[[:space:]]*-[[:space:]]+name:[[:space:]]*"?[^"#]+' /srv/crateos/config/users.yaml; then
  pass "at least one configured operator exists"
else
  fail "users config does not declare any operators"
fi

if grep -Eq '^[[:space:]]*role:[[:space:]]*"?(admin)\b' /srv/crateos/config/users.yaml; then
  pass "admin operator role is present"
else
  fail "users config does not declare an admin operator"
fi

if command -v dos2unix >/dev/null 2>&1; then
  pass "dos2unix present"
else
  fail "dos2unix missing"
fi

require_file "/srv/crateos/logs/agent-watchdog.log" "agent watchdog log exists"

warn "mvp verification completed successfully"
