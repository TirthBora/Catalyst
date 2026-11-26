package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"catalyst/internal/server"
	"catalyst/internal/websocket"
)

func TestServerHandler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "catalyst-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a sample index.html in the project root
	projectHTML := filepath.Join(tempDir, "index.html")
	if err := os.WriteFile(projectHTML, []byte("<h1>My Test Project</h1>"), 0644); err != nil {
		t.Fatalf("failed to create project HTML: %v", err)
	}

	hub := websocket.NewHub()
	srv := server.NewServer("9999", hub, tempDir, nil)

	handler, err := srv.Handler()
	if err != nil {
		t.Fatalf("failed to get handler: %v", err)
	}

	// 1. Test Dashboard UI route (GET /)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for dashboard, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "CATALYST") {
		t.Errorf("dashboard HTML expected to contain 'CATALYST', got: %s", body)
	}

	// 2. Test Project Preview route (GET /preview/)
	previewReq := httptest.NewRequest(http.MethodGet, "/preview/", nil)
	previewRec := httptest.NewRecorder()
	handler.ServeHTTP(previewRec, previewReq)

	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for preview, got %d", previewRec.Code)
	}

	previewBody := previewRec.Body.String()
	if !strings.Contains(previewBody, "My Test Project") {
		t.Errorf("preview HTML expected to contain 'My Test Project', got: %s", previewBody)
	}

	// 3. Test Proxy route registration (GET /proxy/)
	proxyReq := httptest.NewRequest(http.MethodGet, "/proxy/", nil)
	proxyRec := httptest.NewRecorder()
	handler.ServeHTTP(proxyRec, proxyReq)

	if proxyRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for proxy without port, got %d", proxyRec.Code)
	}

	// 4. Test Workspace Metadata API (GET /api/workspace)
	apiReq := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	apiRec := httptest.NewRecorder()
	handler.ServeHTTP(apiRec, apiReq)

	if apiRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/workspace, got %d", apiRec.Code)
	}
	apiBody := apiRec.Body.String()
	if !strings.Contains(apiBody, "Static Web") {
		t.Errorf("expected detected type 'Static Web' in /api/workspace, got: %s", apiBody)
	}

	// 5. Test System Metrics API (GET /api/system)
	sysReq := httptest.NewRequest(http.MethodGet, "/api/system", nil)
	sysRec := httptest.NewRecorder()
	handler.ServeHTTP(sysRec, sysReq)

	if sysRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/system, got %d", sysRec.Code)
	}
	sysBody := sysRec.Body.String()
	if !strings.Contains(sysBody, "numCpu") {
		t.Errorf("expected JSON with numCpu in /api/system, got: %s", sysBody)
	}

	// 6. Test Doctor Diagnostics API (GET /api/doctor)
	docReq := httptest.NewRequest(http.MethodGet, "/api/doctor", nil)
	docRec := httptest.NewRecorder()
	handler.ServeHTTP(docRec, docReq)

	if docRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/doctor, got %d", docRec.Code)
	}
	docBody := docRec.Body.String()
	if !strings.Contains(docBody, "toolchains") {
		t.Errorf("expected JSON with toolchains in /api/doctor, got: %s", docBody)
	}

	// 7. Test Process Monitoring API (GET /api/processes)
	procReq := httptest.NewRequest(http.MethodGet, "/api/processes", nil)
	procRec := httptest.NewRecorder()
	handler.ServeHTTP(procRec, procReq)

	if procRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/processes, got %d", procRec.Code)
	}
	procBody := procRec.Body.String()
	if !strings.HasPrefix(strings.TrimSpace(procBody), "[") {
		t.Errorf("expected JSON array in /api/processes, got: %s", procBody)
	}
}

func TestServerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "catalyst-server-lifecycle-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	// Use an open high-numbered port for testing
	port := "19998"
	srv := server.NewServer(port, hub, tempDir, nil)

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to bind
	time.Sleep(100 * time.Millisecond)

	// Ping server
	resp, err := http.Get("http://localhost:" + port + "/")
	if err != nil {
		t.Fatalf("failed to reach server: %v", err)
	}
	defer resp.Body.Close()

	content, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(content), "CATALYST") {
		t.Errorf("expected dashboard content, got: %s", string(content))
	}

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("expected clean shutdown, got error: %v", err)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("server exited with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for server to stop")
	}
}
