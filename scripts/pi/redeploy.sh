#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

cd "${ROOT_DIR}"
DOCKER_DEFAULT_PLATFORM=linux/arm64 docker compose up -d --force-recreate app relay scan-provider

echo "redeployed app, relay, and scan-provider locally on Raspberry Pi"
