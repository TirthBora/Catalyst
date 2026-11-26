package pairing_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"catalyst/internal/pairing"
)

func TestPINAndAuthorization(t *testing.T) {
	mgr := pairing.NewManager()
	pin := mgr.PIN()

	if len(pin) != 6 {
		t.Errorf("expected 6-character PIN, got %s (len %d)", pin, len(pin))
	}

	// Reject invalid PIN
	_, ok := mgr.AuthorizePIN("WRONG0")
	if ok {
		t.Error("expected invalid PIN to be rejected")
	}

	// Accept valid PIN
	token, ok := mgr.AuthorizePIN(pin)
	if !ok {
		t.Fatal("expected valid PIN to be authorized")
	}
	if token == "" {
		t.Fatal("expected non-empty session token")
	}

	// Validate token
	if !mgr.IsAuthorized(token) {
		t.Error("expected token to be recognized as authorized")
	}

	if mgr.IsAuthorized("fake-token-123") {
		t.Error("unauthorized token was accepted")
	}

	// Test PIN rotation
	newPin := mgr.RotatePIN()
	if newPin == "" {
		t.Error("expected non-empty rotated PIN")
	}
}

func TestDeviceRegistry(t *testing.T) {
	mgr := pairing.NewManager()

	if mgr.DeviceCount() != 0 {
		t.Errorf("expected 0 devices initially, got %d", mgr.DeviceCount())
	}

	dev := mgr.RegisterDevice("dev-1", "192.168.1.105", "iPad / Safari")
	if dev.ID != "dev-1" {
		t.Errorf("expected dev-1, got %s", dev.ID)
	}

	if mgr.DeviceCount() != 1 {
		t.Errorf("expected 1 device, got %d", mgr.DeviceCount())
	}

	devices := mgr.Devices()
	if len(devices) != 1 || devices[0].UserAgent != "iPad / Safari" {
		t.Errorf("expected registered device in list, got %+v", devices)
	}

	// Update existing device
	mgr.RegisterDevice("dev-1", "192.168.1.106", "iPad / Safari")
	if mgr.DeviceCount() != 1 {
		t.Errorf("expected device count to remain 1 on update, got %d", mgr.DeviceCount())
	}

	// Unregister
	mgr.UnregisterDevice("dev-1")
	if mgr.DeviceCount() != 0 {
		t.Errorf("expected 0 devices after unregister, got %d", mgr.DeviceCount())
	}
}

func TestExtractToken(t *testing.T) {
	// Bearer header
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.Header.Set("Authorization", "Bearer my-secret-token")
	if tok := pairing.ExtractToken(req1); tok != "my-secret-token" {
		t.Errorf("expected my-secret-token, got %s", tok)
	}

	// URL query
	req2 := httptest.NewRequest("GET", "/api/test?token=query-token-456", nil)
	if tok := pairing.ExtractToken(req2); tok != "query-token-456" {
		t.Errorf("expected query-token-456, got %s", tok)
	}
}

func TestGenerateQRSVG(t *testing.T) {
	svg := pairing.GenerateQRSVG("http://192.168.1.100:9999")
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Errorf("expected valid SVG graphic, got %s", svg)
	}
}
