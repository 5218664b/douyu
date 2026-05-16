package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/5218664b/douyu-streamer/internal/api"
	"github.com/5218664b/douyu-streamer/internal/config"
	"github.com/5218664b/douyu-streamer/internal/library"
	"github.com/5218664b/douyu-streamer/internal/playlist"
	"github.com/5218664b/douyu-streamer/internal/state"
	"github.com/5218664b/douyu-streamer/internal/stream"
)

func main() {
	cfgPath := os.Getenv("DOUYU_STREAMER_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/app.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	items, err := library.Scan(cfg.Video.SourceDir, cfg.Video.Formats)
	if err != nil {
		log.Fatalf("scan media library: %v", err)
	}

	queue, err := playlist.New(items)
	if err != nil {
		log.Fatalf("build playlist: %v", err)
	}

	runtime := state.New(cfg.Video.SourceDir, cfg.Danmaku.Enabled)
	runtime.SetPlaylist(queue.Current(), queue.Next(), queue.Items(), queue.History())
	runtime.SetStatus("preparing")

	streamManager := stream.New(cfg.Stream)
	if err := streamManager.Start(context.Background(), queue.Current()); err != nil {
		runtime.SetError(err.Error())
		log.Fatalf("start stream: %v", err)
	}
	runtime.SetProcess(streamManager.Snapshot())
	runtime.SetStatus("streaming")

	server := api.New(runtime)

	log.Printf("douyu-streamer starting: room=%s source_dir=%s items=%d api=%s", cfg.Room.URL, cfg.Video.SourceDir, len(queue.Items()), cfg.API.ListenAddr)
	if err := http.ListenAndServe(cfg.API.ListenAddr, server.Handler()); err != nil {
		log.Fatalf("serve api: %v", err)
	}
}
