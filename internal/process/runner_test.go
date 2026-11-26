package process_test

import (
	"runtime"
	"testing"
	"time"

	"catalyst/internal/process"
	"catalyst/internal/project"
	"catalyst/internal/websocket"
)

func TestDevRunnerVetting(t *testing.T) {
	hub := websocket.NewHub()
	pm := process.NewManager(hub)
	runner := process.NewDevRunner(pm)

	allowed := []project.Script{
		{Name: "test-echo", Command: "echo", Args: []string{"test"}},
	}
	if runtime.GOOS == "windows" {
		allowed = []project.Script{
			{Name: "test-echo", Command: "cmd.exe", Args: []string{"/c", "echo", "test"}},
		}
	}

	// 1. Rejects unvetted script
	_, err := runner.StartScript("malicious-command", allowed, "")
	if err == nil {
		t.Error("expected unvetted script to be rejected")
	}

	// 2. Accepts vetted script
	p, err := runner.StartScript("test-echo", allowed, "")
	if err != nil {
		t.Fatalf("expected vetted script to start, got error: %v", err)
	}

	if p.ID != "dev-server" {
		t.Errorf("expected ID dev-server, got %s", p.ID)
	}

	// Check status
	retrieved, ok := runner.GetStatus()
	if !ok || retrieved.ID != "dev-server" {
		t.Error("expected dev-server status to be retrievable")
	}

	time.Sleep(300 * time.Millisecond)

	// Stop
	_ = runner.Stop()
	pm.StopAll()
}
