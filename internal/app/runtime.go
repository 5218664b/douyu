package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/5218664b/douyu-streamer/internal/config"
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

	r.state.SetStatus("preparing")
	if err := r.stream.Start(ctx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
	r.startMonitorLocked()
	return nil
}

func (r *Runtime) Next(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.stream.Stop(); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.playlist.Advance()
	r.syncState("switching")

	if err := r.stream.Start(ctx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
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

	if err := r.stream.Start(ctx, r.playlist.Current()); err != nil {
		r.state.SetError(err.Error())
		return err
	}

	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
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
		r.playlist.Advance()
		r.syncState("switching")

		restartErr := r.stream.Start(context.Background(), r.playlist.Current())
		if restartErr != nil {
			r.state.SetError(restartErr.Error())
			r.state.SetProcess(r.stream.Snapshot())
			return
		}

		r.state.ClearError()
		r.state.SetProcess(r.stream.Snapshot())
		r.state.SetStatus("streaming")
		return
	}

	r.state.SetError(err.Err.Error())
	r.state.SetStatus("recovering")

	restartErr := r.stream.Start(context.Background(), r.playlist.Current())
	if restartErr != nil {
		r.state.SetError(restartErr.Error())
		r.state.SetProcess(r.stream.Snapshot())
		return
	}

	r.state.ClearError()
	r.state.SetProcess(r.stream.Snapshot())
	r.state.SetStatus("streaming")
}
