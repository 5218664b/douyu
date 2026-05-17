#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
RUNTIME_DIR="${ROOT_DIR}/runtime"
RUNTIME_ENV="${RUNTIME_DIR}/stream.env"
QR_IMAGE="${RUNTIME_DIR}/douyu-login-qr.png"
SCAN_CONTAINER_NAME="douyu-scan"
RELAY_CONTAINER_NAME="douyu-relay"
APP_CONTAINER_NAME="douyu-app"

rm -f "${RUNTIME_ENV}" "${QR_IMAGE}"

docker rm -f "${SCAN_CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d \
  --name "${SCAN_CONTAINER_NAME}" \
  --entrypoint node \
  -e DOUYU_SCAN_RUNTIME_DIR=/app/runtime \
  -v "${RUNTIME_DIR}:/app/runtime" \
  -v "${ROOT_DIR}/tools/scan-launcher/start.js:/app/start.js:ro" \
  -v "${ROOT_DIR}/tools/scan-launcher/package.json:/app/package.json:ro" \
  -v "${ROOT_DIR}/tools/scan-launcher/package-lock.json:/app/package-lock.json:ro" \
  "douyu-scan-provider-node:pi4b" \
  /app/start.js >/dev/null

docker logs -f "${SCAN_CONTAINER_NAME}" &
LOGS_PID=$!

cleanup() {
  kill "${LOGS_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

EXIT_CODE="$(docker wait "${SCAN_CONTAINER_NAME}")"

if [ "${EXIT_CODE}" != "0" ]; then
  echo "scan-provider failed with exit code: ${EXIT_CODE}" >&2
  exit 1
fi

if [ ! -f "${RUNTIME_ENV}" ] || \
   ! grep -q '^DOUYU_STREAMER_STREAM_RTMP_URL=' "${RUNTIME_ENV}" || \
   ! grep -q '^DOUYU_STREAMER_STREAM_KEY=' "${RUNTIME_ENV}"; then
  echo "scan-provider exited successfully, but runtime stream credentials were not written" >&2
  exit 1
fi

cleanup
trap - EXIT INT TERM

docker restart "${RELAY_CONTAINER_NAME}" >/dev/null
docker restart "${APP_CONTAINER_NAME}" >/dev/null

echo "scan completed and streaming restarted"
