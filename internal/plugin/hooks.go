package plugin

import (
	"path/filepath"
	"strings"
	"sync"
)

// HookType defines the event trigger for an automated plugin hook.
type HookType string

const (
	HookOnFileChanged HookType = "on_file_changed"
	HookOnStart       HookType = "on_start"
)

// Hook associates an event trigger and pattern with a plugin action.
type Hook struct {
	Type     HookType `json:"type"`
	Pattern  string   `json:"pattern,omitempty"` // e.g. "*.go", "*.proto"
	ActionID string   `json:"actionId"`
}

// HookDispatcher evaluates events and resolves matching actions.
type HookDispatcher struct {
	hooks []Hook
	mu    sync.RWMutex
}

func NewHookDispatcher() *HookDispatcher {
	return &HookDispatcher{
		hooks: make([]Hook, 0),
	}
}

// RegisterHook adds an event hook to the dispatcher.
func (d *HookDispatcher) RegisterHook(h Hook) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hooks = append(d.hooks, h)
}

// MatchFileChanged returns all ActionIDs that match a changed filename.
func (d *HookDispatcher) MatchFileChanged(filename string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var matched []string
	base := filepath.Base(filename)

	for _, h := range d.hooks {
		if h.Type != HookOnFileChanged {
			continue
		}

		if h.Pattern == "" || h.Pattern == "*" {
			matched = append(matched, h.ActionID)
			continue
		}

		// Check extension pattern (e.g. "*.go" or ".go")
		if strings.HasPrefix(h.Pattern, "*.") {
			ext := strings.TrimPrefix(h.Pattern, "*")
			if strings.HasSuffix(base, ext) {
				matched = append(matched, h.ActionID)
			}
		} else if match, _ := filepath.Match(h.Pattern, base); match {
			matched = append(matched, h.ActionID)
		}
	}

	return matched
}

// MatchOnStart returns all ActionIDs configured to run on workspace startup.
func (d *HookDispatcher) MatchOnStart() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var matched []string
	for _, h := range d.hooks {
		if h.Type == HookOnStart {
			matched = append(matched, h.ActionID)
		}
	}
	return matched
}
