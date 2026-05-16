# Docker

This project is designed to run on Raspberry Pi 4B with a small set of Docker
services:

- `app`: streamer control and `ffmpeg`
- `relay`: local SRS RTMP relay for smoother upstream switching
- `scan-provider`: optional browser automation for QR-based startup

Recommended workflow for Raspberry Pi:

```bash
cp .env.example .env
docker load -i douyu-streamer-pi4b.tar
docker compose up -d
```

Recommended image build workflow on a stronger local machine:

```bash
docker buildx build --platform linux/arm64 -t douyu-streamer-base:pi4b --load -f docker/Dockerfile.base .
docker buildx build --platform linux/arm64 -t douyu-streamer:pi4b --load -f docker/Dockerfile .
docker buildx build --platform linux/arm64 -t douyu-relay:pi4b --load -f docker/Dockerfile.relay .
docker save douyu-streamer:pi4b -o douyu-streamer-pi4b.tar
```

For relay, the preferred deployment flow is:

```bash
PI_PASSWORD='x' ./scripts/push_pi_relay_image.sh
PI_PASSWORD='x' ./scripts/sync_pi_relay_config.sh
```

If you want to reuse the preinstalled `ffmpeg` layer across frequent app builds,
build and export the base image too:

```bash
docker save douyu-streamer-base:pi4b -o douyu-streamer-base-pi4b.tar
```

Runtime inputs:

- `configs/app.yaml` for static configuration
- `configs/nginx-rtmp.conf` for relay configuration template
- `.env` for sensitive or host-specific overrides
- `./data/videos` mounted into the container as the media source directory
- `./runtime/stream.env` for runtime stream credentials and relay forward target when not using static `.env` values

The default compose file publishes the local API on `127.0.0.1:8080`.

Optional scan flow:

```bash
docker buildx build --platform linux/arm64 -t douyu-scan-provider-playwright:pi4b --load -f docker/Dockerfile.scan-launcher .
docker compose --profile scan run --rm scan-provider
```

The scan provider writes `runtime/stream.env`. The normal app container stays
separate from this browser automation step.

If you run the scan flow locally instead of in Docker, install the Node
dependencies once under `tools/scan-launcher/` before invoking
`scripts/scan_and_start.sh`.

Current bootstrap behavior:

- scans the mounted media directory on startup
- builds a sequential in-memory playlist from supported file extensions
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
