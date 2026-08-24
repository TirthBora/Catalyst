package server

import (
	"embed"
	"io/fs"
	"net/http"

	"catalyst/internal/websocket"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	hub  *websocket.Hub
	port string
}

func NewServer(port string, hub *websocket.Hub) *Server {
	return &Server{
		hub:  hub,
		port: port,
	}
}

func (s *Server) Start() error {
	subFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.hub.HandleWS)            // The WebSocket route
	mux.Handle("/", http.FileServer(http.FS(subFS))) // The UI route

	return http.ListenAndServe(":"+s.port, mux)
}
