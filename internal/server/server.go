package server

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"catalyst/internal/websocket"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	hub         *websocket.Hub
	port        string
	projectRoot string
}

func NewServer(port string, hub *websocket.Hub, projectRoot string) *Server {
	return &Server{
		hub:         hub,
		port:        port,
		projectRoot: projectRoot,
	}
}

func (s *Server) Start() error {
	// Embedded Catalyst UI
	subFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.hub.HandleWS)

	// Catalyst UI
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// Project preview
	if _, err := os.Stat(s.projectRoot); err != nil {
		return err
	}

	projectFS := http.FileServer(
		http.Dir(filepath.Clean(s.projectRoot)),
	)

	mux.Handle("/preview/",
		http.StripPrefix("/preview", projectFS),
	)

	return http.ListenAndServe(":"+s.port, mux)
}
