#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
OUTPUT_DIR="${ROOT_DIR}/bin"
IMAGE="${GO_BUILDER_IMAGE:-golang:1.22-bookworm}"

mkdir -p "${OUTPUT_DIR}"

docker run --rm \
  -v "${ROOT_DIR}:/src" \
  -w /src \
  "${IMAGE}" \
  sh -lc 'CGO_ENABLED=0 GOOS=linux GOARCH=arm64 /usr/local/go/bin/go build -buildvcs=false -o /src/bin/douyu-streamer ./cmd/streamer'

echo "built: ${OUTPUT_DIR}/douyu-streamer"
