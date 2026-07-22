package browser

import (
	"net/http"
	"sync"
)

type Reloader struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func NewReloader() *Reloader {
	return &Reloader{
		clients: make(map[chan struct{}]struct{}),
	}
}

func (r *Reloader) Handler(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{})

	r.mu.Lock()
	r.clients[ch] = struct{}{}
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.clients, ch)
		r.mu.Unlock()
		close(ch)
	}()

	flusher.Flush()

	for {
		select {
		case <-req.Context().Done():
			return

		case <-ch:
			_, _ = w.Write([]byte("data: reload\n\n"))
			flusher.Flush()
		}
	}
}
func (r *Reloader) Notify() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ch := range r.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (r *Reloader) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/__catalyst/events", r.Handler)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			// Version 1: ignore server shutdown errors.
		}
	}()

	return nil
}
