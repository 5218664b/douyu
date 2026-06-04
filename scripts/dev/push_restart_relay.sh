#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="/home/pi/douyu"
REMOTE_CONFIG="${REMOTE_DIR}/configs/nginx-rtmp.conf"
REMOTE_RUNTIME_ENV="${REMOTE_DIR}/runtime/stream.env"
SYNC_RUNTIME_ENV="${SYNC_RUNTIME_ENV:-1}"
CONTAINER_NAME="douyu-relay"

sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
  "${ROOT_DIR}/configs/nginx-rtmp.conf" \
  "${PI_HOST}:${REMOTE_CONFIG}.new"

if [ "${SYNC_RUNTIME_ENV}" = "1" ] && [ -f "${ROOT_DIR}/runtime/stream.env" ]; then
  sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
    "${ROOT_DIR}/runtime/stream.env" \
    "${PI_HOST}:${REMOTE_RUNTIME_ENV}.new"
fi

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_DIR}/configs' '${REMOTE_DIR}/runtime'
  mv '${REMOTE_CONFIG}.new' '${REMOTE_CONFIG}'
  if [ '${SYNC_RUNTIME_ENV}' = '1' ] && [ -f '${REMOTE_RUNTIME_ENV}.new' ]; then
    mv '${REMOTE_RUNTIME_ENV}.new' '${REMOTE_RUNTIME_ENV}'
  fi
  cd '${REMOTE_DIR}'
  DOCKER_DEFAULT_PLATFORM=linux/arm64 docker compose up -d --force-recreate relay >/dev/null
"

echo "synced relay config and recreated: ${CONTAINER_NAME}"
