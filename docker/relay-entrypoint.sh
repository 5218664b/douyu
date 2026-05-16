#!/bin/sh
set -eu

RUNTIME_ENV="/app/runtime/stream.env"
TEMPLATE_PATH="/etc/nginx/nginx.conf.template"
CONF_PATH="/etc/nginx/nginx.conf"

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

echo "relay forward target: ${DOUYU_RELAY_FORWARD_URL}"

escaped_forward_url="$(printf '%s' "${DOUYU_RELAY_FORWARD_URL}" | sed 's/[&]/\\&/g')"
sed "s|\${DOUYU_RELAY_FORWARD_URL}|${escaped_forward_url}|g" "${TEMPLATE_PATH}" > "${CONF_PATH}"

exec nginx -g 'daemon off;'
