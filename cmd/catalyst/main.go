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

	flag.StringVar(&port, "port", defaultPort, "HTTP and WebSocket server port")
	flag.StringVar(&port, "p", defaultPort, "HTTP and WebSocket server port (shorthand)")
	flag.StringVar(&dir, "dir", defaultDir, "Project root directory to watch and preview")
	flag.StringVar(&dir, "d", defaultDir, "Project root directory to watch and preview (shorthand)")
	flag.Parse()

	// 1. Boot the WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

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

	// 3. Boot the HTTP Server
	srv := server.NewServer(port, hub, dir)

	// Intercept shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down Catalyst...")
		w.Close()

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
