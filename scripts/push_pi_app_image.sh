#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
APP_BASE_IMAGE="${APP_BASE_IMAGE:-douyu-streamer-base:pi4b}"
APP_IMAGE="${APP_IMAGE:-douyu-streamer:pi4b}"
APP_BASE_TAR_NAME="${APP_BASE_TAR_NAME:-douyu-streamer-base-pi4b.tar}"
APP_TAR_NAME="${APP_TAR_NAME:-douyu-streamer-pi4b.tar}"
APP_BASE_TAR_PATH="${ROOT_DIR}/${APP_BASE_TAR_NAME}"
APP_TAR_PATH="${ROOT_DIR}/${APP_TAR_NAME}"
PUSH_BASE_IMAGE="${PUSH_BASE_IMAGE:-1}"

if [ -n "${PI_PASSWORD}" ]; then
  SSH_PREFIX="sshpass -p ${PI_PASSWORD}"
  SCP_CMD="${SSH_PREFIX} scp -o StrictHostKeyChecking=no"
  SSH_CMD="${SSH_PREFIX} ssh -o StrictHostKeyChecking=no"
else
  SCP_CMD="scp"
  SSH_CMD="ssh"
fi

if [ "${PUSH_BASE_IMAGE}" = "1" ]; then
  docker buildx build --platform linux/arm64 -t "${APP_BASE_IMAGE}" --load -f "${ROOT_DIR}/docker/Dockerfile.base" "${ROOT_DIR}"
  docker save "${APP_BASE_IMAGE}" -o "${APP_BASE_TAR_PATH}"
fi

docker buildx build --platform linux/arm64 -t "${APP_IMAGE}" --load -f "${ROOT_DIR}/docker/Dockerfile" "${ROOT_DIR}"
docker save "${APP_IMAGE}" -o "${APP_TAR_PATH}"

if [ "${PUSH_BASE_IMAGE}" = "1" ]; then
  # shellcheck disable=SC2086
  ${SCP_CMD} "${APP_BASE_TAR_PATH}" "${PI_HOST}:${REMOTE_DIR}/${APP_BASE_TAR_NAME}"
fi

# shellcheck disable=SC2086
${SCP_CMD} "${APP_TAR_PATH}" "${PI_HOST}:${REMOTE_DIR}/${APP_TAR_NAME}"

if [ "${PUSH_BASE_IMAGE}" = "1" ]; then
  # shellcheck disable=SC2086
  ${SSH_CMD} "${PI_HOST}" "
    cd '${REMOTE_DIR}' &&
    docker load -i '${APP_BASE_TAR_NAME}' &&
    docker load -i '${APP_TAR_NAME}'
  "
else
  # shellcheck disable=SC2086
  ${SSH_CMD} "${PI_HOST}" "
    cd '${REMOTE_DIR}' &&
    docker load -i '${APP_TAR_NAME}'
  "
fi

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "docker image prune -f >/dev/null"

echo "pushed app image: ${APP_IMAGE}"
