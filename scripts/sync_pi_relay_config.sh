#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
SYNC_RUNTIME_ENV="${SYNC_RUNTIME_ENV:-1}"

SRS_CONF_PATH="${ROOT_DIR}/configs/srs.conf"
RUNTIME_ENV_PATH="${ROOT_DIR}/runtime/stream.env"

if [ -n "${PI_PASSWORD}" ]; then
  SSH_PREFIX="sshpass -p ${PI_PASSWORD}"
  SCP_CMD="${SSH_PREFIX} scp -o StrictHostKeyChecking=no"
  SSH_CMD="${SSH_PREFIX} ssh -o StrictHostKeyChecking=no"
else
  SCP_CMD="scp"
  SSH_CMD="ssh"
fi

# shellcheck disable=SC2086
${SCP_CMD} "${SRS_CONF_PATH}" "${PI_HOST}:${REMOTE_DIR}/configs/srs.conf"

if [ "${SYNC_RUNTIME_ENV}" = "1" ] && [ -f "${RUNTIME_ENV_PATH}" ]; then
  # shellcheck disable=SC2086
  ${SCP_CMD} "${RUNTIME_ENV_PATH}" "${PI_HOST}:${REMOTE_DIR}/runtime/stream.env"
fi

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "
  cd '${REMOTE_DIR}' &&
  docker compose restart relay
"

echo "synced relay config and restarted relay"
