package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	mux.HandleFunc("/notify/event", s.handleNotifyEvent)
	mux.HandleFunc("/notify/problem", s.handleNotifyProblem)
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

func (s *Server) handleNotifyEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload struct {
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.Detail = strings.TrimSpace(payload.Detail)
	if payload.Summary == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "summary is required"})
		return
	}
	if payload.Detail == "" {
		payload.Detail = payload.Summary
	}

	if err := s.app.SendEventEmail(context.Background(), payload.Summary, payload.Detail); err != nil {
		if errors.Is(err, context.Canceled) {
			writeJSON(w, http.StatusRequestTimeout, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleNotifyProblem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload struct {
		Kind    string `json:"kind"`
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	payload.Kind = strings.TrimSpace(payload.Kind)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.Detail = strings.TrimSpace(payload.Detail)
	if payload.Kind == "" || payload.Summary == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind and summary are required"})
		return
	}
	if payload.Detail == "" {
		payload.Detail = payload.Summary
	}

	if err := s.app.SendProblemEmail(context.Background(), payload.Kind, payload.Summary, payload.Detail); err != nil {
		if errors.Is(err, context.Canceled) {
			writeJSON(w, http.StatusRequestTimeout, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
