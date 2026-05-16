package main

import (
	"log"
	"os"

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

	log.Printf("douyu-streamer starting: room=%s source_dir=%s", cfg.Room.URL, cfg.Video.SourceDir)
}
