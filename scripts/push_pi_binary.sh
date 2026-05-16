#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
CONTAINER_NAME="${CONTAINER_NAME:-douyu-rebuild-app-1}"
SYNC_CONFIG="${SYNC_CONFIG:-1}"

BINARY_PATH="${ROOT_DIR}/bin/douyu-streamer"
CONFIG_PATH="${ROOT_DIR}/configs/app.yaml"

if [ ! -f "${BINARY_PATH}" ]; then
  echo "missing binary: ${BINARY_PATH}" >&2
  echo "run ./scripts/build_pi_binary.sh first" >&2
  exit 1
fi

if [ -n "${PI_PASSWORD}" ]; then
  SSH_PREFIX="sshpass -p ${PI_PASSWORD}"
  SCP_CMD="${SSH_PREFIX} scp -o StrictHostKeyChecking=no"
  SSH_CMD="${SSH_PREFIX} ssh -o StrictHostKeyChecking=no"
else
  SCP_CMD="scp"
  SSH_CMD="ssh"
fi

# shellcheck disable=SC2086
${SCP_CMD} "${BINARY_PATH}" "${PI_HOST}:${REMOTE_DIR}/douyu-streamer"

if [ "${SYNC_CONFIG}" = "1" ]; then
  # shellcheck disable=SC2086
  ${SCP_CMD} "${CONFIG_PATH}" "${PI_HOST}:${REMOTE_DIR}/configs/app.yaml"
fi

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "
  docker cp '${REMOTE_DIR}/douyu-streamer' '${CONTAINER_NAME}:/app/douyu-streamer' &&
  docker restart '${CONTAINER_NAME}'
"

echo "updated container: ${CONTAINER_NAME}"
