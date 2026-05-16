package stream

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/5218664b/douyu-streamer/internal/config"
	"github.com/5218664b/douyu-streamer/internal/library"
)

type ProcessState struct {
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	Command string `json:"command,omitempty"`
	Target  string `json:"target,omitempty"`
	Source  string `json:"source,omitempty"`
}

type ExitReason string

const (
	ExitReasonNone      ExitReason = ""
	ExitReasonStopped   ExitReason = "stopped"
	ExitReasonCompleted ExitReason = "completed"
	ExitReasonFailed    ExitReason = "failed"
)

type ExitEvent struct {
	Reason ExitReason
	Err    error
}

type Manager struct {
	mu    sync.RWMutex
	cfg   config.StreamConfig
	cmd   *exec.Cmd
	state ProcessState
	done  chan ExitEvent
	stopRequested bool
}

func New(cfg config.StreamConfig) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) Start(ctx context.Context, item library.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		return fmt.Errorf("stream process already running")
	}

	args := buildArgs(m.cfg, item.Path)
	cmd := exec.CommandContext(ctx, m.cfg.FFmpegPath, args...)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Start(); err != nil {
		return err
	}

	target := buildTarget(m.cfg)
	m.cmd = cmd
	m.done = make(chan ExitEvent, 1)
	m.stopRequested = false
	m.state = ProcessState{
		Running: true,
		PID:     cmd.Process.Pid,
		Command: strings.Join(append([]string{m.cfg.FFmpegPath}, args...), " "),
		Target:  target,
		Source:  item.Path,
	}

	go func(proc *exec.Cmd, done chan ExitEvent) {
		err := proc.Wait()
		if err == nil {
			done <- ExitEvent{Reason: ExitReasonCompleted}
			return
		}
		done <- ExitEvent{Reason: ExitReasonFailed, Err: err}
	}(cmd, m.done)

	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	err := m.cmd.Process.Kill()
	m.cmd = nil
	m.done = nil
	m.stopRequested = true
	m.state = ProcessState{}
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}

func (m *Manager) Snapshot() ProcessState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) Poll() ExitEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.done == nil {
		return ExitEvent{}
	}

	select {
	case event := <-m.done:
		m.cmd = nil
		m.done = nil
		m.state.Running = false
		m.state.PID = 0
		if m.stopRequested {
			m.stopRequested = false
			return ExitEvent{Reason: ExitReasonStopped}
		}
		if event.Reason == ExitReasonCompleted {
			return event
		}
		if event.Err == nil {
			event.Err = errors.New("stream process exited")
		}
		return event
	default:
		return ExitEvent{}
	}
}

func buildArgs(cfg config.StreamConfig, source string) []string {
	args := []string{
		"-hide_banner",
		"-loglevel", cfg.FFmpegLogLevel,
		"-re",
	}
	if cfg.LoopSingleInput {
		args = append(args, "-stream_loop", "-1")
	}
	args = append(args, "-i", source)

	if cfg.CopyVideo {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "veryfast")
	}

	if cfg.CopyAudio {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	}

	args = append(args, "-f", "flv", buildTarget(cfg))
	return args
}

func buildTarget(cfg config.StreamConfig) string {
	return strings.TrimRight(cfg.RTMPURL, "/") + "/" + cfg.StreamKey
}
