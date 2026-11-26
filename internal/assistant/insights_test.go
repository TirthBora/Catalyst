package assistant_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/assistant"
	"catalyst/internal/project"
)

func TestEvaluateWorkspace(t *testing.T) {
	dir := t.TempDir()

	// 1. Create .git without .gitignore
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	info := project.Info{
		Type: project.TypeVite,
	}

	insights := assistant.EvaluateWorkspace(dir, info)

	var foundGitignore bool
	var foundViteTip bool

	for _, ins := range insights {
		if ins.Title == "Missing .gitignore" {
			foundGitignore = true
		}
		if ins.Title == "Vite HMR Active" {
			foundViteTip = true
		}
	}

	if !foundGitignore {
		t.Error("expected Missing .gitignore warning")
	}

	if !foundViteTip {
		t.Error("expected Vite HMR Active insight")
	}

	// 2. Add package.json without node_modules
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	insights2 := assistant.EvaluateWorkspace(dir, info)

	var foundMissingModules bool
	for _, ins := range insights2 {
		if ins.Title == "Dependencies Uninstalled" {
			foundMissingModules = true
		}
	}
	if !foundMissingModules {
		t.Error("expected Dependencies Uninstalled warning")
	}
}
