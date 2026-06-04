#!/bin/sh
set -eu

RUNTIME_ENV="/app/runtime/stream.env"
TEMPLATE_PATH="/etc/nginx/nginx.conf.template"
CONF_PATH="/etc/nginx/nginx.conf"
ERROR_LOG_PATH="/var/log/nginx/error.log"
APP_NOTIFY_URL="${DOUYU_APP_NOTIFY_URL:-http://app:8080/notify/problem}"
APP_NOTIFY_EVENT_URL="${DOUYU_APP_NOTIFY_EVENT_URL:-http://app:8080/notify/event}"
APP_STOP_URL="${DOUYU_APP_STOP_URL:-http://app:8080/stop}"
RELAY_ALERT_KIND="relay_push_failed"
RELAY_ALERT_SUMMARY="推流码可能已失效，推流失败"
RELAY_SUCCESS_SUMMARY="推流成功"
RELAY_SUCCESS_DETAIL="relay 已连续向斗鱼稳定转推 30 秒。"
SUCCESS_DELAY_SECONDS="${DOUYU_RELAY_SUCCESS_DELAY_SECONDS:-30}"
SUCCESS_TIMER_PID=""
RELAY_FAILURE_THRESHOLD="${DOUYU_RELAY_FAILURE_THRESHOLD:-4}"
RELAY_FAILURE_WINDOW_SECONDS="${DOUYU_RELAY_FAILURE_WINDOW_SECONDS:-180}"
FAILURE_COUNT=0
LAST_FAILURE_AT=0

read_runtime_env_value() {
  key="$1"

  if [ ! -f "${RUNTIME_ENV}" ]; then
    return 1
  fi

  line="$(grep "^${key}=" "${RUNTIME_ENV}" | tail -n 1 || true)"
  if [ -z "${line}" ]; then
    return 1
  fi

  printf '%s\n' "${line#*=}"
}

notify_problem() {
  detail="$1"

  payload="$(printf '{"kind":"%s","summary":"%s","detail":"%s"}' \
    "${RELAY_ALERT_KIND}" \
    "${RELAY_ALERT_SUMMARY}" \
    "$(printf '%s' "${detail}" | tr '"' "'" | tr '\n' ' ')")"

  curl -fsS -X POST \
    -H 'Content-Type: application/json' \
    -d "${payload}" \
    "${APP_NOTIFY_URL}" >/dev/null 2>&1 || true
}

notify_event() {
  summary="$1"
  detail="$2"

  payload="$(printf '{"summary":"%s","detail":"%s"}' \
    "$(printf '%s' "${summary}" | tr '"' "'" | tr '\n' ' ')" \
    "$(printf '%s' "${detail}" | tr '"' "'" | tr '\n' ' ')")"

  curl -fsS -X POST \
    -H 'Content-Type: application/json' \
    -d "${payload}" \
    "${APP_NOTIFY_EVENT_URL}" >/dev/null 2>&1
}

stop_app_stream() {
  curl -fsS -X POST "${APP_STOP_URL}" >/dev/null 2>&1
}

stop_relay() {
  kill -TERM 1 >/dev/null 2>&1 || true
}

cancel_success_timer() {
  if [ -n "${SUCCESS_TIMER_PID}" ] && kill -0 "${SUCCESS_TIMER_PID}" >/dev/null 2>&1; then
    echo "relay success timer cancelled" >&2
    kill "${SUCCESS_TIMER_PID}" >/dev/null 2>&1 || true
    wait "${SUCCESS_TIMER_PID}" >/dev/null 2>&1 || true
  fi
  SUCCESS_TIMER_PID=""
}

schedule_success_notification() {
  if [ -n "${SUCCESS_TIMER_PID}" ] && kill -0 "${SUCCESS_TIMER_PID}" >/dev/null 2>&1; then
    return
  fi

  (
    echo "relay success timer scheduled: ${SUCCESS_DELAY_SECONDS}s" >&2
    sleep "${SUCCESS_DELAY_SECONDS}"
    if notify_event "${RELAY_SUCCESS_SUMMARY}" "${RELAY_SUCCESS_DETAIL}"; then
      echo "relay success notification sent" >&2
    else
      echo "relay success notification failed" >&2
    fi
  ) &
  SUCCESS_TIMER_PID="$!"
}

watch_relay_errors() {
  touch "${ERROR_LOG_PATH}"

  tail -Fn0 "${ERROR_LOG_PATH}" | while IFS= read -r line; do
    echo "${line}"
    case "${line}" in
      *"relay: create push "*)
        cancel_success_timer
        schedule_success_notification
        ;;
      *"disconnect"*"sendhw"*|*"disconnect"*"douyu"*|*"push"*"failed"*|*"connect()"*"failed"*|*"NetStream.Publish.BadName"*|*"access denied"*|*"Broken pipe"*|*"Input/output error"*)
        cancel_success_timer
        NOW_AT="$(date +%s)"
        if [ $((NOW_AT - LAST_FAILURE_AT)) -gt "${RELAY_FAILURE_WINDOW_SECONDS}" ]; then
          FAILURE_COUNT=1
        else
          FAILURE_COUNT=$((FAILURE_COUNT + 1))
        fi
        LAST_FAILURE_AT="${NOW_AT}"
        echo "relay detected upstream push failure: ${line}" >&2
        notify_problem "${line}"
        if [ "${FAILURE_COUNT}" -ge "${RELAY_FAILURE_THRESHOLD}" ]; then
          echo "relay failure threshold reached: stopping app stream and relay" >&2
          stop_app_stream || true
          stop_relay
          FAILURE_COUNT=0
        fi
        ;;
      *"deleteStream, client: sendhw"*|*"deleteStream, client: "*".douyu."*)
        cancel_success_timer
        ;;
    esac
  done
}

if [ -z "${DOUYU_RELAY_FORWARD_URL:-}" ]; then
  DOUYU_RELAY_FORWARD_URL="$(read_runtime_env_value DOUYU_RELAY_FORWARD_URL || true)"
  export DOUYU_RELAY_FORWARD_URL
fi

if [ -z "${DOUYU_STREAMER_STREAM_RTMP_URL:-}" ]; then
  DOUYU_STREAMER_STREAM_RTMP_URL="$(read_runtime_env_value DOUYU_STREAMER_STREAM_RTMP_URL || true)"
  export DOUYU_STREAMER_STREAM_RTMP_URL
fi

if [ -z "${DOUYU_STREAMER_STREAM_KEY:-}" ]; then
  DOUYU_STREAMER_STREAM_KEY="$(read_runtime_env_value DOUYU_STREAMER_STREAM_KEY || true)"
  export DOUYU_STREAMER_STREAM_KEY
fi

if [ -z "${DOUYU_RELAY_FORWARD_URL:-}" ] && [ -n "${DOUYU_STREAMER_STREAM_RTMP_URL:-}" ] && [ -n "${DOUYU_STREAMER_STREAM_KEY:-}" ]; then
  DOUYU_RELAY_FORWARD_URL="$(printf '%s' "${DOUYU_STREAMER_STREAM_RTMP_URL}" | sed 's#/*$##')/${DOUYU_STREAMER_STREAM_KEY}"
  export DOUYU_RELAY_FORWARD_URL
fi

if [ -z "${DOUYU_RELAY_FORWARD_URL:-}" ]; then
  echo "missing DOUYU_RELAY_FORWARD_URL; run scan-provider first or set it in .env/runtime/stream.env" >&2
  exit 1
fi

case "${DOUYU_RELAY_FORWARD_URL}" in
  *replace-me*|*send.example.douyu.com*|"")
    echo "invalid DOUYU_RELAY_FORWARD_URL: ${DOUYU_RELAY_FORWARD_URL}" >&2
    echo "run scan-provider first and ensure runtime/stream.env contains a real Douyu forward target" >&2
    exit 1
    ;;
esac

echo "relay forward target: ${DOUYU_RELAY_FORWARD_URL}"

escaped_forward_url="$(printf '%s' "${DOUYU_RELAY_FORWARD_URL}" | sed 's/[&]/\\&/g')"
sed "s|\${DOUYU_RELAY_FORWARD_URL}|${escaped_forward_url}|g" "${TEMPLATE_PATH}" > "${CONF_PATH}"

watch_relay_errors &

exec nginx -g 'daemon off;'
