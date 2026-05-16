# Docker

This project is designed to run on Raspberry Pi 4B as a single Docker
container. The app binary and `ffmpeg` live in the same image.

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
docker save douyu-streamer:pi4b -o douyu-streamer-pi4b.tar
```

If you want to reuse the preinstalled `ffmpeg` layer across frequent app builds,
build and export the base image too:

```bash
docker save douyu-streamer-base:pi4b -o douyu-streamer-base-pi4b.tar
```

Runtime inputs:

- `configs/app.yaml` for static configuration
- `.env` for sensitive or host-specific overrides
- `./data/videos` mounted into the container as the media source directory

The default compose file publishes the local API on `127.0.0.1:8080`.

Current bootstrap behavior:

- scans the mounted media directory on startup
- builds a sequential in-memory playlist from supported file extensions
- starts a single `ffmpeg` process for the current media item
- advances to the next item after normal media completion
- watches the `ffmpeg` process and attempts repeated restart of the current item after unexpected exit
- handles container stop signals and shuts down the child `ffmpeg` process cleanly
- exposes `/healthz`, `/state`, `/next`, and `/reload` on the local API
