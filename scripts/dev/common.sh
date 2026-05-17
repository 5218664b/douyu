#!/usr/bin/env sh

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-x}"
PI_SSH_PORT="${PI_SSH_PORT:-22}"
PI_SCP_OPTS="-P ${PI_SSH_PORT} -o StrictHostKeyChecking=no"
PI_SSH_OPTS="-p ${PI_SSH_PORT} -o StrictHostKeyChecking=no"
