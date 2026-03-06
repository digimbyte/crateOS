#!/bin/bash
set -euo pipefail

# CrateOS qcow2 VM image builder
# Requires: qemu-utils, cloud-image-utils (or cloud-localds)
#
# Strategy: start from Ubuntu cloud image, inject cloud-init to install CrateOS debs.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"

VERSION="${VERSION:-0.1.0-dev}"
CLOUD_IMG_URL="${CLOUD_IMG_URL:-https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img}"
UBUNTU_CODENAME="noble"  # Ubuntu 24.04 LTS
CLOUD_IMG_NAME="$(basename "$CLOUD_IMG_URL")"

echo "==> CrateOS qcow2 builder"
echo "    Version: ${VERSION}"

# --- Download cloud image if not cached ---
mkdir -p "${DIST}/cache"
if [ ! -f "${DIST}/cache/${CLOUD_IMG_NAME}" ]; then
    echo "==> Downloading Ubuntu cloud image..."
    wget -q --show-progress -O "${DIST}/cache/${CLOUD_IMG_NAME}" "${CLOUD_IMG_URL}"
fi

# --- Create working copy ---
OUTPUT="${DIST}/crateos-${VERSION}.qcow2"
cp "${DIST}/cache/${CLOUD_IMG_NAME}" "${OUTPUT}"

# --- Resize disk (optional, default 20G) ---
echo "==> Resizing image to 20G..."
qemu-img resize "${OUTPUT}" 20G

# --- Generate cloud-init seed ISO ---
echo "==> Generating cloud-init seed..."
# TODO: Create a proper cloud-init user-data that installs CrateOS debs on first boot.
echo "ERROR: cloud-init seed generation not yet implemented — this is a stub."
exit 1
