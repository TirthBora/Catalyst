package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"catalyst/internal/cli"
	"catalyst/internal/config"
	"catalyst/internal/monitor"
	"catalyst/internal/pairing"
	"catalyst/internal/process"
	"catalyst/internal/relay"
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
	opts, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Printf("Error: %v\n\n%s", err, cli.HelpText())
		os.Exit(1)
	}

	switch opts.Action {
	case cli.ActionHelp:
		fmt.Print(cli.HelpText())
		return

	case cli.ActionVersion:
		fmt.Println(cli.FormatVersion())
		return

	case cli.ActionStatus:
		res := cli.CheckStatus(opts.Port)
		fmt.Println(cli.FormatStatus(res))
		return

	case cli.ActionStop:
		if err := cli.StopDaemon(opts.Port); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Catalyst daemon on port %s stopped successfully.\n", opts.Port)
		return

	case cli.ActionDoctor:
		report := runtime.Diagnose(opts.Dir)
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

	port := opts.Port
	dir := opts.Dir

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
				if alerts := monitor.CheckAlerts(stats, 85.0); len(alerts) > 0 {
					hub.Broadcast("SYSTEM_ALERT", alerts[0])
				}
			}
		}
	}()

	// 2. Boot the Filesystem Watcher
	w, err := watcher.NewWatcher(dir, hub)
	if err != nil {
		log.Fatalf("Failed to initialize watcher: %v", err)
	}
	defer w.Close()

	// Load workspace configuration
	cfgMgr := config.NewManager(dir)
	if custom := cfgMgr.Get().CustomIgnore; len(custom) > 0 {
		w.SetCustomIgnore(custom)
	}

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}

	localIP := getLocalIP()
	pairingMgr := pairing.NewManager()

	var rClient *relay.Client
	if opts.RelayURL != "" {
		rClient = relay.NewClient(opts.RelayURL, opts.RelayToken)
		go func() {
			_ = rClient.Connect()
		}()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  CATALYST V1 IS ACTIVE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Watching : %s\n", dir)
	fmt.Printf("  Connect  : http://%s:%s\n", localIP, port)
	fmt.Printf("  Pair PIN : %s\n", pairingMgr.PIN())
	if opts.RelayURL != "" {
		fmt.Printf("  WAN Relay: %s\n", opts.RelayURL)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 3. Boot the Process Manager
	pm := process.NewManager(hub)

	// 4. Boot the HTTP Server
	srv := server.NewServer(port, hub, dir, pm, pairingMgr, cfgMgr, nil, w, rClient)

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
