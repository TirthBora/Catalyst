//go:build !windows

package monitor

import (
	"runtime"
)

func getSystemMemory() MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sysMB := m.Sys / (1024 * 1024)
	allocMB := m.Alloc / (1024 * 1024)

	return MemStats{
		TotalMB: sysMB,
		UsedMB:  allocMB,
		FreeMB:  sysMB - allocMB,
		UsedPct: 0.0,
	}
}
