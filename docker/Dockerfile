FROM golang:1.22-bookworm AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/douyu-streamer ./cmd/streamer

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ffmpeg ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/douyu-streamer /app/douyu-streamer
COPY configs/app.yaml /app/configs/app.yaml

ENV DOUYU_STREAMER_CONFIG=/app/configs/app.yaml

CMD ["/app/douyu-streamer"]
