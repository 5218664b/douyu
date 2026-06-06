# Docker

This project is designed to run on Raspberry Pi 4B with a small set of Docker
services:

- `app`: streamer control and `ffmpeg`
- `relay`: local SRS RTMP relay for smoother upstream switching
- `scan-provider`: optional browser automation for QR-based startup

Recommended workflow for Raspberry Pi:

```bash
cp .env.example .env
docker load -i douyu-app-pi4b.tar
docker compose up -d
```

Recommended image build workflow on a stronger local machine:

```bash
docker buildx build --platform linux/arm64 -t douyu-app-base:pi4b --load -f docker/Dockerfile.base .
docker buildx build --platform linux/arm64 -t douyu-app:pi4b --load -f docker/Dockerfile .
docker buildx build --platform linux/arm64 -t douyu-relay:pi4b --load -f docker/Dockerfile.relay .
docker save douyu-app:pi4b -o douyu-app-pi4b.tar
```

Recommended Pi image push workflow:

```bash
./scripts/dev/menu.sh
```

If you prefer the direct scripts, use:

```bash
PI_PASSWORD='x' ./scripts/dev/push_pi_app_image.sh
PI_PASSWORD='x' ./scripts/dev/push_pi_relay_image.sh
PI_PASSWORD='x' ./scripts/dev/push_pi_scan_image.sh
```

For a full release-style refresh, use:

```bash
PI_PASSWORD='x' ./scripts/dev/release_pi_images.sh
```

For relay, the preferred deployment flow is:

```bash
PI_PASSWORD='x' ./scripts/dev/push_pi_relay_image.sh
./scripts/dev/push_restart_relay.sh
```

If you want to reuse the preinstalled `ffmpeg` layer across frequent app builds,
build and export the base image too:

```bash
docker save douyu-app-base:pi4b -o douyu-app-base-pi4b.tar
```

Runtime inputs:

- `configs/app.yaml` for static configuration
- `configs/nginx-rtmp.conf` for relay configuration template
- `.env` for sensitive or host-specific overrides
- `./data/videos` mounted into the container as the optional media source directory
- `./runtime/stream.env` for runtime stream credentials and relay forward target when not using static `.env` values
- `DOUYU_STREAMER_VIDEO_URLS` or `video.urls` for optional remote media inputs
- `DOUYU_STREAMER_NOTIFY_*` or `notify.*` for optional SMTP email alerts on streaming failures

The default compose file publishes the local API on `127.0.0.1:8080`.

Optional scan flow:

```bash
docker buildx build --platform linux/arm64 -t douyu-scan-provider-node:pi4b --load -f docker/Dockerfile.scan-launcher .
docker compose --profile scan run --rm scan-provider
```

The scan provider writes `runtime/stream.env`. The normal `app` and `relay` containers stay
separate from this browser automation step.

If you run the scan flow locally instead of in Docker, install the Node.js
dependencies in `tools/scan-launcher` and run `npm run start` there directly.

Current bootstrap behavior:

- scans the mounted media directory on startup when `video.source_dir` is configured
- appends any configured `video.urls` remote inputs to the playlist
- builds a sequential in-memory playlist from supported file extensions and remote URLs
- starts a single `ffmpeg` process for the current media item
- pushes media to a local RTMP relay at `rtmp://relay:1935/live/input`
- lets the relay container push onward to Douyu
- advances to the next item after normal media completion
- watches the `ffmpeg` process and attempts repeated restart of the current item after unexpected exit
- handles container stop signals and shuts down the child `ffmpeg` process cleanly
- exposes `/healthz`, `/state`, `/next`, and `/reload` on the local API

Relay implementation:

- SRS-based
- relay image defaults to `tiangolo/nginx-rtmp:latest`
- relay config template is mounted from host file `configs/nginx-rtmp.conf`
- relay startup loads `DOUYU_RELAY_FORWARD_URL` from `runtime/stream.env` first, then falls back to container env
- `scan-provider` writes `DOUYU_RELAY_FORWARD_URL` into `runtime/stream.env`
- after a new scan, sync `runtime/stream.env` and restart the relay container to apply the refreshed forward target
- relay upstream failure handling restarts the relay after repeated Douyu push failures; by default it does not stop the app stream
- set `DOUYU_RELAY_STOP_APP_ON_FAILURE=true` only if you explicitly want relay failures to call the app `/stop` endpoint
