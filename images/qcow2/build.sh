#!/bin/bash
set -euo pipefail

# CrateOS qcow2 VM image builder
# Requires: qemu-utils, cloud-image-utils (or cloud-localds)
#
# Strategy: start from Ubuntu cloud image, inject cloud-init to install CrateOS debs.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"
COMMON_DIR="${REPO_ROOT}/images/common"
SEED_DEFAULTS="${COMMON_DIR}/seed-defaults.env"

VERSION="${VERSION:-0.1.0-dev}"
CLOUD_IMG_URL="${CLOUD_IMG_URL:-https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img}"
UBUNTU_CODENAME="noble"  # Ubuntu 24.04 LTS
CLOUD_IMG_NAME="$(basename "$CLOUD_IMG_URL")"

echo "==> CrateOS qcow2 builder"
echo "    Version: ${VERSION}"

for tool in wget qemu-img cloud-localds guestfish awk; do
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

if ! ls "${DIST}"/*.deb >/dev/null 2>&1; then
    echo "ERROR: no .deb files found in ${DIST}/ — run 'make deb' first"
    exit 1
fi

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

SEED_DIR="${WORK:-${DIST}/qcow2-seed}"
rm -rf "${SEED_DIR}"
mkdir -p "${SEED_DIR}"

USER_DATA="${SEED_DIR}/user-data"
META_DATA="${SEED_DIR}/meta-data"
REQUIRED_PACKAGES="$(bash "${COMMON_DIR}/render-required-packages.sh" "  ")"
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
}' "${SCRIPT_DIR}/user-data.template" > "${USER_DATA}"

cat > "${META_DATA}" <<EOF
instance-id: crateos-${VERSION}
local-hostname: ${HOSTNAME}
EOF

echo "==> Building seed.iso..."
cloud-localds "${DIST}/seed-${VERSION}.iso" "${USER_DATA}" "${META_DATA}"

echo "==> Embedding debs into qcow2 via guestfish..."
guestfish --rw -a "${OUTPUT}" -i mkdir-p /var/tmp/crateos-debs
for deb in "${DIST}"/*.deb; do
  guestfish --rw -a "${OUTPUT}" -i upload "${deb}" "/var/tmp/crateos-debs/$(basename "${deb}")"
done

echo "==> qcow2 image ready at ${OUTPUT}"
echo "==> cloud-init seed ready at ${DIST}/seed-${VERSION}.iso"
exit 0
