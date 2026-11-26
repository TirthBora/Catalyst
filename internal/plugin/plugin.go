package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Action represents an executable development task defined by a plugin.
type Action struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Icon        string   `json:"icon,omitempty"`
}

// Plugin represents a collection of custom developer actions and hooks.
type Plugin struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Actions     []Action `json:"actions"`
}

// Manager coordinates plugin discovery, manifest parsing, and safe action resolution.
type Manager struct {
	rootDir string
	plugins []Plugin
	actions map[string]Action
	mu      sync.RWMutex
}

func NewManager(rootDir string) *Manager {
	m := &Manager{
		rootDir: rootDir,
		plugins: make([]Plugin, 0),
		actions: make(map[string]Action),
	}
	_ = m.Load()
	return m
}

func (m *Manager) manifestPath() string {
	dotPath := filepath.Join(m.rootDir, ".catalyst", "plugins.json")
	if _, err := os.Stat(dotPath); err == nil {
		return dotPath
	}

	rootPath := filepath.Join(m.rootDir, "catalyst.plugins.json")
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath
	}

	return dotPath
}

// Load reads plugin manifests from disk.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins = make([]Plugin, 0)
	m.actions = make(map[string]Action)

	path := m.manifestPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var manifest struct {
		Plugins []Plugin `json:"plugins"`
		Actions []Action `json:"actions"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	m.plugins = manifest.Plugins

	// Index direct actions
	for _, a := range manifest.Actions {
		m.actions[a.ID] = a
	}

	// Index actions nested inside plugins
	for _, p := range manifest.Plugins {
		for _, a := range p.Actions {
			m.actions[a.ID] = a
		}
	}

	return nil
}

// Actions returns all registered developer action shortcuts.
func (m *Manager) Actions() []Action {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]Action, 0, len(m.actions))
	for _, a := range m.actions {
		list = append(list, a)
	}
	return list
}

// GetAction retrieves a specific action by ID.
func (m *Manager) GetAction(id string) (Action, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	a, exists := m.actions[id]
	if !exists {
		return Action{}, fmt.Errorf("plugin action %q not found", id)
	}
	return a, nil
}

// Plugins returns all loaded plugins.
func (m *Manager) Plugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Plugin(nil), m.plugins...)
}
