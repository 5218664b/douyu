#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RUNTIME_ENV="${ROOT_DIR}/runtime/stream.env"

cd "${ROOT_DIR}/tools/scan-launcher"
npm install
npm run start

if [ ! -f "${RUNTIME_ENV}" ]; then
  echo "missing runtime env: ${RUNTIME_ENV}" >&2
  exit 1
fi

set -a
. "${RUNTIME_ENV}"
set +a

exec "${ROOT_DIR}/bin/douyu-streamer"
