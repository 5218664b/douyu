# Scan Launcher

This document describes the scan-to-start workflow for `douyu-streamer`.

## Goal

Provide a separate launcher that:

1. opens a Douyu login flow
2. waits for QR-code scan
3. obtains `rtmp_url` and `stream_key`
4. writes them into a runtime env file
5. starts `douyu-streamer`

## Current layout

```text
tools/scan-launcher/
  package.json
  start.js
docker/
  Dockerfile.scan-launcher
runtime/
  stream.env
scripts/
  start_with_scan.sh
```

## Current status

An initial Playwright implementation is in place.

Current behavior:

- opens the Douyu login page in a browser, headless by default
- extracts the QR code canvas
- saves the QR image to `runtime/douyu-login-qr.png`
- prints a terminal QR when decode succeeds
- waits for scan completion
- opens the creator live page
- attempts to read `rtmp_url` and `stream_key` via page clipboard actions
- writes them to `runtime/stream.env`
- starts `bin/douyu-streamer` with exported runtime variables

## Container-first requirements

Preferred runtime:

- Docker
- `docker compose`

The launcher container carries:

- Node.js
- Playwright
- Chromium runtime

Optional environment variables:

- `DOUYU_SCAN_BROWSER_PATH`
- `DOUYU_SCAN_ROOM_URL`
- `DOUYU_SCAN_HEADLESS`

## Raspberry Pi usage

For direct Pi usage, the intended mode is:

- run the launcher container in terminal
- print the QR code in terminal
- scan it with a phone
- continue the login flow without requiring a visible desktop browser

## Suggested flow

Build the launcher image:

```bash
docker buildx build --platform linux/arm64 -t douyu-scan-launcher:pi4b --load -f docker/Dockerfile.scan-launcher .
```

Run the scan profile:

```bash
docker compose --profile scan run --rm scan-launcher
```
