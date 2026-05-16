# Build

This project targets Raspberry Pi 4B with a Go binary plus system `ffmpeg`.

Expected local build command once Go is installed:

```bash
go build -o bin/douyu-streamer ./cmd/streamer
```

Runtime configuration is loaded from `configs/app.yaml` by default, or from the
path defined in `DOUYU_STREAMER_CONFIG`.
