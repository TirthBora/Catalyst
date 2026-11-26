package server_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	srv := server.NewServer("9999", hub, tempDir, nil, nil, nil, nil, nil)

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

	// 8. Test Pairing Status API (GET /api/pairing)
	pairReq := httptest.NewRequest(http.MethodGet, "/api/pairing", nil)
	pairRec := httptest.NewRecorder()
	handler.ServeHTTP(pairRec, pairReq)

	if pairRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/pairing, got %d", pairRec.Code)
	}
	pairBody := pairRec.Body.String()
	if !strings.Contains(pairBody, "pin") {
		t.Errorf("expected pin field in /api/pairing, got: %s", pairBody)
	}

	// 9. Test Pairing Verification API (POST /api/pairing/verify)
	verifyBadReq := httptest.NewRequest(http.MethodPost, "/api/pairing/verify?pin=INVALID", nil)
	verifyBadRec := httptest.NewRecorder()
	handler.ServeHTTP(verifyBadRec, verifyBadReq)

	if verifyBadRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid pin, got %d", verifyBadRec.Code)
	}

	// 10. Test Pairing QR SVG API (GET /api/pairing/qr)
	qrReq := httptest.NewRequest(http.MethodGet, "/api/pairing/qr", nil)
	qrRec := httptest.NewRecorder()
	handler.ServeHTTP(qrRec, qrReq)

	if qrRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /api/pairing/qr, got %d", qrRec.Code)
	}
	if !strings.Contains(qrRec.Body.String(), "<svg") {
		t.Errorf("expected SVG output, got: %s", qrRec.Body.String())
	}

	// 11. Test Config Get API (GET /api/config)
	cfgReq := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	cfgRec := httptest.NewRecorder()
	handler.ServeHTTP(cfgRec, cfgReq)

	if cfgRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET /api/config, got %d", cfgRec.Code)
	}
	if !strings.Contains(cfgRec.Body.String(), "targetUrl") {
		t.Errorf("expected JSON with targetUrl in /api/config, got: %s", cfgRec.Body.String())
	}

	// 12. Test Config Update API (POST /api/config)
	postBody := strings.NewReader(`{"targetUrl": "/proxy/5173/", "defaultDevice": "tablet"}`)
	cfgPostReq := httptest.NewRequest(http.MethodPost, "/api/config", postBody)
	cfgPostRec := httptest.NewRecorder()
	handler.ServeHTTP(cfgPostRec, cfgPostReq)

	if cfgPostRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST /api/config, got %d", cfgPostRec.Code)
	}
	if !strings.Contains(cfgPostRec.Body.String(), "/proxy/5173/") {
		t.Errorf("expected updated target in response, got: %s", cfgPostRec.Body.String())
	}

	// 13. Test Workspace List API (GET /api/workspaces)
	wsListReq := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)
	wsListRec := httptest.NewRecorder()
	handler.ServeHTTP(wsListRec, wsListReq)

	if wsListRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET /api/workspaces, got %d", wsListRec.Code)
	}
	if !strings.Contains(wsListRec.Body.String(), filepath.Base(tempDir)) {
		t.Errorf("expected initial workspace in list, got: %s", wsListRec.Body.String())
	}

	// 14. Test Workspace Register API (POST /api/workspaces)
	dir2 := t.TempDir()
	wsPostReq := httptest.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader(fmt.Sprintf(`{"path": %q}`, dir2)))
	wsPostRec := httptest.NewRecorder()
	handler.ServeHTTP(wsPostRec, wsPostReq)

	if wsPostRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for POST /api/workspaces, got %d", wsPostRec.Code)
	}

	var newWS struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(wsPostRec.Body.Bytes(), &newWS)

	// 15. Test Workspace Switch API (POST /api/workspaces/switch)
	switchReq := httptest.NewRequest(http.MethodPost, "/api/workspaces/switch", strings.NewReader(fmt.Sprintf(`{"id": %q}`, newWS.ID)))
	switchRec := httptest.NewRecorder()
	handler.ServeHTTP(switchRec, switchReq)

	if switchRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST /api/workspaces/switch, got %d", switchRec.Code)
	}
	if !strings.Contains(switchRec.Body.String(), newWS.ID) {
		t.Errorf("expected switched workspace ID in response, got: %s", switchRec.Body.String())
	}

	// 16. Test Dev Scripts Listing API (GET /api/dev/scripts)
	devScriptsReq := httptest.NewRequest(http.MethodGet, "/api/dev/scripts", nil)
	devScriptsRec := httptest.NewRecorder()
	handler.ServeHTTP(devScriptsRec, devScriptsReq)

	if devScriptsRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET /api/dev/scripts, got %d", devScriptsRec.Code)
	}

	// 17. Test Dev Start with unvetted script (POST /api/dev/start) -> should reject
	devStartReq := httptest.NewRequest(http.MethodPost, "/api/dev/start", strings.NewReader(`{"script": "arbitrary-injection"}`))
	devStartRec := httptest.NewRecorder()
	handler.ServeHTTP(devStartRec, devStartReq)

	if devStartRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unvetted script name, got %d", devStartRec.Code)
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
	srv := server.NewServer(port, hub, tempDir, nil, nil, nil, nil, nil)

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
