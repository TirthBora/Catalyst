package proxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"catalyst/internal/proxy"
)

func TestProxyValidation(t *testing.T) {
	p := proxy.NewReverseProxy()

	// Missing port
	req := httptest.NewRequest(http.MethodGet, "/proxy", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty port, got %d", rec.Code)
	}

	// Invalid port
	req = httptest.NewRequest(http.MethodGet, "/proxy/abc/", nil)
	rec = httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric port, got %d", rec.Code)
	}

	// Out of range port
	req = httptest.NewRequest(http.MethodGet, "/proxy/99999/", nil)
	rec = httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for out-of-range port, got %d", rec.Code)
	}
}

func TestProxyForwarding(t *testing.T) {
	// Spin up a mock development server
	mockDevServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintf(w, "Hello from dev server! Path: %s, Host: %s", r.URL.Path, r.Host)
	}))
	defer mockDevServer.Close()

	u, err := url.Parse(mockDevServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	port := u.Port()

	p := proxy.NewReverseProxy()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/proxy/%s/dashboard?filter=all", port), nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Hello from dev server!") {
		t.Errorf("expected response from dev server, got %s", body)
	}
	if !strings.Contains(body, "/dashboard") {
		t.Errorf("expected forwarded path /dashboard, got %s", body)
	}
}

func TestProxyUnreachable(t *testing.T) {
	p := proxy.NewReverseProxy()

	// Use a port where nothing is listening
	req := httptest.NewRequest(http.MethodGet, "/proxy/59998/api", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502 Bad Gateway, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Catalyst Proxy Error") {
		t.Errorf("expected error details in response, got %s", body)
	}
}
