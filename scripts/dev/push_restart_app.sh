#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="/home/pi/douyu-rebuild"
REMOTE_BIN="${REMOTE_DIR}/douyu-streamer"
REMOTE_CONFIG="${REMOTE_DIR}/configs/app.yaml"
CONTAINER_NAME="douyu-app"
SYNC_CONFIG="${SYNC_CONFIG:-1}"
BUILDER_CONTAINER_NAME="douyu-app-builder"
BUILDER_IMAGE="${APP_BUILDER_IMAGE:-golang:1.23-bookworm}"

if ! docker image inspect "${BUILDER_IMAGE}" >/dev/null 2>&1; then
  docker pull "${BUILDER_IMAGE}" >/dev/null
fi

recreate_builder_container=false
if docker ps -a --format '{{.Names}}' | grep -Fx "${BUILDER_CONTAINER_NAME}" >/dev/null 2>&1; then
  current_builder_image="$(docker inspect -f '{{.Config.Image}}' "${BUILDER_CONTAINER_NAME}")"
  if [ "${current_builder_image}" != "${BUILDER_IMAGE}" ]; then
    docker rm -f "${BUILDER_CONTAINER_NAME}" >/dev/null
    recreate_builder_container=true
  fi
else
  recreate_builder_container=true
fi

if [ "${recreate_builder_container}" = false ]; then
  docker start "${BUILDER_CONTAINER_NAME}" >/dev/null
else
  docker run -d \
    --name "${BUILDER_CONTAINER_NAME}" \
    -v "${ROOT_DIR}:/src" \
    -w /src \
    "${BUILDER_IMAGE}" \
    tail -f /dev/null >/dev/null
fi

docker exec \
  "${BUILDER_CONTAINER_NAME}" \
  sh -lc 'CGO_ENABLED=0 GOOS=linux GOARCH=arm64 /usr/local/go/bin/go build -buildvcs=false -o /src/bin/douyu-streamer ./cmd/streamer'

sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
  "${ROOT_DIR}/bin/douyu-streamer" \
  "${PI_HOST}:${REMOTE_BIN}.new"

if [ "${SYNC_CONFIG}" = "1" ]; then
  sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
    "${ROOT_DIR}/configs/app.yaml" \
    "${PI_HOST}:${REMOTE_CONFIG}.new"
fi

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_DIR}/configs'
  mv '${REMOTE_BIN}.new' '${REMOTE_BIN}'
  chmod +x '${REMOTE_BIN}'
  if [ '${SYNC_CONFIG}' = '1' ]; then
    mv '${REMOTE_CONFIG}.new' '${REMOTE_CONFIG}'
  fi
  docker cp '${REMOTE_BIN}' '${CONTAINER_NAME}:/app/douyu-streamer'
  docker restart '${CONTAINER_NAME}' >/dev/null
"

echo "rebuilt and restarted: ${CONTAINER_NAME}"
