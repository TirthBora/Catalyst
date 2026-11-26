package relay_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"catalyst/internal/relay"

	gorilla "github.com/gorilla/websocket"
)

var upgrader = gorilla.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestClientDisconnected(t *testing.T) {
	c := relay.NewClient("ws://localhost:9999", "token")
	status := c.Status()
	if status.State != relay.StateDisconnected {
		t.Errorf("expected disconnected, got %s", status.State)
	}
}

func TestClientConnectAndForward(t *testing.T) {
	// Spin up a mock CatalystRelay server
	receivedMessages := make(chan relay.Message, 5)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade mock relay: %v", err)
			return
		}
		defer conn.Close()

		for {
			var msg relay.Message
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			receivedMessages <- msg

			if msg.Type == relay.TypeCreateSession {
				resp := relay.NewMessage(relay.TypeSessionCreated)
				resp.Session = "WAN8888"
				resp.Sender = "relay"
				_ = conn.WriteJSON(resp)
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	c := relay.NewClient(wsURL, "test-secret")
	if err := c.Connect(); err != nil {
		t.Fatalf("client failed to connect to mock relay: %v", err)
	}
	defer c.Close()

	status := c.Status()
	if status.State != relay.StateConnected {
		t.Errorf("expected connected state, got %s", status.State)
	}
	if status.SessionID != "WAN8888" {
		t.Errorf("expected session ID WAN8888, got %s", status.SessionID)
	}

	// Test forwarding file changed event
	if err := c.SendFileChanged("index.html", "write"); err != nil {
		t.Fatalf("failed to send file changed: %v", err)
	}

	select {
	case msg := <-receivedMessages:
		if msg.Type != relay.TypeCreateSession {
			t.Errorf("expected CREATE_SESSION first, got %s", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for CREATE_SESSION")
	}

	select {
	case msg := <-receivedMessages:
		if msg.Type != relay.TypeFileChanged {
			t.Errorf("expected FILE_CHANGED event, got %s", msg.Type)
		}
		if msg.Session != "WAN8888" {
			t.Errorf("expected session WAN8888 on forwarded event, got %s", msg.Session)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for FILE_CHANGED")
	}

	// Close client
	c.Close()
	if c.Status().State != relay.StateDisconnected {
		t.Error("expected disconnected after close")
	}
}
