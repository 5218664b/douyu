#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
REMOTE_RUNTIME_DIR="${REMOTE_DIR}/runtime"
REMOTE_SCAN_DIR="${REMOTE_DIR}/scan-launcher-src"
CONTAINER_NAME="douyu-scan"
SCAN_IMAGE="douyu-scan-provider-node:pi4b"

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_RUNTIME_DIR}' '${REMOTE_SCAN_DIR}.tmp'
"

sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
  "${ROOT_DIR}/tools/scan-launcher/start.js" \
  "${ROOT_DIR}/tools/scan-launcher/package.json" \
  "${ROOT_DIR}/tools/scan-launcher/package-lock.json" \
  "${PI_HOST}:${REMOTE_SCAN_DIR}.tmp/"

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_SCAN_DIR}'
  cp '${REMOTE_SCAN_DIR}.tmp/start.js' '${REMOTE_SCAN_DIR}/start.js'
  cp '${REMOTE_SCAN_DIR}.tmp/package.json' '${REMOTE_SCAN_DIR}/package.json'
  cp '${REMOTE_SCAN_DIR}.tmp/package-lock.json' '${REMOTE_SCAN_DIR}/package-lock.json'
  rm -rf '${REMOTE_SCAN_DIR}.tmp'
  docker rm -f '${CONTAINER_NAME}' >/dev/null 2>&1 || true
  rm -f '${REMOTE_RUNTIME_DIR}/douyu-login-qr.png'
  docker run -d \
    --name '${CONTAINER_NAME}' \
    --entrypoint node \
    -e DOUYU_SCAN_RUNTIME_DIR=/app/runtime \
    -v '${REMOTE_RUNTIME_DIR}:/app/runtime' \
    -v '${REMOTE_SCAN_DIR}/start.js:/app/start.js:ro' \
    -v '${REMOTE_SCAN_DIR}/package.json:/app/package.json:ro' \
    -v '${REMOTE_SCAN_DIR}/package-lock.json:/app/package-lock.json:ro' \
    '${SCAN_IMAGE}' \
    /app/start.js >/dev/null
"
