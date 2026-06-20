package server

import (
	"encoding/json"
	"net/http"
)

// handleEvents streams run-state changes and new alerts to the client as
// Server-Sent Events, so the GUI can surface updates within seconds without
// polling. Each event is a JSON object on a single `data:` line.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.broker == nil {
		writeError(w, http.StatusServiceUnavailable, CodeInternal, "", "event stream unavailable")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, CodeInternal, "", "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, cancel := s.broker.Subscribe()
	defer cancel()

	// Initial comment so clients know the stream is open.
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("event: " + string(ev.Kind) + "\ndata: ")); err != nil {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
