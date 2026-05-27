# Agent Rules

## Deployment And Build Policy

- Do not build application or relay images on the Raspberry Pi.
- Do not compile Go binaries on the Raspberry Pi.
- All builds must happen on the local development machine first.
- Do not assume the local development machine has `go` or `gofmt` installed in `PATH`.
- For Go build or formatting tasks, prefer the project's Docker-based builder workflow instead of relying on a host Go toolchain.
- The default builder path is `scripts/dev/push_restart_app.sh`, which starts or reuses the local `douyu-app-builder` container from `golang:1.23-bookworm`.
- If direct Go validation is needed, run it inside the same Docker builder container pattern used by the project.
- Raspberry Pi is only a deployment target for:
  - loading prebuilt images
  - syncing configuration
  - restarting or recreating containers
  - runtime verification and log inspection

## Operational Rule

- If a deployment problem occurs, fix the local build-and-push path rather than falling back to building on the Raspberry Pi.
