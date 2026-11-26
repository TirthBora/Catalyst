package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/workspace"
)

func TestWorkspaceRegistry(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir1, "go.mod"), []byte("module dir1\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir2, "package.json"), []byte(`{"name": "frontend"}`), 0644)

	reg := workspace.NewRegistry(dir1)

	if reg.Count() != 1 {
		t.Fatalf("expected 1 workspace, got %d", reg.Count())
	}

	active, ok := reg.GetActive()
	if !ok || active.Path != filepath.Clean(dir1) {
		t.Fatalf("expected dir1 to be active, got %+v", active)
	}

	// Register second workspace
	ws2 := reg.Register(dir2)
	if reg.Count() != 2 {
		t.Errorf("expected 2 workspaces, got %d", reg.Count())
	}
	if ws2.Name != "frontend" {
		t.Errorf("expected name frontend, got %s", ws2.Name)
	}

	// Initial workspace should still be active
	active, _ = reg.GetActive()
	if active.Path != filepath.Clean(dir1) {
		t.Errorf("expected dir1 to remain active after registering dir2")
	}

	// Switch active workspace to dir2
	switched, err := reg.SetActive(ws2.ID)
	if err != nil {
		t.Fatalf("failed to switch active workspace: %v", err)
	}
	if !switched.Active || switched.ID != ws2.ID {
		t.Errorf("expected switched workspace to be active")
	}

	activeNow, _ := reg.GetActive()
	if activeNow.ID != ws2.ID {
		t.Errorf("expected active workspace to be ws2, got %s", activeNow.ID)
	}

	// Test invalid switch
	_, err = reg.SetActive("invalid-id")
	if err == nil {
		t.Error("expected error when setting active with invalid ID")
	}

	// Test duplicate registration
	dup := reg.Register(dir1)
	if dup.Path != filepath.Clean(dir1) || reg.Count() != 2 {
		t.Errorf("duplicate register should return existing workspace and not increment count")
	}
}
