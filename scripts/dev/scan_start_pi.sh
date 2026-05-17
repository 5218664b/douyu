#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

PI_HOST="${PI_HOST:-pi@192.168.2.105}"
PI_PASSWORD="${PI_PASSWORD:-x}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu-rebuild}"
REMOTE_RUNTIME_ENV="${REMOTE_DIR}/runtime/stream.env"
REMOTE_QR_IMAGE="${REMOTE_DIR}/runtime/douyu-login-qr.png"
SCAN_CONTAINER_NAME="${SCAN_CONTAINER_NAME:-douyu-scan}"

sshpass -p "${PI_PASSWORD}" ssh -o StrictHostKeyChecking=no "${PI_HOST}" "
  rm -f '${REMOTE_RUNTIME_ENV}' '${REMOTE_QR_IMAGE}'
"

"${ROOT_DIR}/scripts/dev/push_restart_scan_provider.sh"

sshpass -p "${PI_PASSWORD}" ssh -tt -o StrictHostKeyChecking=no "${PI_HOST}" "
  docker logs -f '${SCAN_CONTAINER_NAME}'
" &
LOGS_PID=$!

cleanup() {
  kill "${LOGS_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

EXIT_CODE="$(sshpass -p "${PI_PASSWORD}" ssh -o StrictHostKeyChecking=no "${PI_HOST}" "
  docker wait '${SCAN_CONTAINER_NAME}'
")"

if [ "${EXIT_CODE}" != "0" ]; then
  echo "scan-provider failed with exit code: ${EXIT_CODE}" >&2
  exit 1
fi

if ! sshpass -p "${PI_PASSWORD}" ssh -o StrictHostKeyChecking=no "${PI_HOST}" "
  test -f '${REMOTE_RUNTIME_ENV}' &&
  grep -q '^DOUYU_STREAMER_STREAM_RTMP_URL=' '${REMOTE_RUNTIME_ENV}' &&
  grep -q '^DOUYU_STREAMER_STREAM_KEY=' '${REMOTE_RUNTIME_ENV}'
"; then
  echo "scan-provider exited successfully, but runtime stream credentials were not written" >&2
  exit 1
fi

cleanup
trap - EXIT INT TERM

echo "stream credentials received; restarting relay..."
"${ROOT_DIR}/scripts/dev/push_restart_relay.sh"

echo "restarting app..."
SYNC_CONFIG=0 "${ROOT_DIR}/scripts/dev/push_restart_app.sh"

echo "scan completed and streaming restarted"
