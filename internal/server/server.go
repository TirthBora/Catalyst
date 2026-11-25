package server

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"catalyst/internal/project"
	"catalyst/internal/proxy"
	"catalyst/internal/websocket"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	hub         *websocket.Hub
	port        string
	projectRoot string
	projectInfo project.Info
	httpServer  *http.Server
	mu          sync.Mutex
}

func NewServer(port string, hub *websocket.Hub, projectRoot string) *Server {
	return &Server{
		hub:         hub,
		port:        port,
		projectRoot: projectRoot,
		projectInfo: project.Detect(projectRoot),
	}
}

// Handler builds and returns the HTTP request multiplexer.
func (s *Server) Handler() (http.Handler, error) {
	subFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.hub.HandleWS)

	// Workspace metadata endpoint
	mux.HandleFunc("/api/workspace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]interface{}{
			"name":         s.projectInfo.Name,
			"type":         s.projectInfo.Type,
			"framework":    s.projectInfo.Framework,
			"defaultPort":  s.projectInfo.DefaultPort,
			"rootDir":      s.projectRoot,
			"serverPort":   s.port,
			"clientsCount": s.hub.ClientCount(),
		}
		_ = json.NewEncoder(w).Encode(data)
	})

	// Reverse proxy endpoint for local development servers
	reverseProxy := proxy.NewReverseProxy()
	mux.Handle("/proxy/", reverseProxy)

	// Catalyst UI
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// Project preview
	if _, err := os.Stat(s.projectRoot); err != nil {
		return nil, err
	}

	projectFS := http.FileServer(
		http.Dir(filepath.Clean(s.projectRoot)),
	)

	mux.Handle("/preview/",
		http.StripPrefix("/preview", projectFS),
	)

	return mux, nil
}

func (s *Server) Start() error {
	handler, err := s.Handler()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:    ":" + s.port,
		Handler: handler,
	}
	s.mu.Unlock()

	err = s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
