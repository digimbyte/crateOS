#!/bin/bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
    echo "usage: $0 <indent>" >&2
    exit 1
fi

INDENT="$1"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PACKAGES_FILE="${REPO_ROOT}/packaging/config/packages.yaml"

if [ ! -f "${PACKAGES_FILE}" ]; then
    echo "ERROR: packages file not found: ${PACKAGES_FILE}" >&2
    exit 1
fi

awk -v indent="${INDENT}" '
    /^required:/ { in_required=1; next }
    /^[^[:space:]]/ && $0 !~ /^required:/ { if (in_required) exit }
    in_required && /^[[:space:]]*-[[:space:]]+/ {
        sub(/^[[:space:]]*-[[:space:]]+/, "", $0)
        printf "%s- %s\n", indent, $0
    }
' "${PACKAGES_FILE}"
