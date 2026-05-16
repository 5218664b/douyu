# Binary Workflow

This project supports a fast Raspberry Pi validation workflow based on a single
`linux/arm64` binary instead of repeated Docker image transfer.

## Build on local machine

Use the provided helper script:

```bash
./scripts/build_pi_binary.sh
```

This uses `golang:1.22-bookworm` in Docker to build:

```text
bin/douyu-streamer
```

## Copy to Raspberry Pi

```bash
scp bin/douyu-streamer pi@<pi-ip>:/home/pi/douyu-rebuild/
scp configs/app.yaml pi@<pi-ip>:/home/pi/douyu-rebuild/configs/app.yaml
```

If needed, also sync the media directory or point `configs/app.yaml` at the
existing media path on the Pi.

## Run on Raspberry Pi

```bash
ssh pi@<pi-ip>
cd /home/pi/douyu-rebuild
chmod +x ./douyu-streamer
./douyu-streamer
```

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

If the container already exists on the Pi, you can push the new binary directly
into it instead of rebuilding or reloading the whole image:

```bash
./scripts/build_pi_binary.sh
PI_HOST=pi@<pi-ip> ./scripts/push_pi_binary.sh
```

Defaults used by the helper script:

- remote directory: `/home/pi/douyu-rebuild`
- container name: `douyu-rebuild-app-1`
- config sync: enabled

You can override them:

```bash
PI_HOST=pi@<pi-ip> \
REMOTE_DIR=/home/pi/douyu-rebuild \
CONTAINER_NAME=douyu-rebuild-app-1 \
SYNC_CONFIG=0 \
./scripts/push_pi_binary.sh
```
