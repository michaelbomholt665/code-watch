package util

import (
	"runtime"
)

// GetHeapAllocMB returns the current heap allocation in MB.
func GetHeapAllocMB() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc / 1024 / 1024
}
