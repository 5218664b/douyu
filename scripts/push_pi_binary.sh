#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_HOST="${PI_HOST:-pi@192.168.2.105}"
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

scp "${BINARY_PATH}" "${PI_HOST}:${REMOTE_DIR}/douyu-streamer"

if [ "${SYNC_CONFIG}" = "1" ]; then
  scp "${CONFIG_PATH}" "${PI_HOST}:${REMOTE_DIR}/configs/app.yaml"
fi

ssh "${PI_HOST}" "
  docker cp '${REMOTE_DIR}/douyu-streamer' '${CONTAINER_NAME}:/app/douyu-streamer' &&
  docker restart '${CONTAINER_NAME}'
"

echo "updated container: ${CONTAINER_NAME}"
