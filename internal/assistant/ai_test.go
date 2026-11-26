package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"catalyst/internal/assistant"
)

func TestDefaultAIConfig(t *testing.T) {
	cfg := assistant.DefaultAIConfig()
	if cfg.Endpoint == "" {
		t.Error("expected non-empty default AI endpoint")
	}
	if cfg.Model == "" {
		t.Error("expected non-empty default AI model")
	}
}

func TestExplainErrorOffline(t *testing.T) {
	cfg := assistant.AIConfig{
		Endpoint: "http://127.0.0.1:59993/v1/chat/completions",
		Model:    "test-model",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	res := assistant.ExplainError(ctx, cfg, "SyntaxError: Unexpected token", "Node.js")
	if res.Available {
		t.Error("expected Available to be false for offline endpoint")
	}
	if !strings.Contains(res.Error, "unreachable") {
		t.Errorf("expected unreachable error message, got: %s", res.Error)
	}
}

func TestExplainErrorMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "The module is missing. Run: npm install express",
					},
				},
			},
		})
	}))
	defer server.Close()

	cfg := assistant.AIConfig{
		Endpoint: server.URL,
		Model:    "test-model",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res := assistant.ExplainError(ctx, cfg, "Cannot find module 'express'", "Node.js")
	if !res.Available {
		t.Fatalf("expected Available true, got error: %s", res.Error)
	}

	if !strings.Contains(res.SuggestedFix, "npm install express") {
		t.Errorf("expected suggested fix content, got: %s", res.SuggestedFix)
	}
}
