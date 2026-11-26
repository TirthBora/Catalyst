package monitor

import (
	"os"
	"runtime"
	"time"
)

// Stats holds host and runtime system metrics.
type Stats struct {
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Hostname    string    `json:"hostname"`
	NumCPU      int       `json:"numCpu"`
	NumGoroutine int      `json:"numGoroutine"`
	UptimeSec   int64     `json:"uptimeSec"`
	Memory      MemStats  `json:"memory"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemStats contains memory utilization details in megabytes and percentage.
type MemStats struct {
	TotalMB uint64  `json:"totalMb"`
	UsedMB  uint64  `json:"usedMb"`
	FreeMB  uint64  `json:"freeMb"`
	UsedPct float64 `json:"usedPct"`
	AllocMB uint64  `json:"allocMb"` // Process heap alloc
}

var startTime = time.Now()

// GetStats returns current system and process metrics.
func GetStats() Stats {
	hostname, _ := os.Hostname()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mem := getSystemMemory()
	mem.AllocMB = m.Alloc / (1024 * 1024)

	return Stats{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Hostname:     hostname,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		UptimeSec:    int64(time.Since(startTime).Seconds()),
		Memory:       mem,
		Timestamp:    time.Now().UTC(),
	}
}
