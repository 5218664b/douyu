#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

if [ ! -f "${ROOT_DIR}/.env" ]; then
  cp "${ROOT_DIR}/.env.example" "${ROOT_DIR}/.env"
  echo "created .env from .env.example"
fi

"${ROOT_DIR}/scripts/pi/redeploy.sh"
"${ROOT_DIR}/scripts/pi/scan_start.sh"
