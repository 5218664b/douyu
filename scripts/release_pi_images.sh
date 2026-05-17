#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

"${ROOT_DIR}/scripts/push_pi_app_image.sh"
"${ROOT_DIR}/scripts/push_pi_relay_image.sh"
"${ROOT_DIR}/scripts/push_pi_scan_image.sh"

echo "release images pushed to Raspberry Pi"
