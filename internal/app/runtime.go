package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/5218664b/douyu-streamer/internal/config"
	"github.com/5218664b/douyu-streamer/internal/danmaku"
	"github.com/5218664b/douyu-streamer/internal/library"
	"github.com/5218664b/douyu-streamer/internal/playlist"
	"github.com/5218664b/douyu-streamer/internal/state"
	"github.com/5218664b/douyu-streamer/internal/stream"
)

type Runtime struct {
	mu      sync.Mutex
	cfg     config.Config
	state   *state.RuntimeState
	playlist *playlist.Playlist
	stream  *stream.Manager
	cancel  context.CancelFunc
	rootCtx context.Context
	danmaku *danmaku.Client
}

func New(cfg config.Config) (*Runtime, error) {
	items, err := library.Scan(cfg.Video.SourceDir, cfg.Video.Formats)
	if err != nil {
		return nil, fmt.Errorf("scan media library: %w", err)
	}

	queue, err := playlist.New(items)
	if err != nil {
		return nil, fmt.Errorf("build playlist: %w", err)
	}

	runtime := &Runtime{
		cfg:      cfg,
		state:    state.New(cfg.Video.SourceDir, cfg.Danmaku.Enabled),
		playlist: queue,
		stream:   stream.New(cfg.Stream),
		rootCtx:  context.Background(),
	}
	if cfg.Danmaku.Enabled {
		runtime.danmaku = danmaku.New(cfg.Room.URL, cfg.Danmaku.CommandPrefix, runtime.handleDanmakuCommand)
	}
	runtime.syncState("ready")

	return runtime, nil
}

func (r *Runtime) State() *state.RuntimeState {
	return r.state
}

func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rootCtx = ctx

	r.state.SetStatus("preparing")
	if err := r.stream.Start(r.rootCtx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: started stream source=%s target=%s", r.playlist.Current().Path, r.stream.Snapshot().Target)
	r.startMonitorLocked()
	if r.danmaku != nil {
		if err := r.danmaku.Start(r.rootCtx); err != nil {
			r.state.SetError(err.Error())
			log.Printf("runtime: danmaku start failed: %v", err)
		} else {
			log.Printf("runtime: danmaku enabled with prefix=%s", r.cfg.Danmaku.CommandPrefix)
		}
	}
	return nil
}

func (r *Runtime) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}

	r.state.SetStatus("stopping")
	err := r.stream.Stop()
	r.state.SetProcess(r.stream.Snapshot())
	if err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.ClearError()
	r.state.SetStatus("stopped")
	log.Printf("runtime: stopped")
	return nil
}

func (r *Runtime) Next(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nextLocked()
}

func (r *Runtime) nextLocked() error {
	log.Printf("runtime: switching to next item from=%s", r.playlist.Current().Path)

	if err := r.stream.Stop(); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.playlist.Advance()
	r.syncState("switching")

	if err := r.stream.Start(r.rootCtx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: switched to current=%s target=%s", r.playlist.Current().Path, r.stream.Snapshot().Target)
	return nil
}

func (r *Runtime) Select(ctx context.Context, index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.stream.Stop(); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	if _, err := r.playlist.Select(index); err != nil {
		r.state.SetError(err.Error())
		return err
	}
	log.Printf("runtime: selecting item index=%d path=%s", index, r.playlist.Current().Path)

	r.syncState("switching")
	if err := r.stream.Start(r.rootCtx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: selected current=%s target=%s", r.playlist.Current().Path, r.stream.Snapshot().Target)
	return nil
}

func (r *Runtime) Reload(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := library.Scan(r.cfg.Video.SourceDir, r.cfg.Video.Formats)
	if err != nil {
		r.state.SetError(err.Error())
		return err
	}

	queue, err := playlist.New(items)
	if err != nil {
		r.state.SetError(err.Error())
		return err
	}

	if err := r.stream.Stop(); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.playlist = queue
	r.syncState("reloading")
	log.Printf("runtime: reloaded media library items=%d current=%s", len(r.playlist.Items()), r.playlist.Current().Path)

	if err := r.stream.Start(r.rootCtx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: restarted after reload source=%s target=%s", r.playlist.Current().Path, r.stream.Snapshot().Target)
	return nil
}

func (r *Runtime) syncState(status string) {
	r.state.SetPlaylist(r.playlist.Current(), r.playlist.Next(), r.playlist.Items(), r.playlist.History())
	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus(status)
}

func (r *Runtime) startMonitorLocked() {
	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	go r.monitor(ctx)
}

func (r *Runtime) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkStream()
		}
	}
}

func (r *Runtime) checkStream() {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.stream.Poll()
	r.state.SetProcess(r.stream.Snapshot())
	if err.Reason == stream.ExitReasonNone || err.Reason == stream.ExitReasonStopped {
		return
	}

	if err.Reason == stream.ExitReasonCompleted {
		log.Printf("runtime: stream completed source=%s", r.playlist.Current().Path)
		r.playlist.Advance()
		r.syncState("switching")

		restartErr := r.stream.Start(r.rootCtx, r.playlist.Current())
		if restartErr != nil {
			r.scheduleRecoveryLocked(restartErr.Error())
			return
		}

		r.state.ClearError()
		r.state.SetProcess(r.stream.Snapshot())
		r.state.SetStatus("streaming")
		log.Printf("runtime: advanced to next source=%s", r.playlist.Current().Path)
		return
	}

	r.state.SetError(err.Err.Error())
	r.state.SetStatus("recovering")
	r.state.IncrementRecoveries()
	log.Printf("runtime: stream failure source=%s err=%v", r.playlist.Current().Path, err.Err)

	restartErr := r.stream.Start(r.rootCtx, r.playlist.Current())
	if restartErr != nil {
		r.scheduleRecoveryLocked(restartErr.Error())
		return
	}

	r.state.ClearError()
	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: recovered source=%s", r.playlist.Current().Path)
}

func (r *Runtime) handleDanmakuCommand(cmd string) {
	log.Printf("runtime: handling danmaku command=%s", cmd)
	switch cmd {
	case r.cfg.Danmaku.CommandPrefix + "next":
		_ = r.Next(context.Background())
		return
	case r.cfg.Danmaku.CommandPrefix + "reload":
		_ = r.Reload(context.Background())
		return
	}

	if index, ok := danmaku.ParseIndexCommand(cmd, r.cfg.Danmaku.CommandPrefix); ok {
		_ = r.Select(context.Background(), index)
	}
}

func (r *Runtime) scheduleRecoveryLocked(reason string) {
	r.state.SetError(reason)
	r.state.SetProcess(r.stream.Snapshot())
	log.Printf("runtime: scheduling recovery in 3s reason=%s", reason)
	go func() {
		time.Sleep(3 * time.Second)
		_ = r.retryRecover()
	}()
}

func (r *Runtime) retryRecover() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state.Snapshot().Status == "stopped" {
		return nil
	}

	r.state.SetStatus("recovering")
	r.state.IncrementRecoveries()
	log.Printf("runtime: retrying recovery source=%s", r.playlist.Current().Path)

	if err := r.stream.Start(r.rootCtx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		r.state.SetProcess(r.stream.Snapshot())
		log.Printf("runtime: recovery retry failed err=%v", err)
		go func() {
			time.Sleep(3 * time.Second)
			_ = r.retryRecover()
		}()
		return err
	}

	r.state.ClearError()
	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	log.Printf("runtime: recovery retry succeeded source=%s", r.playlist.Current().Path)
	return nil
}
