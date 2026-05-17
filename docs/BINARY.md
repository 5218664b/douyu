# Binary Workflow

This project supports a fast Raspberry Pi validation workflow based on a single
`linux/arm64` binary instead of repeated Docker image transfer.

## One Command

Use the unified helper script:

```bash
./scripts/dev/menu.sh
```

Then choose `2` to update and restart the Raspberry Pi app only.

This script will:

- start or reuse a local builder container
- compile `bin/douyu-streamer` for `linux/arm64`
- copy the new binary to the Raspberry Pi
- optionally sync `configs/app.yaml`
- copy both into the running app container
- restart the app container

## Validate

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/state
curl -X POST http://127.0.0.1:8080/next
curl -X POST http://127.0.0.1:8080/reload
```

## When to prefer this workflow

Prefer the binary workflow when:

- you are iterating on code frequently
- you want the shortest edit-test loop
- Docker image export/import is too slow

Prefer Docker when:

- you are validating the final packaged runtime
- you want the exact deployment shape used in production

## Fast binary refresh into a running container

`scripts/dev/push_restart_app.sh` assumes the container already exists on the Pi and updates
it in place instead of rebuilding or reloading the whole image.

```bash
./scripts/dev/push_restart_app.sh
```

Defaults baked into the helper script:

- host: `pi@192.168.2.105`
- password: `x`
- remote directory: `/home/pi/douyu-rebuild`
- container name: `douyu-rebuild-app-1`
- config sync: enabled

Optional overrides:

```bash
SYNC_CONFIG=0 ./scripts/dev/push_restart_app.sh
APP_CONTAINER_NAME=another-app-container ./scripts/dev/push_restart_app.sh
APP_BUILDER_IMAGE=golang:1.22-bookworm ./scripts/dev/push_restart_app.sh
```
