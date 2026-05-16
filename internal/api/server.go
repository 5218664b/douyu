package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/5218664b/douyu-streamer/internal/app"
	"github.com/5218664b/douyu-streamer/internal/state"
)

type Server struct {
	runtime *state.RuntimeState
	app     *app.Runtime
}

func New(appRuntime *app.Runtime) *Server {
	return &Server{
		runtime: appRuntime.State(),
		app:     appRuntime,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/state", s.handleState)
	mux.HandleFunc("/next", s.handleNext)
	mux.HandleFunc("/reload", s.handleReload)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.runtime.Snapshot())
}

func (s *Server) handleNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if err := s.app.Next(context.Background()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, s.runtime.Snapshot())
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if err := s.app.Reload(context.Background()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, s.runtime.Snapshot())
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
