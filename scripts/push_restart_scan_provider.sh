#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-x}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
REMOTE_RUNTIME_DIR="${REMOTE_DIR}/runtime"
CONTAINER_NAME="${CONTAINER_NAME:-douyu-scan-test}"
SCAN_IMAGE="${SCAN_IMAGE:-douyu-scan-provider-node:pi4b}"

sshpass -p "${PI_PASSWORD}" ssh -o StrictHostKeyChecking=no "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_RUNTIME_DIR}'
  docker rm -f '${CONTAINER_NAME}' >/dev/null 2>&1 || true
  rm -f '${REMOTE_RUNTIME_DIR}/douyu-login-qr.png'
  docker run -d \
    --name '${CONTAINER_NAME}' \
    -v '${REMOTE_RUNTIME_DIR}:/app/runtime' \
    '${SCAN_IMAGE}'
"

echo "rebuilt and restarted: ${CONTAINER_NAME}"
