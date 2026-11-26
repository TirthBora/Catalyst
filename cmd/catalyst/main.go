package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"catalyst/internal/monitor"
	"catalyst/internal/process"
	"catalyst/internal/runtime"
	"catalyst/internal/server"
	"catalyst/internal/watcher"
	"catalyst/internal/websocket"
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				if !strings.HasPrefix(ip, "169.254.") {
					return ip
				}
			}
		}
	}
	return "localhost"
}

func main() {
	defaultPort := os.Getenv("CATALYST_PORT")
	if defaultPort == "" {
		defaultPort = "9999"
	}

	defaultDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	var port string
	var dir string
	var doctor bool

	flag.StringVar(&port, "port", defaultPort, "HTTP and WebSocket server port")
	flag.StringVar(&port, "p", defaultPort, "HTTP and WebSocket server port (shorthand)")
	flag.StringVar(&dir, "dir", defaultDir, "Project root directory to watch and preview")
	flag.StringVar(&dir, "d", defaultDir, "Project root directory to watch and preview (shorthand)")
	flag.BoolVar(&doctor, "doctor", false, "Run environment and toolchain diagnostics")
	flag.Parse()

	// If -doctor flag is specified, run diagnostics and exit
	if doctor {
		report := runtime.Diagnose(dir)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  CATALYST ENVIRONMENT DIAGNOSTICS")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Toolchains:")
		for _, tc := range report.Toolchains {
			if tc.Installed {
				fmt.Printf("  ✓ %-10s : %s\n", tc.Name, tc.Version)
			} else {
				fmt.Printf("  ✗ %-10s : Not installed or not in PATH\n", tc.Name)
			}
		}
		fmt.Println("\nWorkspace Checks:")
		for _, c := range report.Checks {
			if c.Category == "workspace" {
				mark := "✓"
				if c.Status == "warning" {
					mark = "!"
				} else if c.Status == "error" {
					mark = "✗"
				}
				fmt.Printf("  %s %-18s : %s\n", mark, c.Name, c.Message)
			}
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return
	}

	// 1. Boot the WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// Periodic system stats broadcaster (every 5 seconds when clients are connected)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if hub.ClientCount() > 0 {
				stats := monitor.GetStats()
				hub.Broadcast("SYSTEM_STATS", stats)
			}
		}
	}()

	// 2. Boot the Filesystem Watcher
	w, err := watcher.NewWatcher(dir, hub)
	if err != nil {
		log.Fatalf("Failed to initialize watcher: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}

	localIP := getLocalIP()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  CATALYST V1 IS ACTIVE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Watching: %s\n", dir)
	fmt.Printf("  Open on your Secondary Device: http://%s:%s\n", localIP, port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 3. Boot the Process Manager
	pm := process.NewManager(hub)

	// 4. Boot the HTTP Server
	srv := server.NewServer(port, hub, dir, pm)

	// Intercept shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down Catalyst...")
		w.Close()
		pm.StopAll()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v\n", err)
		}
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server failed to start: %v\n", err)
	}

	fmt.Println("Catalyst stopped cleanly.")
}
