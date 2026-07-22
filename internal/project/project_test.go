package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module example\n\ngo 1.24\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(root, "cmd", "app"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "cmd", "app", "main.go"),
		[]byte("package main\nfunc main() {}\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	proj, err := DetectFrom(root) // or DetectFrom(root) if you've refactored
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proj.Root != root {
		t.Fatalf("expected root %q, got %q", root, proj.Root)
	}

	expected := filepath.Join("cmd", "app", "main.go")
	if proj.EntryPoint != expected {
		t.Fatalf("expected entry point %q, got %q", expected, proj.EntryPoint)
	}
}
