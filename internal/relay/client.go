package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// State represents the lifecycle of the relay tunnel.
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
)

// Status contains the current WAN relay connection state.
type Status struct {
	State     State  `json:"state"`
	RelayURL  string `json:"relayUrl,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Client manages an outbound WebSocket tunnel to a CatalystRelay server.
type Client struct {
	relayURL  string
	token     string
	sessionID string
	state     State
	lastErr   string
	conn      *websocket.Conn
	writeMu   sync.Mutex
	mu        sync.RWMutex
	stopChan  chan struct{}
}

func NewClient(relayURL, token string) *Client {
	return &Client{
		relayURL: relayURL,
		token:    token,
		state:    StateDisconnected,
		stopChan: make(chan struct{}),
	}
}

// Status returns current WAN relay connectivity details.
func (c *Client) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Status{
		State:     c.state,
		RelayURL:  c.relayURL,
		SessionID: c.sessionID,
		Error:     c.lastErr,
	}
}

// Connect dials the relay server and creates a primary WAN session.
func (c *Client) Connect() error {
	c.mu.Lock()
	if c.state == StateConnected {
		c.mu.Unlock()
		return nil
	}
	c.state = StateConnecting
	c.lastErr = ""
	c.mu.Unlock()

	headers := http.Header{}
	if c.token != "" {
		headers.Set("Authorization", "Bearer "+c.token)
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second

	conn, _, err := dialer.Dial(c.relayURL, headers)
	if err != nil {
		c.mu.Lock()
		c.state = StateDisconnected
		c.lastErr = err.Error()
		c.mu.Unlock()
		return fmt.Errorf("relay dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Send CREATE_SESSION request
	createMsg := NewMessage(TypeCreateSession)
	createMsg.Sender = "primary"

	c.writeMu.Lock()
	err = conn.WriteJSON(createMsg)
	c.writeMu.Unlock()

	if err != nil {
		_ = conn.Close()
		c.mu.Lock()
		c.state = StateDisconnected
		c.lastErr = err.Error()
		c.mu.Unlock()
		return fmt.Errorf("failed to send CREATE_SESSION: %w", err)
	}

	// Read SESSION_CREATED response
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var resp Message
	if err := conn.ReadJSON(&resp); err != nil {
		_ = conn.Close()
		c.mu.Lock()
		c.state = StateDisconnected
		c.lastErr = err.Error()
		c.mu.Unlock()
		return fmt.Errorf("failed to read session confirmation: %w", err)
	}

	if resp.Type != TypeSessionCreated || resp.Session == "" {
		_ = conn.Close()
		c.mu.Lock()
		c.state = StateDisconnected
		c.lastErr = "unexpected relay response"
		c.mu.Unlock()
		return fmt.Errorf("relay returned invalid response type: %s", resp.Type)
	}

	c.mu.Lock()
	c.sessionID = resp.Session
	c.state = StateConnected
	c.lastErr = ""
	c.mu.Unlock()

	// Start pump to handle incoming messages and pings
	go c.readPump()

	return nil
}

func (c *Client) readPump() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.state = StateDisconnected
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()
			if conn == nil {
				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}
}

// SendFileChanged forwards file modifications across the WAN relay to connected secondary clients.
func (c *Client) SendFileChanged(file, action string) error {
	c.mu.RLock()
	if c.state != StateConnected || c.conn == nil {
		c.mu.RUnlock()
		return nil // Quietly ignore if WAN relay is not active
	}
	sessionID := c.sessionID
	c.mu.RUnlock()

	payloadData, err := json.Marshal(map[string]string{
		"file":   file,
		"action": action,
	})
	if err != nil {
		return err
	}

	msg := NewMessage(TypeFileChanged)
	msg.Session = sessionID
	msg.Sender = "primary"
	msg.Payload = payloadData

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("relay connection is closed")
	}

	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteJSON(msg)
}

// Close terminates the relay connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.state = StateDisconnected
}
