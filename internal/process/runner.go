package process

import (
	"fmt"
	"sync"

	"catalyst/internal/project"
)

// DevRunner manages the lifecycle of the active project's development server.
type DevRunner struct {
	pm       *Manager
	activeID string
	mu       sync.RWMutex
}

func NewDevRunner(pm *Manager) *DevRunner {
	return &DevRunner{
		pm:       pm,
		activeID: "dev-server",
	}
}

// StartScript executes ONLY a script that exists in the allowed project script list.
// Any command or script name not found in allowedScripts is rejected.
func (r *DevRunner) StartScript(scriptName string, allowedScripts []project.Script, workDir string) (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var matched *project.Script
	for i := range allowedScripts {
		if allowedScripts[i].Name == scriptName {
			matched = &allowedScripts[i]
			break
		}
	}

	if matched == nil {
		return nil, fmt.Errorf("script %q is not a recognized project script", scriptName)
	}

	// Stop existing dev server if already running
	_ = r.pm.Stop(r.activeID)

	return r.pm.Start(r.activeID, matched.Command, matched.Args, workDir)
}

// Stop terminates the currently running dev server.
func (r *DevRunner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pm.Stop(r.activeID)
}

// Restart restarts the currently running dev server.
func (r *DevRunner) Restart() (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pm.Restart(r.activeID)
}

// GetStatus returns the process status of the active dev server.
func (r *DevRunner) GetStatus() (*Process, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pm.Get(r.activeID)
}
