package state

import (
	"sync"

	"github.com/5218664b/douyu-streamer/internal/library"
	"github.com/5218664b/douyu-streamer/internal/stream"
)

type RuntimeState struct {
	mu          sync.RWMutex
	status      string
	current     *library.Item
	next        *library.Item
	queue       []library.Item
	history     []library.Item
	lastError   string
	sourceDir   string
	danmakuOn   bool
	process     stream.ProcessState
}

type Snapshot struct {
	Status    string         `json:"status"`
	Current   *library.Item  `json:"current,omitempty"`
	Next      *library.Item  `json:"next,omitempty"`
	Queue     []library.Item `json:"queue"`
	History   []library.Item `json:"history"`
	LastError string         `json:"last_error,omitempty"`
	SourceDir string         `json:"source_dir"`
	DanmakuOn bool           `json:"danmaku_enabled"`
	Process   stream.ProcessState `json:"process"`
}

func New(sourceDir string, danmakuOn bool) *RuntimeState {
	return &RuntimeState{
		status:    "idle",
		sourceDir: sourceDir,
		danmakuOn: danmakuOn,
	}
}

func (s *RuntimeState) SetPlaylist(current, next library.Item, queue, history []library.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = "ready"
	s.current = cloneItem(current)
	s.next = cloneItem(next)
	s.queue = append([]library.Item(nil), queue...)
	s.history = append([]library.Item(nil), history...)
}

func (s *RuntimeState) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func (s *RuntimeState) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
	s.status = "error"
}

func (s *RuntimeState) ClearError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = ""
}

func (s *RuntimeState) SetProcess(process stream.ProcessState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.process = process
}

func (s *RuntimeState) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Snapshot{
		Status:    s.status,
		Current:   cloneItemPtr(s.current),
		Next:      cloneItemPtr(s.next),
		Queue:     append([]library.Item(nil), s.queue...),
		History:   append([]library.Item(nil), s.history...),
		LastError: s.lastError,
		SourceDir: s.sourceDir,
		DanmakuOn: s.danmakuOn,
		Process:   s.process,
	}
}

func cloneItem(item library.Item) *library.Item {
	copied := item
	return &copied
}

func cloneItemPtr(item *library.Item) *library.Item {
	if item == nil {
		return nil
	}
	copied := *item
	return &copied
}
