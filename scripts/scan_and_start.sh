#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
SCAN_DIR="${ROOT_DIR}/tools/scan-launcher"

if [ -d "${SCAN_DIR}/node_modules" ] && command -v npm >/dev/null 2>&1; then
  cd "${SCAN_DIR}"
  npm run start
elif command -v docker >/dev/null 2>&1; then
  cd "${ROOT_DIR}"
  docker compose --profile scan run --rm scan-provider
else
  echo "missing scan launcher runtime" >&2
  echo "run npm install in tools/scan-launcher, or run the scan provider container with Docker" >&2
  exit 1
fi

exec "${ROOT_DIR}/scripts/start.sh"
