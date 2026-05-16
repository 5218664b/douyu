package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/5218664b/douyu-streamer/internal/app"
	"github.com/5218664b/douyu-streamer/internal/api"
	"github.com/5218664b/douyu-streamer/internal/config"
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

	runtime, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init runtime: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		log.Fatalf("start runtime: %v", err)
	}

	server := api.New(runtime)

	log.Printf("douyu-streamer starting: room=%s source_dir=%s api=%s", cfg.Room.URL, cfg.Video.SourceDir, cfg.API.ListenAddr)
	if err := http.ListenAndServe(cfg.API.ListenAddr, server.Handler()); err != nil {
		log.Fatalf("serve api: %v", err)
	}
}
