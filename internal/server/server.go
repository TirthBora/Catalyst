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

	"catalyst/internal/assistant"
	"catalyst/internal/config"
	"catalyst/internal/monitor"
	"catalyst/internal/pairing"
	"catalyst/internal/plugin"
	"catalyst/internal/process"
	"catalyst/internal/project"
	"catalyst/internal/proxy"
	"catalyst/internal/relay"
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
	devRunner    *process.DevRunner
	pairingMgr   *pairing.Manager
	configMgr    *config.Manager
	workspaceReg *workspace.Registry
	watcher      *watcher.Watcher
	relayClient  *relay.Client
	pluginMgr    *plugin.Manager
	httpServer   *http.Server
	mu           sync.Mutex
}

func NewServer(port string, hub *websocket.Hub, projectRoot string, pm *process.Manager, pairingMgr *pairing.Manager, cfgMgr *config.Manager, wsReg *workspace.Registry, w *watcher.Watcher, rClient *relay.Client, plugMgr *plugin.Manager) *Server {
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
	if plugMgr == nil {
		plugMgr = plugin.NewManager(projectRoot)
	}
	return &Server{
		hub:          hub,
		port:         port,
		projectRoot:  projectRoot,
		projectInfo:  project.Detect(projectRoot),
		processMgr:   pm,
		devRunner:    process.NewDevRunner(pm),
		pairingMgr:   pairingMgr,
		configMgr:    cfgMgr,
		workspaceReg: wsReg,
		watcher:      w,
		relayClient:  rClient,
		pluginMgr:    plugMgr,
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

	// Cloud relay WAN status and connection endpoints
	mux.HandleFunc("/api/relay", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s.mu.Lock()
		client := s.relayClient
		s.mu.Unlock()

		if client == nil {
			_ = json.NewEncoder(w).Encode(relay.Status{
				State: relay.StateDisconnected,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(client.Status())
	})

	mux.HandleFunc("/api/relay/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RelayURL string `json:"relayUrl"`
			Token    string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.RelayURL) == "" {
			http.Error(w, "Relay URL required", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if s.relayClient != nil {
			s.relayClient.Close()
		}
		s.relayClient = relay.NewClient(body.RelayURL, body.Token)
		client := s.relayClient
		s.mu.Unlock()

		go func() {
			_ = client.Connect()
		}()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.Status())
	})

	// Dev server execution endpoints
	mux.HandleFunc("/api/dev/scripts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status, running := s.devRunner.GetStatus()
		data := map[string]interface{}{
			"scripts": s.projectInfo.Scripts,
			"running": running,
			"process": status,
		}
		_ = json.NewEncoder(w).Encode(data)
	})

	mux.HandleFunc("/api/dev/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Script string `json:"script"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Script) == "" {
			http.Error(w, "Script name required", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		allowed := s.projectInfo.Scripts
		workDir := s.projectRoot
		s.mu.Unlock()

		p, err := s.devRunner.StartScript(body.Script, allowed, workDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if s.hub != nil {
			s.hub.Broadcast("DEV_SERVER_STATE", map[string]interface{}{
				"running": true,
				"pid":     p.PID,
				"script":  body.Script,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("/api/dev/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = s.devRunner.Stop()
		if s.hub != nil {
			s.hub.Broadcast("DEV_SERVER_STATE", map[string]interface{}{
				"running": false,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
	})

	mux.HandleFunc("/api/dev/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p, err := s.devRunner.Restart()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.hub != nil {
			s.hub.Broadcast("DEV_SERVER_STATE", map[string]interface{}{
				"running": true,
				"pid":     p.PID,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("/api/dev/input", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid input payload", http.StatusBadRequest)
			return
		}

		if err := s.devRunner.WriteInput(body.Text); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if s.hub != nil {
			s.hub.Broadcast("PROCESS_OUTPUT", map[string]interface{}{
				"stream":    "stdin",
				"text":      body.Text,
				"timestamp": time.Now().Format("15:04:05"),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	})

	// Assistant diagnostic and proactive insight endpoints
	mux.HandleFunc("/api/assistant/insights", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s.mu.Lock()
		root := s.projectRoot
		info := s.projectInfo
		s.mu.Unlock()

		insights := assistant.EvaluateWorkspace(root, info)
		_ = json.NewEncoder(w).Encode(insights)
	})

	mux.HandleFunc("/api/assistant/analyze", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Logs []string `json:"logs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid analyze payload", http.StatusBadRequest)
			return
		}

		analysis := assistant.AnalyzeLogs(body.Logs)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(analysis)
	})

	// Plugin and custom action shortcuts endpoints
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s.mu.Lock()
		pm := s.pluginMgr
		s.mu.Unlock()

		data := map[string]interface{}{
			"plugins": pm.Plugins(),
			"actions": pm.Actions(),
		}
		_ = json.NewEncoder(w).Encode(data)
	})

	mux.HandleFunc("/api/plugins/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			ActionID string `json:"actionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ActionID) == "" {
			http.Error(w, "Action ID required", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		pm := s.pluginMgr
		workDir := s.projectRoot
		s.mu.Unlock()

		act, err := pm.GetAction(body.ActionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		p, err := s.processMgr.Start("plugin-"+act.ID, act.Command, act.Args, workDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
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
		s.pluginMgr = plugin.NewManager(ws.Path)

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

	// Close WAN relay tunnel if active
	if s.relayClient != nil {
		s.relayClient.Close()
	}

	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
