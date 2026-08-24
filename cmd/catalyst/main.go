package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// getLocalIP finds your machine's LAN IP,filtering out invalid virtual ones
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
	localIP := getLocalIP()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(" CATALYST V1 IS ACTIVE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━═══════════════════════════════")
	fmt.Printf("  Open on your Secondary Device: http://%s:%s\n", localIP, port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	// A simple route to prove the connection works
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Catalyst Workspace Connected</h1><p>Waiting for code changes...")
	})
	fmt.Printf("Starting server on port %s ...\n", port)
	//Start the server and catch any errors
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
	// log.Fatal(http.ListenAndServe(":"+port, nil))
}
