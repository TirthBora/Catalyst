package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Version is the current Catalyst Relay Protocol specification version.
const Version = 1

// Relay message types.
const (
	TypeCreateSession  = "CREATE_SESSION"
	TypeSessionCreated = "SESSION_CREATED"
	TypeJoinSession    = "JOIN_SESSION"
	TypeSessionJoined  = "SESSION_JOINED"
	TypeFileChanged    = "FILE_CHANGED"
	TypeError          = "ERROR"
)

// Message is the standardized versioned communication envelope.
type Message struct {
	ID        string          `json:"id"`
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	Session   string          `json:"session,omitempty"`
	Sender    string          `json:"sender,omitempty"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func generateMessageID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewMessage creates a new initialized protocol message envelope.
func NewMessage(msgType string) Message {
	return Message{
		ID:        generateMessageID(),
		Version:   Version,
		Type:      msgType,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

// NewMessageWithPayload creates a protocol message with JSON payload.
func NewMessageWithPayload(msgType string, payload interface{}) (Message, error) {
	msg := NewMessage(msgType)
	data, err := json.Marshal(payload)
	if err != nil {
		return msg, err
	}
	msg.Payload = data
	return msg, nil
}
