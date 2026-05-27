ARG DOUYU_STREAMER_BASE_IMAGE=douyu-app-base:pi4b
FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/douyu-streamer ./cmd/streamer

FROM ${DOUYU_STREAMER_BASE_IMAGE}

COPY --from=builder /out/douyu-streamer /app/douyu-streamer
COPY configs/app.yaml /app/configs/app.yaml

ENV DOUYU_STREAMER_CONFIG=/app/configs/app.yaml

CMD ["/app/douyu-streamer"]
