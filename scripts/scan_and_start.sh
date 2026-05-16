#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
SCAN_DIR="${ROOT_DIR}/tools/scan-launcher"

if [ ! -d "${SCAN_DIR}/node_modules" ]; then
  echo "missing scan launcher dependencies: ${SCAN_DIR}/node_modules" >&2
  echo "build and run the scan provider container, or install dependencies once with: cd ${SCAN_DIR} && npm install" >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required to run the local scan launcher" >&2
  echo "use the scan provider container instead, or install Node.js and npm on this machine" >&2
  exit 1
fi

cd "${SCAN_DIR}"
npm run start

exec "${ROOT_DIR}/scripts/start.sh"
