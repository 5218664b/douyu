# Scan Launcher

This document describes the scan-to-start workflow for `douyu-streamer`.

## Goal

Provide a lightweight launcher plus an optional scan provider that:

1. opens a Douyu login flow
2. waits for QR-code scan
3. obtains `rtmp_url` and `stream_key`
4. writes them into a runtime env file
5. probes the scanned target with a short ffmpeg test push
6. starts `douyu-streamer` only if the probe succeeds

## Current layout

```text
tools/scan-launcher/
  start.js
docker/
  Dockerfile.scan-launcher
runtime/
  stream.env
scripts/
  dev/menu.sh
  dev/scan_start_pi.sh
  dev/push_restart_scan_provider.sh
  pi/scan_start.sh
```

## Current status

The active scan provider uses Node.js browser automation.

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

The preferred workflows now target Raspberry Pi deployment or direct local
execution inside `tools/scan-launcher`.

## Container-first requirements

Preferred runtime:

- Docker
- `docker compose`

The optional scan-provider container carries:

- Node.js
- Puppeteer
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
docker buildx build --platform linux/arm64 -t douyu-scan-provider-node:pi4b --load -f docker/Dockerfile.scan-launcher .
```

Run the optional scan profile:

```bash
docker compose --profile scan run --rm scan-provider
```

Default local startup path:

```bash
cd tools/scan-launcher
npm run start
```

If `runtime/stream.env` is not present yet, use:

```bash
./scripts/dev/menu.sh
```

Then choose `1` to start the QR-scan flow on Raspberry Pi from your host machine.

For local Node.js-based use:

```bash
cd tools/scan-launcher
npm install
npm run start
```

For Raspberry Pi local use directly on the device:

```bash
./scripts/pi/scan_start.sh
```
