#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
RUNTIME_DIR="${ROOT_DIR}/runtime"
RUNTIME_ENV="${RUNTIME_DIR}/stream.env"
QR_IMAGE="${RUNTIME_DIR}/douyu-login-qr.png"
SCAN_CONTAINER_NAME="douyu-scan"
RELAY_CONTAINER_NAME="douyu-relay"
APP_CONTAINER_NAME="douyu-app"
APP_API_URL="${APP_API_URL:-http://127.0.0.1:8080/state}"
APP_STABILITY_SECONDS="${APP_STABILITY_SECONDS:-15}"
APP_STARTUP_TIMEOUT_SECONDS="${APP_STARTUP_TIMEOUT_SECONDS:-30}"

fetch_app_state() {
  curl -fsS "${APP_API_URL}" || true
}

validate_stream_env() {
  [ -f "${RUNTIME_ENV}" ] || return 1

  rtmp_url="$(grep '^DOUYU_STREAMER_STREAM_RTMP_URL=' "${RUNTIME_ENV}" | tail -n 1 | cut -d= -f2-)"
  stream_key="$(grep '^DOUYU_STREAMER_STREAM_KEY=' "${RUNTIME_ENV}" | tail -n 1 | cut -d= -f2-)"

  printf '%s\n' "${rtmp_url}" | grep -Eq '^rtmp://send[a-z0-9.-]*\.douyu\.com/live$'
  printf '%s\n' "${stream_key}" | grep -q 'wsSecret='
  printf '%s\n' "${stream_key}" | grep -q 'wsTime='
}

probe_stream_target() {
  rtmp_url="$(grep '^DOUYU_STREAMER_STREAM_RTMP_URL=' "${RUNTIME_ENV}" | tail -n 1 | cut -d= -f2-)"
  stream_key="$(grep '^DOUYU_STREAMER_STREAM_KEY=' "${RUNTIME_ENV}" | tail -n 1 | cut -d= -f2-)"
  target="${rtmp_url%/}/${stream_key}"

  docker exec "${APP_CONTAINER_NAME}" sh -lc '
    ffmpeg -hide_banner -loglevel error \
      -f lavfi -i testsrc2=size=640x360:rate=15 \
      -f lavfi -i anullsrc=r=44100:cl=stereo \
      -t 5 \
      -c:v libx264 -preset ultrafast \
      -c:a aac -b:a 128k \
      -f flv "$0"
  ' "${target}"
}

verify_app_streaming() {
  startup_deadline="$(( $(date +%s) + ${APP_STARTUP_TIMEOUT_SECONDS} ))"
  stable_count=0
  last_state=""

  while [ "$(date +%s)" -lt "${startup_deadline}" ]; do
    state="$(fetch_app_state)"
    if [ -n "${state}" ]; then
      last_state="${state}"
    fi
    if [ -n "${state}" ] \
      && printf '%s\n' "${state}" | grep -q '"status":"streaming"' \
      && printf '%s\n' "${state}" | grep -q '"running":true'; then
      stable_count=$((stable_count + 1))
      if [ "${stable_count}" -ge "${APP_STABILITY_SECONDS}" ]; then
        return 0
      fi
    else
      stable_count=0
    fi
    sleep 1
  done

  if [ -n "${last_state}" ]; then
    printf '%s\n' "${last_state}" >&2
  else
    echo "no app state response received from ${APP_API_URL}" >&2
  fi

  return 1
}

app_state_is_stopped() {
  state="$(fetch_app_state)"
  if [ -n "${state}" ]; then
    printf '%s\n' "${state}" >&2
  fi
  [ -n "${state}" ] && printf '%s\n' "${state}" | grep -q '"status":"stopped"'
}

restart_app_container() {
  docker restart "${APP_CONTAINER_NAME}" >/dev/null
}

rm -f "${RUNTIME_ENV}" "${QR_IMAGE}"

docker rm -f "${SCAN_CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d \
  --name "${SCAN_CONTAINER_NAME}" \
  --network "container:${APP_CONTAINER_NAME}" \
  --entrypoint node \
  -e DOUYU_SCAN_RUNTIME_DIR=/app/runtime \
  -e DOUYU_APP_NOTIFY_EVENT_URL=http://127.0.0.1:8080/notify/event \
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

if ! validate_stream_env; then
  echo "scan-provider exited successfully, but runtime stream credentials are missing or invalid; refusing to restart streaming" >&2
  exit 1
fi

echo "scan credentials captured successfully"

echo "probing scanned stream target..."
if ! probe_stream_target; then
  echo "scanned stream target probe failed; refusing to restart streaming" >&2
  exit 1
fi

cleanup
trap - EXIT INT TERM

docker restart "${RELAY_CONTAINER_NAME}" >/dev/null

echo "verifying app streaming stability..."
if ! verify_app_streaming; then
  if app_state_is_stopped; then
    echo "app is stopped after relay restart; restarting app container and verifying again..." >&2
    restart_app_container
    if verify_app_streaming; then
      echo "scan completed after app restart"
      exit 0
    fi
  fi
  echo "scan credentials were captured, but app did not stay in streaming state after relay restart" >&2
  echo "check 'docker logs ${RELAY_CONTAINER_NAME}' for upstream Douyu disconnects and 'curl -fsS ${APP_API_URL}' for current app state" >&2
  exit 1
fi

echo "scan completed and relay restarted"
