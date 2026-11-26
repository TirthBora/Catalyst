package process_test

import (
	"runtime"
	"testing"
	"time"

	"catalyst/internal/process"
	"catalyst/internal/websocket"
)

func getEchoCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/c", "echo", "catalyst-process-test"}
	}
	return "echo", []string{"catalyst-process-test"}
}

func getSleepCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "5", "127.0.0.1"}
	}
	return "sleep", []string{"5"}
}

func TestProcessLifecycle(t *testing.T) {
	hub := websocket.NewHub()
	pm := process.NewManager(hub)

	cmd, args := getEchoCommand()
	p, err := pm.Start("echo-job", cmd, args, "")
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if p.ID != "echo-job" {
		t.Errorf("expected ID echo-job, got %s", p.ID)
	}

	if p.PID <= 0 {
		t.Errorf("expected PID > 0, got %d", p.PID)
	}

	// Wait for echo command to complete
	time.Sleep(300 * time.Millisecond)

	retrieved, ok := pm.Get("echo-job")
	if !ok {
		t.Fatal("expected process to be found in manager")
	}

	if retrieved.State != process.StateStopped {
		t.Errorf("expected process to be stopped, got %s", retrieved.State)
	}

	if len(retrieved.Logs) == 0 {
		t.Error("expected at least one log line from stdout")
	}
}

func TestProcessStopAndRestart(t *testing.T) {
	hub := websocket.NewHub()
	pm := process.NewManager(hub)

	cmd, args := getSleepCommand()
	p, err := pm.Start("sleep-job", cmd, args, "")
	if err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}

	if p.State != process.StateRunning {
		t.Errorf("expected StateRunning, got %s", p.State)
	}

	// Stop process
	if err := pm.Stop("sleep-job"); err != nil {
		t.Errorf("failed to stop process: %v", err)
	}

	stopped, _ := pm.Get("sleep-job")
	if stopped.State != process.StateStopped {
		t.Errorf("expected StateStopped after stop, got %s", stopped.State)
	}

	// Restart process
	restarted, err := pm.Restart("sleep-job")
	if err != nil {
		t.Fatalf("failed to restart process: %v", err)
	}

	if restarted.State != process.StateRunning {
		t.Errorf("expected StateRunning after restart, got %s", restarted.State)
	}

	// Cleanup
	pm.StopAll()
}
