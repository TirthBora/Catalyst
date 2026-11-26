package plugin_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/plugin"
)

func TestPluginManagerEmpty(t *testing.T) {
	dir := t.TempDir()
	mgr := plugin.NewManager(dir)

	if len(mgr.Actions()) != 0 {
		t.Errorf("expected 0 actions for empty directory, got %d", len(mgr.Actions()))
	}
}

func TestPluginManagerLoad(t *testing.T) {
	dir := t.TempDir()
	catDir := filepath.Join(dir, ".catalyst")
	if err := os.MkdirAll(catDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifest := `{
		"plugins": [
			{
				"name": "git-tools",
				"version": "1.0.0",
				"actions": [
					{
						"id": "git-status",
						"name": "Git Status",
						"command": "git",
						"args": ["status", "-s"]
					}
				]
			}
		],
		"actions": [
			{
				"id": "lint",
				"name": "Run Linter",
				"command": "npm",
				"args": ["run", "lint"]
			}
		]
	}`

	if err := os.WriteFile(filepath.Join(catDir, "plugins.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := plugin.NewManager(dir)

	actions := mgr.Actions()
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}

	// Retrieve action
	act, err := mgr.GetAction("git-status")
	if err != nil {
		t.Fatalf("failed to get git-status: %v", err)
	}
	if act.Name != "Git Status" || act.Command != "git" {
		t.Errorf("unexpected action details: %+v", act)
	}

	// Invalid action
	_, err = mgr.GetAction("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent action")
	}
}
