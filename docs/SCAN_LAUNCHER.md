# Scan Launcher

This document describes the scan-to-start workflow for `douyu-streamer`.

## Goal

Provide a lightweight launcher plus an optional scan provider that:

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
  start.sh
  scan_and_start.sh
  start_with_scan.sh
```

## Current status

An initial Playwright scan provider is in place.

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

The default local wrapper no longer installs Node dependencies at runtime.
It expects either:

- the scan-provider container to be used, or
- `tools/scan-launcher/node_modules` to have been prepared in advance

## Container-first requirements

Preferred runtime:

- Docker
- `docker compose`

The optional scan-provider container carries:

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

Build the scan-provider image:

```bash
docker buildx build --platform linux/arm64 -t douyu-scan-provider-playwright:pi4b --load -f docker/Dockerfile.scan-launcher .
```

Run the optional scan profile:

```bash
docker compose --profile scan run --rm scan-provider
```

Default local startup path:

```bash
scripts/start.sh
```

If `runtime/stream.env` is not present yet, use:

```bash
scripts/scan_and_start.sh
```

For local Node-based use, prepare dependencies once:

```bash
cd tools/scan-launcher
npm install
```
