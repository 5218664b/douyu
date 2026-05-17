package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	concatFile    string
}

func New(cfg config.StreamConfig) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) Start(ctx context.Context, item library.Item) error {
	return m.StartPlaylist(ctx, []library.Item{item}, 0)
}

func (m *Manager) StartPlaylist(ctx context.Context, items []library.Item, startIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		return fmt.Errorf("stream process already running")
	}

	if len(items) == 0 {
		return fmt.Errorf("playlist is empty")
	}
	if startIndex < 0 || startIndex >= len(items) {
		return fmt.Errorf("playlist start index out of range")
	}

	orderedItems := append([]library.Item(nil), items[startIndex:]...)
	orderedItems = append(orderedItems, items[:startIndex]...)

	concatFile, err := writeConcatFile(orderedItems)
	if err != nil {
		return err
	}

	args := buildConcatArgs(m.cfg, concatFile)
	cmd := exec.CommandContext(ctx, m.cfg.FFmpegPath, args...)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Start(); err != nil {
		_ = os.Remove(concatFile)
		return err
	}

	target := buildTarget(m.cfg)
	m.cmd = cmd
	m.concatFile = concatFile
	m.done = make(chan ExitEvent, 1)
	m.stopRequested = false
	m.state = ProcessState{
		Running: true,
		PID:     cmd.Process.Pid,
		Command: strings.Join(append([]string{m.cfg.FFmpegPath}, args...), " "),
		Target:  target,
		Source:  orderedItems[0].Path,
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
	m.cleanupLocked()
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
		m.cleanupLocked()
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

func buildConcatArgs(cfg config.StreamConfig, concatFile string) []string {
	args := []string{
		"-hide_banner",
		"-loglevel", cfg.FFmpegLogLevel,
		"-re",
	}
	if cfg.LoopSingleInput {
		args = append(args, "-stream_loop", "-1")
	}
	args = append(args, "-f", "concat", "-safe", "0", "-i", concatFile)

	if cfg.CopyVideo {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "veryfast")
	}

	if cfg.CopyAudio {
		args = append(args, "-c:a", "copy")
		args = append(args, "-bsf:a", "aac_adtstoasc")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	}

	args = append(args, "-f", "flv", buildTarget(cfg))
	return args
}

func buildTarget(cfg config.StreamConfig) string {
	return strings.TrimRight(cfg.RTMPURL, "/") + "/" + cfg.StreamKey
}

func writeConcatFile(items []library.Item) (string, error) {
	file, err := os.CreateTemp("", "douyu-streamer-*.ffconcat")
	if err != nil {
		return "", err
	}

	if _, err := io.WriteString(file, "ffconcat version 1.0\n"); err != nil {
		file.Close()
		_ = os.Remove(file.Name())
		return "", err
	}

	for _, item := range items {
		if _, err := fmt.Fprintf(file, "file '%s'\n", escapeConcatPath(item.Path)); err != nil {
			file.Close()
			_ = os.Remove(file.Name())
			return "", err
		}
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

func escapeConcatPath(path string) string {
	return strings.ReplaceAll(filepath.Clean(path), "'", "'\\''")
}

func (m *Manager) cleanupLocked() {
	if m.concatFile == "" {
		return
	}
	_ = os.Remove(m.concatFile)
	m.concatFile = ""
}
