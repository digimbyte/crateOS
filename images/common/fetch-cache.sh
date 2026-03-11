#!/bin/bash
set -euo pipefail

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "usage: fetch-cache.sh <scope> <url> [label]" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

SCOPE="$1"
URL="$2"
LABEL="${3:-artifact}"
FILE_NAME="$(basename "$URL")"
CACHE_DIR="${REPO_ROOT}/.cache/${SCOPE}"
TARGET_PATH="${CACHE_DIR}/${FILE_NAME}"

mkdir -p "${CACHE_DIR}"

if [ -f "${TARGET_PATH}" ]; then
    echo "==> Reusing cached ${LABEL}: ${TARGET_PATH}" >&2
else
    echo "==> Downloading ${LABEL}..." >&2
    wget -q --show-progress -O "${TARGET_PATH}" "${URL}"
fi

printf '%s\n' "${TARGET_PATH}"
