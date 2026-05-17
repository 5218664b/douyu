#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

PI_HOST="pi@192.168.2.105"
PI_PASSWORD="x"
REMOTE_DIR="/home/pi/douyu-rebuild"
REMOTE_CONFIG="${REMOTE_DIR}/configs/srs.conf"
REMOTE_RUNTIME_ENV="${REMOTE_DIR}/runtime/stream.env"
SYNC_RUNTIME_ENV="${SYNC_RUNTIME_ENV:-1}"
CONTAINER_NAME="${RELAY_CONTAINER_NAME:-douyu-rebuild-relay-1}"

sshpass -p "${PI_PASSWORD}" scp -o StrictHostKeyChecking=no \
  "${ROOT_DIR}/configs/srs.conf" \
  "${PI_HOST}:${REMOTE_CONFIG}.new"

if [ "${SYNC_RUNTIME_ENV}" = "1" ] && [ -f "${ROOT_DIR}/runtime/stream.env" ]; then
  sshpass -p "${PI_PASSWORD}" scp -o StrictHostKeyChecking=no \
    "${ROOT_DIR}/runtime/stream.env" \
    "${PI_HOST}:${REMOTE_RUNTIME_ENV}.new"
fi

sshpass -p "${PI_PASSWORD}" ssh -o StrictHostKeyChecking=no "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_DIR}/configs' '${REMOTE_DIR}/runtime'
  mv '${REMOTE_CONFIG}.new' '${REMOTE_CONFIG}'
  if [ '${SYNC_RUNTIME_ENV}' = '1' ] && [ -f '${REMOTE_RUNTIME_ENV}.new' ]; then
    mv '${REMOTE_RUNTIME_ENV}.new' '${REMOTE_RUNTIME_ENV}'
  fi
  docker restart '${CONTAINER_NAME}' >/dev/null
"

echo "synced relay config and restarted: ${CONTAINER_NAME}"
