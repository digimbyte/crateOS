#!/bin/bash
set -euo pipefail

# CrateOS autoinstall ISO builder
# Requires: xorriso, p7zip-full (or 7z), wget
#
# Usage:
#   ISO_MODE=online  bash images/iso/build.sh   # pull deps from Ubuntu repos
#   ISO_MODE=offline-lite bash images/iso/build.sh  # embed CrateOS debs

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"

VERSION="${VERSION:-0.1.0-dev}"
ISO_MODE="${ISO_MODE:-online}"
UBUNTU_ISO_URL="${UBUNTU_ISO_URL:-https://releases.ubuntu.com/noble/ubuntu-24.04.2-live-server-amd64.iso}"
UBUNTU_ISO_NAME="$(basename "$UBUNTU_ISO_URL")"

echo "==> CrateOS ISO builder (mode=${ISO_MODE})"
echo "    Version: ${VERSION}"
echo "    Base ISO: ${UBUNTU_ISO_NAME}"

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
mkdir -p "${WORK}/source/autoinstall"
cp "${SCRIPT_DIR}/autoinstall/user-data" "${WORK}/source/autoinstall/"
cp "${SCRIPT_DIR}/autoinstall/meta-data" "${WORK}/source/autoinstall/"

# --- Embed .deb packages if offline-lite mode ---
if [ "$ISO_MODE" = "offline-lite" ]; then
    echo "==> Embedding CrateOS .deb packages..."
    mkdir -p "${WORK}/source/crateos-debs"
    cp "${DIST}"/*.deb "${WORK}/source/crateos-debs/" 2>/dev/null || {
        echo "ERROR: no .deb files found in ${DIST}/ — run 'make deb' first"
        exit 1
    }
fi

# --- Rebuild ISO ---
OUTPUT="${DIST}/crateos-${VERSION}.iso"
echo "==> Rebuilding ISO → ${OUTPUT}"

# TODO: Replace this with proper xorriso invocation once tested.
echo "ERROR: ISO repack not yet implemented — this is a stub."
echo "       Install xorriso and finalize the repack command for your Ubuntu base."
exit 1
