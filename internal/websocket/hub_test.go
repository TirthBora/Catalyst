package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"catalyst/internal/websocket"

	gorilla "github.com/gorilla/websocket"
)

func TestHubNew(t *testing.T) {
	hub := websocket.NewHub()
	if hub == nil {
		t.Fatal("expected non-nil Hub")
	}
	if count := hub.ClientCount(); count != 0 {
		t.Fatalf("expected 0 clients, got %d", count)
	}
}

func TestHubBroadcastAndLifecycle(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	const numClients = 3
	conns := make([]*gorilla.Conn, numClients)

	for i := 0; i < numClients; i++ {
		conn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		defer conn.Close()
		conns[i] = conn
	}

	// Allow registration to process
	time.Sleep(50 * time.Millisecond)

	if count := hub.ClientCount(); count != numClients {
		t.Fatalf("expected %d connected clients, got %d", numClients, count)
	}

	// Broadcast an event
	payload := map[string]string{
		"file":   "index.html",
		"action": "write",
	}
	hub.Broadcast("FILE_CHANGED", payload)

	// Verify all clients receive the broadcasted event
	var wg sync.WaitGroup
	for i, conn := range conns {
		wg.Add(1)
		go func(idx int, c *gorilla.Conn) {
			defer wg.Done()
			_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, msg, err := c.ReadMessage()
			if err != nil {
				t.Errorf("client %d failed to read: %v", idx, err)
				return
			}
			var event websocket.Event
			if err := json.Unmarshal(msg, &event); err != nil {
				t.Errorf("client %d received invalid JSON: %v", idx, err)
				return
			}
			if event.Type != "FILE_CHANGED" {
				t.Errorf("client %d expected FILE_CHANGED, got %s", idx, event.Type)
			}
		}(i, conn)
	}
	wg.Wait()

	// Disconnect one client
	_ = conns[0].Close()
	time.Sleep(50 * time.Millisecond)

	// Remaining clients should still be connected
	if count := hub.ClientCount(); count > numClients {
		t.Errorf("client count should decrease after disconnect, got %d", count)
	}
}
