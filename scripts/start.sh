#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RUNTIME_ENV="${ROOT_DIR}/runtime/stream.env"

if [ ! -f "${RUNTIME_ENV}" ]; then
  echo "missing runtime env: ${RUNTIME_ENV}" >&2
  echo "run scripts/scan_and_start.sh first, or write DOUYU_STREAMER_STREAM_RTMP_URL and DOUYU_STREAMER_STREAM_KEY into runtime/stream.env" >&2
  exit 1
fi

set -a
. "${RUNTIME_ENV}"
set +a

if [ -z "${DOUYU_STREAMER_STREAM_RTMP_URL:-}" ] || [ -z "${DOUYU_STREAMER_STREAM_KEY:-}" ]; then
  echo "runtime env is missing DOUYU_STREAMER_STREAM_RTMP_URL or DOUYU_STREAMER_STREAM_KEY: ${RUNTIME_ENV}" >&2
  exit 1
fi

exec "${ROOT_DIR}/bin/douyu-streamer"
