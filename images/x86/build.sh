#!/bin/bash
set -euo pipefail

# CrateOS autoinstall ISO builder
# Requires: xorriso, p7zip-full (or 7z), wget
#
# Usage:
#   bash images/iso/build.sh  # embeds CrateOS debs and builds forced-install CrateOS media

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"
COMMON_DIR="${REPO_ROOT}/images/common"
SEED_DEFAULTS="${COMMON_DIR}/seed-defaults.env"

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

# shellcheck disable=SC1090
source "${SEED_DEFAULTS}"

# --- Download base ISO if not cached ---
mkdir -p "${DIST}/cache"
if [ ! -f "${DIST}/cache/${UBUNTU_ISO_NAME}" ]; then
    echo "==> Downloading base Ubuntu ISO..."
    wget -q --show-progress -O "${DIST}/cache/${UBUNTU_ISO_NAME}" "${UBUNTU_ISO_URL}"
fi

# --- Extract ISO ---
WORK="${DIST}/iso-work"
rm -rf "${WORK}"
mkdir -p "${WORK}/source"

echo "==> Extracting base ISO..."
7z x -o"${WORK}/source" "${DIST}/cache/${UBUNTU_ISO_NAME}" > /dev/null

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
content = content.replace("__REQUIRED_PACKAGES__", os.environ["REQUIRED_PACKAGES"])
content = content.replace("__HOSTNAME__", os.environ["HOSTNAME"])
content = content.replace("__DEFAULT_USER__", os.environ["DEFAULT_USER"])
content = content.replace("__DEFAULT_PASSWORD__", os.environ["DEFAULT_PASSWORD"])
content = content.replace("__PASSWORD_HASH__", os.environ["PASSWORD_HASH"])
output_path.write_text(content, encoding="utf-8")
PY
mkdir -p "${WORK}/source/nocloud"
cp "${RENDERED_USER_DATA}" "${WORK}/source/nocloud/user-data"
cp "${SCRIPT_DIR}/autoinstall/meta-data" "${WORK}/source/nocloud/meta-data"

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

# Ensure kernel cmdline enables autoinstall
if grep -q "autoinstall" "${WORK}/source/boot/grub/grub.cfg"; then
    :
else
    sed -i 's|\\(linux\\s\\+\\S*\\)|\\1 autoinstall ds=nocloud\\;s=/cdrom/nocloud/|g' "${WORK}/source/boot/grub/grub.cfg"
fi

# Override operator-facing installer and boot branding across GRUB assets.
WORK_SOURCE="${WORK}/source" python3 <<'PY'
import os
from pathlib import Path

root = Path(os.environ["WORK_SOURCE"])
replacements = [
    ("Try or Install Ubuntu Server", "Install CrateOS"),
    ("Install Ubuntu Server", "Install CrateOS"),
    ("Try Ubuntu Server", "Install CrateOS"),
    ("Ubuntu Server", "CrateOS"),
    ("Try or Install", "Install"),
    ("Ubuntu", "CrateOS"),
]

for path in (root / "boot" / "grub").glob("*.cfg"):
    content = path.read_text(encoding="utf-8", errors="ignore")
    updated = content
    for old, new in replacements:
        updated = updated.replace(old, new)
    if updated != content:
        path.write_text(updated, encoding="utf-8")
PY

if [ -f "${WORK}/source/.disk/info" ]; then
    printf 'CrateOS %s LTS - Release amd64\n' "${VERSION}" > "${WORK}/source/.disk/info"
fi

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
    xorriso -indev "${DIST}/cache/${UBUNTU_ISO_NAME}" \
        -report_el_torito as_mkisofs \
        -report_system_area as_mkisofs 2>/dev/null \
        | awk 'BEGIN { ORS=" " } /^[[:space:]]*-/ { sub(/^[[:space:]]+/, ""); printf "%s", $0 " " }'
)"
if [ -z "${BOOT_OPTS// }" ]; then
    echo "ERROR: failed to derive boot metadata from source ISO"
    exit 1
fi

# --- Rebuild ISO ---
OUTPUT="${DIST}/crateos-${VERSION}.iso"
echo "==> Rebuilding ISO → ${OUTPUT}"
eval "xorriso -as mkisofs \
  -r -V \"CrateOS ${VERSION}\" \
  -o \"${OUTPUT}\" \
  -J -joliet-long -cache-inodes \
  ${BOOT_OPTS} \
  \"${WORK}/source\""

echo "==> ISO ready at ${OUTPUT}"
exit 0
