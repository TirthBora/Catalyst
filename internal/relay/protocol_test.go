package relay_test

import (
	"encoding/json"
	"testing"

	"catalyst/internal/relay"
)

func TestNewMessage(t *testing.T) {
	msg := relay.NewMessage(relay.TypeCreateSession)

	if msg.Version != relay.Version {
		t.Errorf("expected version %d, got %d", relay.Version, msg.Version)
	}

	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}

	if msg.Type != relay.TypeCreateSession {
		t.Errorf("expected type %s, got %s", relay.TypeCreateSession, msg.Type)
	}

	if msg.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestNewMessageWithPayload(t *testing.T) {
	payload := map[string]string{
		"file":   "main.go",
		"action": "write",
	}

	msg, err := relay.NewMessageWithPayload(relay.TypeFileChanged, payload)
	if err != nil {
		t.Fatalf("failed to create message with payload: %v", err)
	}

	var extracted map[string]string
	if err := json.Unmarshal(msg.Payload, &extracted); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if extracted["file"] != "main.go" || extracted["action"] != "write" {
		t.Errorf("unexpected extracted payload: %+v", extracted)
	}
}
