# Docker

This project is designed to run on Raspberry Pi 4B as a single Docker
container. The app binary and `ffmpeg` live in the same image.

Expected workflow:

```bash
cp .env.example .env
docker compose build
docker compose up -d
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
- exposes `/healthz`, `/state`, `/next`, and `/reload` on the local API
