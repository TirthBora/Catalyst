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

func TestCheckAlerts(t *testing.T) {
	statsNormal := monitor.Stats{
		Memory: monitor.MemStats{UsedPct: 50.0, TotalMB: 16000, UsedMB: 8000},
	}
	if alerts := monitor.CheckAlerts(statsNormal, 85.0); len(alerts) != 0 {
		t.Errorf("expected 0 alerts for normal usage, got %d", len(alerts))
	}

	statsWarn := monitor.Stats{
		Memory: monitor.MemStats{UsedPct: 88.0, TotalMB: 16000, UsedMB: 14080},
	}
	alertsWarn := monitor.CheckAlerts(statsWarn, 85.0)
	if len(alertsWarn) != 1 || alertsWarn[0].Severity != "warning" {
		t.Errorf("expected 1 warning alert for 88%% usage, got %+v", alertsWarn)
	}

	statsCrit := monitor.Stats{
		Memory: monitor.MemStats{UsedPct: 96.5, TotalMB: 16000, UsedMB: 15440},
	}
	alertsCrit := monitor.CheckAlerts(statsCrit, 85.0)
	if len(alertsCrit) != 1 || alertsCrit[0].Severity != "critical" {
		t.Errorf("expected 1 critical alert for 96.5%% usage, got %+v", alertsCrit)
	}
}
