package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

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
	port := "9999"
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// 1. Boot the WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 2. Boot the Filesystem Watcher
	w, err := watcher.NewWatcher(cwd, hub)
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
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Watching: %s\n", cwd)
	fmt.Printf("  Open on your Secondary Device: http://%s:%s\n", localIP, port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 3. Boot the HTTP Server
	srv := server.NewServer(port, hub)
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server failed to start: %v\n", err)
	}
}
