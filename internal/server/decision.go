package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/tomasz-tomczyk/crit/internal/decision"
)

// SetDecisionStore installs a dedicated router before serving. Review approval,
// agent hooks, repository file access and review cleanup are unavailable here.
// ServeHTTP still enforces the common Host and Sec-Fetch-Site guards.
func (s *Server) SetDecisionStore(store *decision.Store) {
	mux := http.NewServeMux()
	var clients atomic.Int32
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "ok", "browser_clients": clients.Load() > 0, "api_version": APIVersion, "mode": "decide"})
	})
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		state, _ := store.Snapshot()
		writeJSON(w, map[string]any{"mode": "decide", "session_id": state.SessionID, "revision": len(state.Rounds)})
	})
	mux.HandleFunc("GET /api/decision", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		state, _ := store.Snapshot()
		writeJSON(w, state)
	})
	mux.HandleFunc("PUT /api/decision", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Checklist    decision.Checklist `json:"checklist"`
			BaseRevision int                `json:"base_revision"`
			NewRound     bool               `json:"new_round"`
		}
		if !decodeDecision(w, r, &req) {
			return
		}
		state, err := store.Put(req.Checklist, req.BaseRevision, req.NewRound)
		if decisionError(w, err) {
			return
		}
		writeJSON(w, state)
	})
	mux.HandleFunc("PUT /api/decision/draft", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Revision     int                        `json:"revision"`
			DraftVersion int                        `json:"draft_version"`
			Draft        map[string]decision.Answer `json:"draft"`
		}
		if !decodeDecision(w, r, &req) {
			return
		}
		state, err := store.SaveDraft(req.Revision, req.DraftVersion, req.Draft)
		if decisionError(w, err) {
			return
		}
		writeJSON(w, state)
	})
	mux.HandleFunc("POST /api/decision/submit", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Revision     int    `json:"revision"`
			DraftVersion int    `json:"draft_version"`
			SubmissionID string `json:"submission_id"`
		}
		if !decodeDecision(w, r, &req) {
			return
		}
		result, err := store.Submit(req.Revision, req.DraftVersion, req.SubmissionID)
		if decisionError(w, err) {
			return
		}
		writeJSON(w, result)
	})
	mux.HandleFunc("GET /api/decision/wait", func(w http.ResponseWriter, r *http.Request) { s.waitDecision(store, w, r) })
	mux.HandleFunc("GET /api/decision/events", func(w http.ResponseWriter, r *http.Request) { s.decisionEvents(store, &clients, w, r) })
	page := func(w http.ResponseWriter, r *http.Request) {
		content, err := fs.ReadFile(s.assets, "decide.html")
		if err != nil {
			http.Error(w, "decision page unavailable", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(content)
	}
	mux.HandleFunc("GET /{$}", page)
	mux.HandleFunc("GET /decide", page)
	mux.HandleFunc("/api/", http.NotFound)
	mux.Handle("/", http.FileServer(http.FS(s.assets)))
	s.mux = mux
}
func decodeDecision(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, decision.MaxInputBytes)
	if err := decision.Decode(r.Body, value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}
func decisionError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	code := http.StatusInternalServerError
	if errors.Is(err, decision.ErrConflict) {
		code = http.StatusConflict
	}
	if errors.Is(err, decision.ErrInvalid) {
		code = http.StatusBadRequest
	}
	http.Error(w, err.Error(), code)
	return true
}

func (s *Server) waitDecision(store *decision.Store, w http.ResponseWriter, r *http.Request) {
	revision, err := strconv.Atoi(r.URL.Query().Get("revision"))
	if err != nil || revision < 1 {
		http.Error(w, "revision required", http.StatusBadRequest)
		return
	}
	timer := time.NewTimer(25 * time.Second)
	defer timer.Stop()
	for {
		state, changed := store.Snapshot()
		if revision <= len(state.Rounds) && state.Rounds[revision-1].Submission != nil {
			writeJSON(w, state.Rounds[revision-1].Submission)
			return
		}
		if revision != len(state.Rounds) {
			decisionError(w, decision.ErrConflict)
			return
		}
		select {
		case <-changed:
		case <-r.Context().Done():
			return
		case <-s.effectiveCtx().Done():
			http.Error(w, "decision server stopped", http.StatusServiceUnavailable)
			return
		case <-timer.C:
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
}

func (s *Server) decisionEvents(store *decision.Store, clients *atomic.Int32, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	clients.Add(1)
	defer clients.Add(-1)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		state, changed := store.Snapshot()
		round := state.Current()
		draftVersion := 0
		if round != nil {
			draftVersion = round.DraftVersion
		}
		// Always send a state hint on reconnect, including a submission at the same draft version.
		if _, err := fmt.Fprintf(w, "event: decision-updated\ndata: {\"revision\":%d,\"draft_version\":%d}\n\n", len(state.Rounds), draftVersion); err != nil {
			return
		}
		flusher.Flush()
		select {
		case <-changed:
		case <-ticker.C:
		case <-r.Context().Done():
			return
		case <-s.effectiveCtx().Done():
			return
		}
	}
}
