package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"catalyst/internal/config"
	"catalyst/internal/monitor"
	"catalyst/internal/pairing"
	"catalyst/internal/process"
	"catalyst/internal/project"
	"catalyst/internal/proxy"
	"catalyst/internal/runtime"
	"catalyst/internal/watcher"
	"catalyst/internal/websocket"
	"catalyst/internal/workspace"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	hub          *websocket.Hub
	port         string
	projectRoot  string
	projectInfo  project.Info
	processMgr   *process.Manager
	pairingMgr   *pairing.Manager
	configMgr    *config.Manager
	workspaceReg *workspace.Registry
	watcher      *watcher.Watcher
	httpServer   *http.Server
	mu           sync.Mutex
}

func NewServer(port string, hub *websocket.Hub, projectRoot string, pm *process.Manager, pairingMgr *pairing.Manager, cfgMgr *config.Manager, wsReg *workspace.Registry, w *watcher.Watcher) *Server {
	if pm == nil {
		pm = process.NewManager(hub)
	}
	if pairingMgr == nil {
		pairingMgr = pairing.NewManager()
	}
	if cfgMgr == nil {
		cfgMgr = config.NewManager(projectRoot)
	}
	if wsReg == nil {
		wsReg = workspace.NewRegistry(projectRoot)
	}
	return &Server{
		hub:          hub,
		port:         port,
		projectRoot:  projectRoot,
		projectInfo:  project.Detect(projectRoot),
		processMgr:   pm,
		pairingMgr:   pairingMgr,
		configMgr:    cfgMgr,
		workspaceReg: wsReg,
		watcher:      w,
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

	// System metrics endpoint
	mux.HandleFunc("/api/system", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := monitor.GetStats()
		_ = json.NewEncoder(w).Encode(stats)
	})

	// Environment doctor diagnostics endpoint
	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		report := runtime.Diagnose(s.projectRoot)
		_ = json.NewEncoder(w).Encode(report)
	})

	// Process observation endpoint (read-only monitoring of supervised processes)
	mux.HandleFunc("/api/processes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.processMgr.List())
	})

	// Multi-workspace management endpoints
	mux.HandleFunc("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var body struct {
				Path string `json:"path"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
				http.Error(w, "Invalid workspace path", http.StatusBadRequest)
				return
			}
			if _, err := os.Stat(body.Path); err != nil {
				http.Error(w, "Directory does not exist", http.StatusBadRequest)
				return
			}
			ws := s.workspaceReg.Register(body.Path)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(ws)
			return
		}
		_ = json.NewEncoder(w).Encode(s.workspaceReg.List())
	})

	mux.HandleFunc("/api/workspaces/switch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
			http.Error(w, "Workspace ID required", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		ws, err := s.workspaceReg.SetActive(body.ID)
		if err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		s.projectRoot = ws.Path
		s.projectInfo = ws.Project
		s.configMgr = config.NewManager(ws.Path)

		if s.watcher != nil {
			_ = s.watcher.SwitchRoot(ws.Path)
			if custom := s.configMgr.Get().CustomIgnore; len(custom) > 0 {
				s.watcher.SetCustomIgnore(custom)
			}
		}
		s.mu.Unlock()

		if s.hub != nil {
			s.hub.Broadcast("WORKSPACE_CHANGED", map[string]interface{}{
				"id":        ws.ID,
				"name":      ws.Name,
				"path":      ws.Path,
				"type":      ws.Project.Type,
				"framework": ws.Project.Framework,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ws)
	})

	// Workspace configuration endpoint (GET & POST)
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var newCfg config.Config
			if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
				http.Error(w, "Invalid configuration JSON", http.StatusBadRequest)
				return
			}
			if err := s.configMgr.Update(newCfg); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if s.hub != nil {
				s.hub.Broadcast("CONFIG_UPDATED", s.configMgr.Get())
			}
			_ = json.NewEncoder(w).Encode(s.configMgr.Get())
			return
		}
		_ = json.NewEncoder(w).Encode(s.configMgr.Get())
	})

	// Local-only shutdown endpoint for CLI 'catalyst stop'
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		// Enforce loopback security: only localhost can trigger shutdown
		remoteHost := r.RemoteAddr
		if !strings.HasPrefix(remoteHost, "127.0.0.1") &&
			!strings.HasPrefix(remoteHost, "[::1]") &&
			!strings.HasPrefix(remoteHost, "localhost") {
			http.Error(w, "Forbidden: shutdown is restricted to localhost", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})

		// Trigger shutdown asynchronously so response can be delivered first
		go func() {
			time.Sleep(100 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = s.Shutdown(ctx)
		}()
	})

	// Pairing status and connected devices endpoint
	mux.HandleFunc("/api/pairing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]interface{}{
			"pin":          s.pairingMgr.PIN(),
			"devicesCount": s.pairingMgr.DeviceCount(),
			"devices":      s.pairingMgr.Devices(),
		}
		_ = json.NewEncoder(w).Encode(data)
	})

	// Pairing PIN verification endpoint
	mux.HandleFunc("/api/pairing/verify", func(w http.ResponseWriter, r *http.Request) {
		pin := strings.TrimSpace(r.URL.Query().Get("pin"))
		if pin == "" && r.Body != nil {
			var body struct {
				PIN string `json:"pin"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			pin = strings.TrimSpace(body.PIN)
		}

		token, ok := s.pairingMgr.AuthorizePIN(pin)
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid pairing code",
			})
			return
		}

		// Register device
		deviceID := fmt.Sprintf("dev-%d", time.Now().UnixNano()%10000)
		s.pairingMgr.RegisterDevice(deviceID, r.RemoteAddr, r.UserAgent())

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"token":   token,
			"device":  deviceID,
		})
	})

	// Pairing QR code SVG endpoint
	mux.HandleFunc("/api/pairing/qr", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		targetURL := fmt.Sprintf("http://%s/?pin=%s", r.Host, s.pairingMgr.PIN())
		svg := pairing.GenerateQRSVG(targetURL)
		_, _ = w.Write([]byte(svg))
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

	// Terminate all supervised child processes to prevent orphans
	if s.processMgr != nil {
		s.processMgr.StopAll()
	}

	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
