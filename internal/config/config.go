package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config represents persistent workspace and development settings.
type Config struct {
	Port          string   `json:"port"`
	TargetURL     string   `json:"targetUrl"`
	CustomIgnore  []string `json:"customIgnore"`
	DefaultDevice string   `json:"defaultDevice"` // "full", "tablet", "mobile"
	AutoReload    bool     `json:"autoReload"`
	Theme         string   `json:"theme"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Port:          "9999",
		TargetURL:     "/preview/",
		CustomIgnore:  []string{},
		DefaultDevice: "full",
		AutoReload:    true,
		Theme:         "dark",
	}
}

// Manager handles loading, updating, and saving workspace configurations.
type Manager struct {
	rootDir string
	config  Config
	mu      sync.RWMutex
}

func NewManager(rootDir string) *Manager {
	m := &Manager{
		rootDir: rootDir,
		config:  DefaultConfig(),
	}
	_ = m.Load()
	return m
}

func (m *Manager) configFilePath() string {
	// Priority 1: .catalyst/config.json
	dotPath := filepath.Join(m.rootDir, ".catalyst", "config.json")
	if _, err := os.Stat(dotPath); err == nil {
		return dotPath
	}

	// Priority 2: catalyst.config.json
	rootPath := filepath.Join(m.rootDir, "catalyst.config.json")
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath
	}

	// Default to .catalyst/config.json for new saves
	return dotPath
}

// Load reads the workspace configuration from disk if present.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	// Merge with defaults
	if loaded.Port == "" {
		loaded.Port = "9999"
	}
	if loaded.TargetURL == "" {
		loaded.TargetURL = "/preview/"
	}
	if loaded.DefaultDevice == "" {
		loaded.DefaultDevice = "full"
	}
	if loaded.Theme == "" {
		loaded.Theme = "dark"
	}

	m.config = loaded
	return nil
}

// Get returns a copy of the current configuration.
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update updates the in-memory configuration and persists it to disk.
func (m *Manager) Update(newCfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if newCfg.Port == "" {
		newCfg.Port = m.config.Port
	}
	if newCfg.TargetURL == "" {
		newCfg.TargetURL = m.config.TargetURL
	}
	if newCfg.DefaultDevice == "" {
		newCfg.DefaultDevice = m.config.DefaultDevice
	}
	if newCfg.Theme == "" {
		newCfg.Theme = m.config.Theme
	}

	m.config = newCfg

	// Save to disk
	path := m.configFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
