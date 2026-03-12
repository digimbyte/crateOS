#!/bin/bash
set -euo pipefail

# CrateOS autoinstall ISO builder
# Requires: xorriso, p7zip-full (or 7z), wget
#
# Usage:
#   bash images/iso/build.sh  # generic ISO build lane sharing the CrateOS autoinstall pipeline

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"
COMMON_DIR="${REPO_ROOT}/images/common"
SEED_DEFAULTS="${COMMON_DIR}/seed-defaults.env"
OVERLAY_DIR="${SCRIPT_DIR}/overlay"
FETCH_CACHE="${COMMON_DIR}/fetch-cache.sh"
NORMALIZE_LINUX_PAYLOADS="${COMMON_DIR}/normalize-linux-payloads.py"

VERSION="${VERSION:-0.1.0-dev}"
UBUNTU_RELEASES_INDEX="${UBUNTU_RELEASES_INDEX:-https://releases.ubuntu.com/noble/}"
if [ -n "${UBUNTU_ISO_URL:-}" ]; then
    RESOLVED_UBUNTU_ISO_URL="${UBUNTU_ISO_URL}"
else
    echo "==> Resolving latest Ubuntu Noble live-server ISO..."
    RELEASE_PAGE="$(wget -qO- "${UBUNTU_RELEASES_INDEX}")"
    UBUNTU_ISO_PATH="$(printf '%s\n' "${RELEASE_PAGE}" \
        | grep -oE 'ubuntu-24\.04\.[0-9]+-live-server-amd64\.iso' \
        | sort -V \
        | tail -n 1)"
    if [ -z "${UBUNTU_ISO_PATH}" ]; then
        echo "ERROR: failed to resolve a Ubuntu Noble live-server ISO from ${UBUNTU_RELEASES_INDEX}"
        exit 1
    fi
    RESOLVED_UBUNTU_ISO_URL="${UBUNTU_RELEASES_INDEX}${UBUNTU_ISO_PATH}"
fi
UBUNTU_ISO_URL="${RESOLVED_UBUNTU_ISO_URL}"
UBUNTU_ISO_NAME="$(basename "$UBUNTU_ISO_URL")"
echo "==> CrateOS ISO builder (forced CrateOS install media)"
echo "    Version: ${VERSION}"
echo "    Base ISO: ${UBUNTU_ISO_NAME}"

for tool in wget 7z xorriso sed grep awk python3; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
        echo "ERROR: required tool not found: ${tool}"
        exit 1
    fi
done

if [ ! -f "${SEED_DEFAULTS}" ]; then
    echo "ERROR: seed defaults file not found: ${SEED_DEFAULTS}"
    exit 1
fi
if [ ! -f "${FETCH_CACHE}" ]; then
    echo "ERROR: cache helper not found: ${FETCH_CACHE}"
    exit 1
fi
if [ ! -f "${NORMALIZE_LINUX_PAYLOADS}" ]; then
    echo "ERROR: LF normalization helper not found: ${NORMALIZE_LINUX_PAYLOADS}"
    exit 1
fi
if [ ! -f "${OVERLAY_DIR}/usr/local/bin/crateos-login-shell" ]; then
    echo "ERROR: missing CrateOS login shell overlay: ${OVERLAY_DIR}/usr/local/bin/crateos-login-shell"
    exit 1
fi
if [ ! -f "${OVERLAY_DIR}/etc/systemd/system/getty@tty1.service.d/override.conf.template" ]; then
    echo "ERROR: missing CrateOS tty1 override overlay: ${OVERLAY_DIR}/etc/systemd/system/getty@tty1.service.d/override.conf.template"
    exit 1
fi

echo "==> Normalizing Linux payload line endings..."
python3 "${NORMALIZE_LINUX_PAYLOADS}"
# shellcheck disable=SC1090
source "${SEED_DEFAULTS}"
HOSTNAME="$(printf '%s' "${HOSTNAME}" | tr -d '\r')"
DEFAULT_USER="$(printf '%s' "${DEFAULT_USER}" | tr -d '\r')"
DEFAULT_PASSWORD="$(printf '%s' "${DEFAULT_PASSWORD}" | tr -d '\r')"
PASSWORD_HASH="$(printf '%s' "${PASSWORD_HASH}" | tr -d '\r')"

# --- Download base ISO if not cached ---
BASE_ISO_PATH="$(bash "${FETCH_CACHE}" "iso" "${UBUNTU_ISO_URL}" "base Ubuntu ISO")"

# --- Extract ISO ---
WORK="${DIST}/iso-work"
rm -rf "${WORK}"
mkdir -p "${WORK}/source"

echo "==> Extracting base ISO..."
7z x -o"${WORK}/source" "${BASE_ISO_PATH}" > /dev/null

# --- Inject autoinstall ---
REQUIRED_PACKAGES="$(bash "${COMMON_DIR}/render-required-packages.sh" "    ")"
RENDERED_USER_DATA="${WORK}/rendered-user-data"
REQUIRED_PACKAGES="${REQUIRED_PACKAGES}" \
HOSTNAME="${HOSTNAME}" \
DEFAULT_USER="${DEFAULT_USER}" \
DEFAULT_PASSWORD="${DEFAULT_PASSWORD}" \
PASSWORD_HASH="${PASSWORD_HASH}" \
RENDERED_USER_DATA="${RENDERED_USER_DATA}" \
USER_DATA_TEMPLATE="${SCRIPT_DIR}/autoinstall/user-data.template" \
python3 <<'PY'
import os
from pathlib import Path

template_path = Path(os.environ["USER_DATA_TEMPLATE"])
output_path = Path(os.environ["RENDERED_USER_DATA"])

content = template_path.read_text(encoding="utf-8")
required_packages = os.environ["REQUIRED_PACKAGES"].replace("\r", "")
hostname = os.environ["HOSTNAME"].replace("\r", "")
default_user = os.environ["DEFAULT_USER"].replace("\r", "")
default_password = os.environ["DEFAULT_PASSWORD"].replace("\r", "")
password_hash = os.environ["PASSWORD_HASH"].replace("\r", "")
content = content.replace("__REQUIRED_PACKAGES__", required_packages)
content = content.replace("__HOSTNAME__", hostname)
content = content.replace("__DEFAULT_USER__", default_user)
content = content.replace("__DEFAULT_PASSWORD__", default_password)
content = content.replace("__PASSWORD_HASH__", password_hash)
output_path.write_text(content, encoding="utf-8", newline="\n")
PY
mkdir -p "${WORK}/source/nocloud"
cp "${RENDERED_USER_DATA}" "${WORK}/source/nocloud/user-data"
python3 - "${SCRIPT_DIR}/autoinstall/meta-data" "${WORK}/source/nocloud/meta-data" <<'PY'
from pathlib import Path
import sys

source = Path(sys.argv[1])
target = Path(sys.argv[2])
target.write_text(source.read_text(encoding="utf-8").replace("\r", ""), encoding="utf-8", newline="\n")
PY

# --- Inject installer overlay takeover payload ---
if [ -d "${OVERLAY_DIR}" ]; then
    mkdir -p "${WORK}/source/overlay"
    cp -R "${OVERLAY_DIR}/." "${WORK}/source/overlay/"
    DEFAULT_USER="${DEFAULT_USER}" WORK_SOURCE="${WORK}/source" python3 <<'PY'
import os
from pathlib import Path

root = Path(os.environ["WORK_SOURCE"]) / "overlay"
default_user = os.environ["DEFAULT_USER"]
replacements = {"__DEFAULT_USER__": default_user}

for relative in [
    Path("usr/local/bin/crateos-login-shell"),
    Path("etc/systemd/system/getty@tty1.service.d/override.conf.template"),
]:
    path = root / relative
    content = path.read_text(encoding="utf-8")
    for old, new in replacements.items():
        content = content.replace(old, new)
    if path.name.endswith(".template"):
        path = path.with_suffix("")
        original = root / relative
        original.unlink()
    path.write_text(content, encoding="utf-8")
PY
fi

# --- Embed .deb packages (required) ---
echo "==> Embedding CrateOS .deb packages..."
mkdir -p "${WORK}/source/crateos-debs"
cp "${DIST}"/*.deb "${WORK}/source/crateos-debs/" 2>/dev/null || {
    echo "ERROR: no .deb files found in ${DIST}/ — run 'make deb' first"
    exit 1
}

# --- Validate extracted ISO layout ---
if [ ! -f "${WORK}/source/boot/grub/grub.cfg" ]; then
    echo "ERROR: extracted ISO missing boot/grub/grub.cfg"
    exit 1
fi
if [ ! -d "${WORK}/source/casper" ]; then
    echo "ERROR: extracted ISO missing casper/ payload"
    exit 1
fi

# Preserve upstream Ubuntu installer-facing metadata so hypervisors and installer tooling
# continue to recognize the media as unattended-capable Ubuntu Server, while still forcing
# our own nocloud autoinstall path via kernel cmdline.
WORK_SOURCE="${WORK}/source" python3 <<'PY'
import os
import re
from pathlib import Path

root = Path(os.environ["WORK_SOURCE"])
for path in (root / "boot" / "grub").glob("*.cfg"):
    content = path.read_text(encoding="utf-8", errors="ignore")
    updated = re.sub(
        r'(^[ \t]*linux[^\n]*?)(\s+---(?:\s|$))',
        lambda match: match.group(0)
        if "autoinstall" in match.group(1)
        else f'{match.group(1)} autoinstall ds=nocloud\\;s=/cdrom/nocloud/{match.group(2)}',
        content,
        flags=re.MULTILINE,
    )
    if updated != content:
        path.write_text(updated, encoding="utf-8")
PY

# Refresh ISO checksums after mutating media contents.
if [ -f "${WORK}/source/md5sum.txt" ]; then
    (
        cd "${WORK}/source"
        rm -f md5sum.txt
        find . -type f ! -name md5sum.txt -print0 \
            | xargs -0 md5sum \
            | grep -v -E '(\./boot\.catalog|\.?/isolinux/boot\.cat)$' \
            > md5sum.txt
    )
fi

# Replay the source ISO boot metadata rather than assuming an older isolinux layout.
BOOT_OPTS="$(
    xorriso -indev "${BASE_ISO_PATH}" \
        -report_el_torito as_mkisofs \
        -report_system_area as_mkisofs 2>/dev/null \
        | awk 'BEGIN { ORS=" " } /^[[:space:]]*-/ { sub(/^[[:space:]]+/, ""); printf "%s", $0 " " }'
)"
if [ -z "${BOOT_OPTS// }" ]; then
    echo "ERROR: failed to derive boot metadata from source ISO"
    exit 1
fi

filter_xorriso_progress() {
    awk '
        /xorriso : UPDATE :/ {
            count++
            if (count % 20 == 1) {
                print
                fflush()
            }
            next
        }
        { print; fflush() }
    '
}

# --- Rebuild ISO ---
OUTPUT="${DIST}/crateos-${VERSION}.iso"
echo "==> Rebuilding ISO → ${OUTPUT}"
eval "xorriso -as mkisofs \
  -r -V \"CrateOS ${VERSION}\" \
  -o \"${OUTPUT}\" \
  -J -joliet-long -cache-inodes \
  ${BOOT_OPTS} \
  \"${WORK}/source\"" 2>&1 | filter_xorriso_progress

pipe_status=("${PIPESTATUS[@]}")
if [ "${pipe_status[0]}" -ne 0 ]; then
    exit "${pipe_status[0]}"
fi

echo "==> ISO ready at ${OUTPUT}"
exit 0
