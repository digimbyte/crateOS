#!/bin/bash
set -euo pipefail

# CrateOS Raspberry Pi 4/5 image builder
# Requires: xz-utils, parted, kpartx, losetup, mkfs.ext4
#
# Usage:
#   bash images/rpi/build.sh  # Creates arm64 Raspberry Pi OS image with CrateOS

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DIST="${REPO_ROOT}/dist"
COMMON_DIR="${REPO_ROOT}/images/common"
SEED_DEFAULTS="${COMMON_DIR}/seed-defaults-rpi.env"

VERSION="${VERSION:-0.1.0+rpi1}"
RPI_OS_URL="${RPI_OS_URL:-https://downloads.raspberrypi.com/raspios_arm64/images/raspios_arm64-2024-03-15/2024-03-15-raspios-bookworm-arm64.img.xz}"
RPI_OS_NAME="$(basename "$RPI_OS_URL")"
RPI_OS_IMG="${RPI_OS_NAME%.xz}"

echo "==> CrateOS Raspberry Pi 4/5 image builder"
echo "    Version: ${VERSION}"
echo "    Base image: ${RPI_OS_IMG}"

for tool in xz parted kpartx losetup mkfs.ext4 rsync sed grep awk; do
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

# --- Download base image if not cached ---
mkdir -p "${DIST}/cache"
if [ ! -f "${DIST}/cache/${RPI_OS_IMG}" ]; then
    if [ ! -f "${DIST}/cache/${RPI_OS_NAME}" ]; then
        echo "==> Downloading Raspberry Pi OS image..."
        wget -q --show-progress -O "${DIST}/cache/${RPI_OS_NAME}" "${RPI_OS_URL}"
    fi
    echo "==> Extracting Raspberry Pi OS image..."
    xz -d "${DIST}/cache/${RPI_OS_NAME}"
fi

# --- Mount and customize image ---
WORK="${DIST}/rpi-work"
rm -rf "${WORK}"
mkdir -p "${WORK}"

IMG="${DIST}/cache/${RPI_OS_IMG}"
MOUNT_POINT="${WORK}/mnt"
mkdir -p "${MOUNT_POINT}"

echo "==> Mounting Raspberry Pi OS image..."
LOOP_DEV=$(losetup --find --partscan --show "${IMG}")
sleep 1

# Determine partition offsets (typically root is partition 2)
PARTITIONS=$(lsblk -no PATH,TYPE "${LOOP_DEV}" | grep part | awk '{print $1}')
ROOT_PART=$(echo "${PARTITIONS}" | tail -1)

if [ -z "${ROOT_PART}" ]; then
    echo "ERROR: could not find root partition in ${IMG}"
    losetup -d "${LOOP_DEV}"
    exit 1
fi

mount "${ROOT_PART}" "${MOUNT_POINT}"
trap "umount -R '${MOUNT_POINT}' 2>/dev/null; losetup -d '${LOOP_DEV}' 2>/dev/null" EXIT

# --- Copy CrateOS debs to image ---
echo "==> Installing CrateOS .deb packages..."
mkdir -p "${MOUNT_POINT}/tmp/crateos-debs"
cp "${DIST}"/*.deb "${MOUNT_POINT}/tmp/crateos-debs/" 2>/dev/null || {
    echo "ERROR: no .deb files found in ${DIST}/ — run 'make deb-rpi' first"
    exit 1
}

# --- Inject hostname and user config ---
if [ -f "${MOUNT_POINT}/etc/hostname" ]; then
    echo "${HOSTNAME}" | tee "${MOUNT_POINT}/etc/hostname" > /dev/null
fi

# Create/update default user on first boot
mkdir -p "${MOUNT_POINT}/root"
cat > "${MOUNT_POINT}/root/create-user.sh" <<'EOF'
#!/bin/bash
USER="${DEFAULT_USER}"
if ! id "${USER}" &>/dev/null; then
    adduser --disabled-password --gecos "" "${USER}" || true
fi
echo "${USER}:${DEFAULT_PASSWORD}" | chpasswd
for group in sudo netdev gpio i2c spi dialout; do
    usermod -aG "${group}" "${USER}" 2>/dev/null || true
done
EOF

chmod 755 "${MOUNT_POINT}/root/create-user.sh"

# --- Create systemd service to install CrateOS on first boot ---
mkdir -p "${MOUNT_POINT}/etc/systemd/system"
cat > "${MOUNT_POINT}/etc/systemd/system/crateos-install.service" <<'EOF'
[Unit]
Description=CrateOS First-Boot Installation
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/root/install-crateos.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

cat > "${MOUNT_POINT}/root/install-crateos.sh" <<'EOF'
#!/bin/bash
set -euo pipefail

echo "==> Installing CrateOS packages..."
cd /tmp/crateos-debs
dpkg -i *.deb || apt-get install -f -y

echo "==> Setting up default user..."
/root/create-user.sh

echo "==> Enforcing CrateOS local console takeover..."
test -x /usr/local/bin/crateos-login-shell
usermod -s /usr/local/bin/crateos-login-shell "${DEFAULT_USER}"
getent passwd "${DEFAULT_USER}" | grep ':/usr/local/bin/crateos-login-shell$'
mkdir -p /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=-/sbin/agetty --noissue --autologin ${DEFAULT_USER} %I \$TERM
Type=idle
EOF
chmod 0644 /etc/systemd/system/getty@tty1.service.d/override.conf
grep -q -- '--autologin ${DEFAULT_USER}' /etc/systemd/system/getty@tty1.service.d/override.conf
chmod 0755 /usr/local/bin/crateos-login-shell
systemctl daemon-reload || true
systemctl restart getty@tty1.service || true
/usr/local/bin/verify-bootstrap-artifacts

echo "==> Disabling first-boot service..."
systemctl disable crateos-install.service || true
rm -f /etc/systemd/system/crateos-install.service

echo "==> CrateOS installation complete"
EOF

chmod 755 "${MOUNT_POINT}/root/install-crateos.sh"

# Enable the service on first boot
ln -sf /etc/systemd/system/crateos-install.service \
    "${MOUNT_POINT}/etc/systemd/system/multi-user.target.wants/crateos-install.service" 2>/dev/null || true

# --- Finalize image ---
echo "==> Finalizing Raspberry Pi image..."
sync

# --- Output image ---
OUTPUT="${DIST}/crateos-rpi-${VERSION}.img"
cp "${IMG}" "${OUTPUT}"

echo "==> RPi image ready at ${OUTPUT}"
echo "    Flash to microSD: xzcat ${OUTPUT}.xz | dd of=/dev/sdX bs=4M && sync"
echo "    Or: etcher --flash-to-device /dev/sdX --bz2 ${OUTPUT}"
