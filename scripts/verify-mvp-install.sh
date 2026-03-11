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

require_file "/usr/local/bin/crateos-shell-wrapper" "crateos shell wrapper exists"
require_file "/etc/systemd/system/getty@tty1.service.d/override.conf" "tty1 override exists"
if grep -q -- '--autologin' /etc/systemd/system/getty@tty1.service.d/override.conf; then
  pass "tty1 autologin override is configured for CrateOS takeover"
else
  fail "tty1 autologin override is missing CrateOS takeover settings"
fi
require_file "/etc/os-release" "os-release exists"
if grep -q '^NAME="CrateOS"$' /etc/os-release; then
  pass "installed system identity is branded as CrateOS"
else
  fail "installed system identity is not branded as CrateOS in /etc/os-release"
fi
if grep -q '^PRETTY_NAME="CrateOS (Ubuntu noble derivative)"$' /etc/os-release; then
  pass "installed system pretty name exposes CrateOS as an Ubuntu-derived framework"
else
  fail "installed system pretty name is not branded as the CrateOS Ubuntu-derived framework"
fi
if grep -q '^ID_LIKE=ubuntu debian$' /etc/os-release; then
  pass "installed system identifies as an Ubuntu-derived framework"
else
  fail "installed system does not expose Ubuntu-derived identity in /etc/os-release"
fi
require_file "/etc/issue" "local login banner exists"
if grep -q 'CrateOS - Ubuntu-derived framework appliance' /etc/issue; then
  pass "local login banner presents CrateOS as the Ubuntu-derived framework appliance"
else
  fail "local login banner does not present the CrateOS framework appliance identity"
fi
require_file "/etc/issue.net" "remote login banner exists"
if grep -q 'CrateOS - Ubuntu-derived framework appliance' /etc/issue.net; then
  pass "remote login banner presents CrateOS as the Ubuntu-derived framework appliance"
else
  fail "remote login banner does not present the CrateOS framework appliance identity"
fi
require_file "/etc/default/motd-news" "motd-news config exists"
if grep -q '^ENABLED=0$' /etc/default/motd-news; then
  pass "motd-news is disabled"
else
  fail "motd-news is not disabled"
fi
for disabled_motd in \
  /etc/update-motd.d/50-landscape-sysinfo \
  /etc/update-motd.d/50-motd-news \
  /etc/update-motd.d/80-livepatch \
  /etc/update-motd.d/88-esm-announce \
  /etc/update-motd.d/91-contract-ua-esm-status \
  /etc/update-motd.d/91-release-upgrade \
  /etc/update-motd.d/92-unattended-upgrades; do
  if [[ -e "$disabled_motd" ]]; then
    if [[ -x "$disabled_motd" ]]; then
      fail "stock Ubuntu MOTD surface still executable: $disabled_motd"
    else
      pass "stock Ubuntu MOTD surface disabled: $disabled_motd"
    fi
  fi
done

installer_user="$(awk '
  /^[[:space:]]*-[[:space:]]+name:[[:space:]]*/ {
    line=$0
    sub(/^[[:space:]]*-[[:space:]]+name:[[:space:]]*/, "", line)
    gsub(/"/, "", line)
    gsub(/[[:space:]]+#.*$/, "", line)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
    if (line != "") {
      print line
      exit
    }
  }
' /srv/crateos/config/users.yaml)"

if [[ -n "$installer_user" ]]; then
  if getent passwd "$installer_user" | grep -q ':/usr/local/bin/crateos-shell-wrapper$'; then
    pass "initial CrateOS operator shell is forced through crateos-shell-wrapper"
  else
    fail "initial CrateOS operator shell is not forced through crateos-shell-wrapper"
  fi
else
  fail "could not determine initial CrateOS operator from users.yaml"
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
