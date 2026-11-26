//go:build windows

package monitor

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func getSystemMemory() MemStats {
	var memStatus memoryStatusEx
	memStatus.cbSize = uint32(unsafe.Sizeof(memStatus))

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	r, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if r == 0 {
		return MemStats{}
	}

	totalMB := memStatus.ullTotalPhys / (1024 * 1024)
	freeMB := memStatus.ullAvailPhys / (1024 * 1024)
	usedMB := totalMB - freeMB
	usedPct := 0.0
	if totalMB > 0 {
		usedPct = float64(usedMB) / float64(totalMB) * 100.0
	}

	return MemStats{
		TotalMB: totalMB,
		UsedMB:  usedMB,
		FreeMB:  freeMB,
		UsedPct: float64(int(usedPct*10)) / 10.0,
	}
}
