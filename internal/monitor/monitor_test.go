package monitor_test

import (
	"runtime"
	"testing"

	"catalyst/internal/monitor"
)

func TestGetStats(t *testing.T) {
	stats := monitor.GetStats()

	if stats.OS != runtime.GOOS {
		t.Errorf("expected OS %s, got %s", runtime.GOOS, stats.OS)
	}

	if stats.Arch != runtime.GOARCH {
		t.Errorf("expected Arch %s, got %s", runtime.GOARCH, stats.Arch)
	}

	if stats.NumCPU <= 0 {
		t.Errorf("expected NumCPU > 0, got %d", stats.NumCPU)
	}

	if stats.Memory.TotalMB == 0 {
		t.Errorf("expected Memory.TotalMB > 0, got %d", stats.Memory.TotalMB)
	}

	if stats.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}
