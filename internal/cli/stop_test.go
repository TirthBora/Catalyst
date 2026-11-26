package cli_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"catalyst/internal/cli"
)

func TestStopDaemonUnreachable(t *testing.T) {
	err := cli.StopDaemon("59992")
	if err == nil {
		t.Error("expected error when stopping daemon on closed port")
	}
}

func TestStopDaemonSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/shutdown" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"shutting_down"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.StopDaemon(u.Port()); err != nil {
		t.Errorf("expected clean stop, got: %v", err)
	}
}

func TestStopDaemonForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.StopDaemon(u.Port()); err == nil {
		t.Error("expected error when server responds with 403 Forbidden")
	}
}
