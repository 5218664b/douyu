#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
RELAY_IMAGE="douyu-relay:pi4b"
RELAY_TAR_NAME="douyu-relay-pi4b.tar"
RELAY_TAR_PATH="${ROOT_DIR}/${RELAY_TAR_NAME}"

if [ -n "${PI_PASSWORD}" ]; then
  SSH_PREFIX="sshpass -p ${PI_PASSWORD}"
  SCP_CMD="${SSH_PREFIX} scp ${PI_SCP_OPTS}"
  SSH_CMD="${SSH_PREFIX} ssh ${PI_SSH_OPTS}"
else
  SCP_CMD="scp ${PI_SCP_OPTS}"
  SSH_CMD="ssh ${PI_SSH_OPTS}"
fi

docker buildx build --platform linux/arm64 -t "${RELAY_IMAGE}" --load -f "${ROOT_DIR}/docker/Dockerfile.relay" "${ROOT_DIR}"
docker save "${RELAY_IMAGE}" -o "${RELAY_TAR_PATH}"

# shellcheck disable=SC2086
${SCP_CMD} "${RELAY_TAR_PATH}" "${PI_HOST}:${REMOTE_DIR}/${RELAY_TAR_NAME}"

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "
  cd '${REMOTE_DIR}' &&
  docker load -i '${RELAY_TAR_NAME}'
"

# shellcheck disable=SC2086
${SSH_CMD} "${PI_HOST}" "docker image prune -f >/dev/null"

echo "pushed relay image: ${RELAY_IMAGE}"
