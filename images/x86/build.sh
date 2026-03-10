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
UBUNTU_ISO_URL="${UBUNTU_ISO_URL:-https://releases.ubuntu.com/noble/ubuntu-24.04.2-live-server-amd64.iso}"
UBUNTU_ISO_NAME="$(basename "$UBUNTU_ISO_URL")"
echo "==> CrateOS ISO builder (forced CrateOS install media)"
echo "    Version: ${VERSION}"
echo "    Base ISO: ${UBUNTU_ISO_NAME}"

for tool in wget 7z xorriso sed grep awk; do
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
awk \
    -v packages="${REQUIRED_PACKAGES}" \
    -v hostname="${HOSTNAME}" \
    -v default_user="${DEFAULT_USER}" \
    -v default_password="${DEFAULT_PASSWORD}" \
    -v password_hash="${PASSWORD_HASH}" \
    '
    $0 == "__REQUIRED_PACKAGES__" { print packages; next }
    { gsub(/__HOSTNAME__/, hostname) }
    { gsub(/__DEFAULT_USER__/, default_user) }
    { gsub(/__DEFAULT_PASSWORD__/, default_password) }
    { gsub(/__PASSWORD_HASH__/, password_hash) }
    { print }
}' "${SCRIPT_DIR}/autoinstall/user-data.template" > "${RENDERED_USER_DATA}"
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
    sed -i 's/\\(linux\\s\\+\\S*\\)/\\1 autoinstall ds=nocloud\\;s=\\/cdrom\\/nocloud\\//g' "${WORK}/source/boot/grub/grub.cfg"
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
