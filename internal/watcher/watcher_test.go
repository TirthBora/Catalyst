package watcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"catalyst/internal/watcher"
	"catalyst/internal/websocket"
)

func TestShouldIgnore(t *testing.T) {
	ignored := []string{
		".git",
		"node_modules",
		"dist",
		"build",
		".vscode",
		".idea",
		"vendor",
		".venv",
		"__pycache__",
	}

	for _, dir := range ignored {
		if !watcher.ShouldIgnore(dir) {
			t.Errorf("expected %q to be ignored", dir)
		}
	}

	allowed := []string{
		"src",
		"components",
		"internal",
		"pkg",
		"cmd",
		"assets",
		"public",
	}

	for _, dir := range allowed {
		if watcher.ShouldIgnore(dir) {
			t.Errorf("expected %q to NOT be ignored", dir)
		}
	}
}

func TestWatcherFileEvents(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "catalyst-watcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hub := websocket.NewHub()
	w, err := watcher.NewWatcher(tempDir, hub)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Create a new file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello catalyst"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a new subdirectory dynamically
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Allow dynamic registration to process
	time.Sleep(150 * time.Millisecond)

	// Write inside the subdirectory
	subFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(subFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	// Allow debouncing to fire
	time.Sleep(200 * time.Millisecond)

	// Remove the file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("failed to remove test file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
}
