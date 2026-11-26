package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"catalyst/internal/project"
)

// Workspace represents a distinct development project environment.
type Workspace struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Active    bool         `json:"active"`
	Project   project.Info `json:"project"`
	CreatedAt time.Time    `json:"createdAt"`
}

// Registry manages multiple workspace directories and tracks the active project.
type Registry struct {
	activeID   string
	workspaces map[string]*Workspace
	mu         sync.RWMutex
}

func NewRegistry(initialRoot string) *Registry {
	r := &Registry{
		workspaces: make(map[string]*Workspace),
	}
	if initialRoot != "" {
		_ = r.Register(initialRoot)
	}
	return r
}

func generateID() string {
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Register adds a workspace path to the registry, detecting project metadata.
func (r *Registry) Register(dir string) *Workspace {
	r.mu.Lock()
	defer r.mu.Unlock()

	cleanPath := filepath.Clean(dir)

	// Check if already registered
	for _, w := range r.workspaces {
		if w.Path == cleanPath {
			return w
		}
	}

	info := project.Detect(cleanPath)
	name := info.Name
	if name == "" {
		name = filepath.Base(cleanPath)
	}

	id := generateID()
	ws := &Workspace{
		ID:        id,
		Name:      name,
		Path:      cleanPath,
		Active:    len(r.workspaces) == 0, // First registered workspace defaults to active
		Project:   info,
		CreatedAt: time.Now().UTC(),
	}

	if ws.Active {
		r.activeID = id
	}

	r.workspaces[id] = ws
	return ws
}

// GetActive returns the currently active workspace.
func (r *Registry) GetActive() (*Workspace, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.activeID == "" {
		return nil, false
	}
	ws, ok := r.workspaces[r.activeID]
	return ws, ok
}

// SetActive switches the active workspace by ID.
func (r *Registry) SetActive(id string) (*Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	target, exists := r.workspaces[id]
	if !exists {
		return nil, fmt.Errorf("workspace %q not found", id)
	}

	for _, w := range r.workspaces {
		w.Active = (w.ID == id)
	}

	r.activeID = id
	return target, nil
}

// List returns all registered workspaces.
func (r *Registry) List() []*Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Workspace, 0, len(r.workspaces))
	for _, w := range r.workspaces {
		list = append(list, w)
	}
	return list
}

// Count returns the number of registered workspaces.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.workspaces)
}
