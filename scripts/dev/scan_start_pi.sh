#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1091
. "${SCRIPT_DIR}/common.sh"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/douyu}"
REMOTE_RUNTIME_ENV="${REMOTE_DIR}/runtime/stream.env"
REMOTE_QR_IMAGE="${REMOTE_DIR}/runtime/douyu-login-qr.png"
SCAN_CONTAINER_NAME="douyu-scan"
APP_CONTAINER_NAME="${APP_CONTAINER_NAME:-douyu-app}"
APP_API_URL="${APP_API_URL:-http://127.0.0.1:8080/state}"
APP_STABILITY_SECONDS="${APP_STABILITY_SECONDS:-15}"
APP_STARTUP_TIMEOUT_SECONDS="${APP_STARTUP_TIMEOUT_SECONDS:-30}"

remote_app_container_exists() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    docker ps -a --format '{{.Names}}' | grep -Fx '${APP_CONTAINER_NAME}' >/dev/null
  "
}

ensure_remote_app_and_relay_ready() {
  if remote_app_container_exists; then
    return 0
  fi

  echo "remote app container is missing; redeploying app and relay before scan..." >&2
  "${ROOT_DIR}/scripts/dev/redeploy_pi.sh"
}

fetch_remote_app_state() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    curl -fsS '${APP_API_URL}' || true
  "
}

validate_remote_stream_env() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    set -eu
    test -f '${REMOTE_RUNTIME_ENV}'
    rtmp_url=\$(grep '^DOUYU_STREAMER_STREAM_RTMP_URL=' '${REMOTE_RUNTIME_ENV}' | tail -n 1 | cut -d= -f2-)
    stream_key=\$(grep '^DOUYU_STREAMER_STREAM_KEY=' '${REMOTE_RUNTIME_ENV}' | tail -n 1 | cut -d= -f2-)
    printf '%s\n' \"\${rtmp_url}\" | grep -Eq '^rtmp://send[a-z0-9.-]*\\.douyu\\.com/live$'
    printf '%s\n' \"\${stream_key}\" | grep -q 'wsSecret='
    printf '%s\n' \"\${stream_key}\" | grep -q 'wsTime='
  "
}

probe_remote_stream_target() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    set -eu
    rtmp_url=\$(grep '^DOUYU_STREAMER_STREAM_RTMP_URL=' '${REMOTE_RUNTIME_ENV}' | tail -n 1 | cut -d= -f2-)
    stream_key=\$(grep '^DOUYU_STREAMER_STREAM_KEY=' '${REMOTE_RUNTIME_ENV}' | tail -n 1 | cut -d= -f2-)
    target=\"\${rtmp_url%/}/\${stream_key}\"
    docker exec '${APP_CONTAINER_NAME}' sh -lc '
      ffmpeg -hide_banner -loglevel error \
        -f lavfi -i testsrc2=size=640x360:rate=15 \
        -f lavfi -i anullsrc=r=44100:cl=stereo \
        -t 5 \
        -c:v libx264 -preset ultrafast \
        -c:a aac -b:a 128k \
        -f flv \"\$0\"
    ' \"\${target}\"
  "
}

verify_remote_app_streaming() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    set -eu
    startup_deadline=\$((\$(date +%s) + ${APP_STARTUP_TIMEOUT_SECONDS}))
    stable_count=0
    last_state=''
    while [ \$(date +%s) -lt \${startup_deadline} ]; do
      state=\$(curl -fsS '${APP_API_URL}' || true)
      if [ -n \"\${state}\" ]; then
        last_state=\"\${state}\"
      fi
      if [ -n \"\${state}\" ] \
        && printf '%s\n' \"\${state}\" | grep -q '\"status\":\"streaming\"' \
        && printf '%s\n' \"\${state}\" | grep -q '\"running\":true'; then
        stable_count=\$((stable_count + 1))
        if [ \"\${stable_count}\" -ge ${APP_STABILITY_SECONDS} ]; then
          exit 0
        fi
      else
        stable_count=0
      fi
      sleep 1
    done
    if [ -n \"\${last_state}\" ]; then
      printf '%s\n' \"\${last_state}\" >&2
    else
      echo 'no app state response received from ${APP_API_URL}' >&2
    fi
    exit 1
  "
}

remote_app_state_is_stopped() {
  state="$(fetch_remote_app_state)"
  if [ -n "${state}" ]; then
    printf '%s\n' "${state}" >&2
  fi
  [ -n "${state}" ] && printf '%s\n' "${state}" | grep -q '"status":"stopped"'
}

restart_remote_app_container() {
  sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
    docker restart '${APP_CONTAINER_NAME}' >/dev/null
  "
}

sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  rm -f '${REMOTE_RUNTIME_ENV}' '${REMOTE_QR_IMAGE}'
"

ensure_remote_app_and_relay_ready

"${ROOT_DIR}/scripts/dev/push_restart_scan_provider.sh"

sshpass -p "${PI_PASSWORD}" ssh -tt ${PI_SSH_OPTS} "${PI_HOST}" "
  docker logs -f '${SCAN_CONTAINER_NAME}'
" &
LOGS_PID=$!

cleanup() {
  kill "${LOGS_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

EXIT_CODE="$(sshpass -p "${PI_PASSWORD}" ssh ${PI_SSH_OPTS} "${PI_HOST}" "
  docker wait '${SCAN_CONTAINER_NAME}'
")"

if [ "${EXIT_CODE}" != "0" ]; then
  echo "scan-provider failed with exit code: ${EXIT_CODE}" >&2
  exit 1
fi

if ! validate_remote_stream_env; then
  echo "scan-provider exited successfully, but runtime stream credentials are missing or invalid; refusing to restart streaming" >&2
  exit 1
fi

echo "scan credentials captured successfully"

echo "probing scanned stream target..."
if ! probe_remote_stream_target; then
  echo "scanned stream target probe failed; refusing to restart streaming" >&2
  exit 1
fi

cleanup
trap - EXIT INT TERM

echo "stream credentials received; restarting relay..."
"${ROOT_DIR}/scripts/dev/push_restart_relay.sh"

echo "verifying app streaming stability..."
if ! verify_remote_app_streaming; then
  if remote_app_state_is_stopped; then
    echo "app is stopped after relay restart; restarting app container and verifying again..." >&2
    restart_remote_app_container
    if verify_remote_app_streaming; then
      echo "scan completed after app restart"
      exit 0
    fi
  fi
  echo "scan credentials were captured, but app did not stay in streaming state after relay restart" >&2
  echo "check remote 'docker logs douyu-relay' for upstream Douyu disconnects and 'curl -fsS ${APP_API_URL}' for current app state" >&2
  exit 1
fi

echo "scan completed and relay restarted"
