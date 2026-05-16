package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	httpServer := &http.Server{
		Addr:    cfg.API.ListenAddr,
		Handler: server.Handler(),
	}

	log.Printf(
		"douyu-streamer starting: room=%s source_dir=%s api=%s stream_target=%s/%s",
		cfg.Room.URL,
		cfg.Video.SourceDir,
		cfg.API.ListenAddr,
		cfg.Stream.RTMPURL,
		cfg.Stream.StreamKey,
	)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve api: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if err := runtime.Shutdown(); err != nil {
		log.Fatalf("runtime shutdown: %v", err)
	}

	log.Printf("douyu-streamer stopped")
}
