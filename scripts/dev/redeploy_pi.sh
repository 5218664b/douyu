#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu}"
VIDEO_SOURCE_HOST_DIR="${VIDEO_SOURCE_HOST_DIR:-/home/pi/samba/hard02/magic/电视剧/士兵突击(Soldiers Sortie)624x336.X264.AAC.350M.30集全[DVDRip]/output}"
SYNC_ENV="${SYNC_ENV:-1}"

scp_files="
  ${ROOT_DIR}/docker-compose.yml
  ${ROOT_DIR}/.env.example
  ${ROOT_DIR}/configs/app.yaml
  ${ROOT_DIR}/configs/nginx-rtmp.conf
"

if [ "${SYNC_ENV}" = "1" ] && [ -f "${ROOT_DIR}/.env" ]; then
  scp_files="${scp_files}
  ${ROOT_DIR}/.env
"
fi

sshpass -p "${PI_PASSWORD}" scp ${PI_SCP_OPTS} \
  ${scp_files} \
  "${PI_HOST}:${REMOTE_DIR}/"

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  set -eu
  mkdir -p '${REMOTE_DIR}/configs'
  cp '${REMOTE_DIR}/app.yaml' '${REMOTE_DIR}/configs/app.yaml'
  cp '${REMOTE_DIR}/nginx-rtmp.conf' '${REMOTE_DIR}/configs/nginx-rtmp.conf'
  if [ -f '${REMOTE_DIR}/.env' ]; then
    grep -v '^DOUYU_VIDEO_SOURCE_HOST_DIR=' '${REMOTE_DIR}/.env' > '${REMOTE_DIR}/.env.tmp' || true
    mv '${REMOTE_DIR}/.env.tmp' '${REMOTE_DIR}/.env'
  fi
  printf '%s\n' 'DOUYU_VIDEO_SOURCE_HOST_DIR=${VIDEO_SOURCE_HOST_DIR}' >> '${REMOTE_DIR}/.env'
  cd '${REMOTE_DIR}'
  DOCKER_DEFAULT_PLATFORM=linux/arm64 docker compose up -d --force-recreate app relay
"

echo "redeployed app and relay on Raspberry Pi"
