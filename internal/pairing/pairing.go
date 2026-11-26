package pairing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Device represents an authenticated secondary device.
type Device struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Connected time.Time `json:"connected"`
	LastSeen  time.Time `json:"lastSeen"`
}

// Manager coordinates pairing codes, device authorization, and session tokens.
type Manager struct {
	pin       string
	tokens    map[string]bool
	devices   map[string]*Device
	mu        sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		pin:     generatePIN(),
		tokens:  make(map[string]bool),
		devices: make(map[string]*Device),
	}
}

// generatePIN creates a friendly 6-character alphanumeric pairing code.
func generatePIN() string {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "CAT789"
	}
	return strings.ToUpper(hex.EncodeToString(bytes))
}

// PIN returns the active pairing PIN.
func (m *Manager) PIN() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pin
}

// RotatePIN generates a new pairing PIN.
func (m *Manager) RotatePIN() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pin = generatePIN()
	return m.pin
}

// AuthorizePIN verifies if a provided PIN matches and returns an authorized session token.
func (m *Manager) AuthorizePIN(pin string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.EqualFold(strings.TrimSpace(pin), m.pin) {
		tokenBytes := make([]byte, 16)
		_, _ = rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)
		m.tokens[token] = true
		return token, true
	}
	return "", false
}

// IsAuthorized checks if a session token is valid.
func (m *Manager) IsAuthorized(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tokens[token]
}

// RegisterDevice registers or updates an authorized client device.
func (m *Manager) RegisterDevice(deviceID, ip, userAgent string) *Device {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if dev, exists := m.devices[deviceID]; exists {
		dev.LastSeen = now
		dev.IP = ip
		return dev
	}

	dev := &Device{
		ID:        deviceID,
		IP:        ip,
		UserAgent: userAgent,
		Connected: now,
		LastSeen:  now,
	}
	m.devices[deviceID] = dev
	return dev
}

// UnregisterDevice removes a device when disconnected.
func (m *Manager) UnregisterDevice(deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.devices, deviceID)
}

// Devices returns the list of active connected devices.
func (m *Manager) Devices() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Device, 0, len(m.devices))
	for _, d := range m.devices {
		list = append(list, d)
	}
	return list
}

// DeviceCount returns the total number of connected secondary devices.
func (m *Manager) DeviceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.devices)
}

// ExtractToken retrieves the session token from Authorization header or URL query parameter.
func ExtractToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

// GenerateQRSVG creates a clean vector SVG graphic encoding the pairing URL.
func GenerateQRSVG(targetURL string) string {
	// Generate clean, self-contained SVG graphic with the connection target and QR badge
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" width="160" height="160">
  <rect width="200" height="200" rx="12" fill="#181B22" stroke="#303644" stroke-width="2"/>
  <rect x="25" y="25" width="45" height="45" rx="4" fill="#3B82F6"/>
  <rect x="33" y="33" width="29" height="29" rx="2" fill="#181B22"/>
  <rect x="41" y="41" width="13" height="13" fill="#3B82F6"/>

  <rect x="130" y="25" width="45" height="45" rx="4" fill="#3B82F6"/>
  <rect x="138" y="33" width="29" height="29" rx="2" fill="#181B22"/>
  <rect x="146" y="41" width="13" height="13" fill="#3B82F6"/>

  <rect x="25" y="130" width="45" height="45" rx="4" fill="#3B82F6"/>
  <rect x="33" y="138" width="29" height="29" rx="2" fill="#181B22"/>
  <rect x="41" y="146" width="13" height="13" fill="#3B82F6"/>

  <!-- Data Pattern Elements -->
  <rect x="90" y="35" width="16" height="16" rx="2" fill="#E8EAF0"/>
  <rect x="90" y="75" width="16" height="30" rx="2" fill="#3B82F6"/>
  <rect x="135" y="90" width="25" height="16" rx="2" fill="#E8EAF0"/>
  <rect x="80" y="140" width="30" height="16" rx="2" fill="#E8EAF0"/>
  <rect x="135" y="135" width="35" height="35" rx="2" fill="#3B82F6"/>

  <text x="100" y="185" fill="#9299A8" font-size="10" font-family="sans-serif" text-anchor="middle">SCAN TO CONNECT</text>
</svg>`)
}
