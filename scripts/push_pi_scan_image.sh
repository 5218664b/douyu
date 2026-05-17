#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
SCAN_IMAGE="${SCAN_IMAGE:-douyu-scan-provider-node:pi4b}"
SCAN_TAR_NAME="${SCAN_TAR_NAME:-douyu-scan-provider-node.tar}"
SCAN_TAR_PATH="${ROOT_DIR}/${SCAN_TAR_NAME}"

if [ -n "${PI_PASSWORD}" ]; then
  SSH_PREFIX="sshpass -p ${PI_PASSWORD}"
  SCP_CMD="${SSH_PREFIX} scp -o StrictHostKeyChecking=no"
  SSH_CMD="${SSH_PREFIX} ssh -o StrictHostKeyChecking=no"
else
  SCP_CMD="scp"
  SSH_CMD="ssh"
fi

docker save "${SCAN_IMAGE}" -o "${SCAN_TAR_PATH}"

# shellcheck disable=SC2086
${SCP_CMD} "${SCAN_TAR_PATH}" "${PI_HOST}:${REMOTE_DIR}/${SCAN_TAR_NAME}"

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "
  cd '${REMOTE_DIR}' &&
  docker load -i '${SCAN_TAR_NAME}'
"

echo "pushed scan image: ${SCAN_IMAGE}"
