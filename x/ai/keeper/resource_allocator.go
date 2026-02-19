package keeper

import (
	"context"
	"math"
	"sync"
	"time"
)

// ResourceAllocator provides AI-driven resource allocation recommendations
// for individual nodes. It monitors CPU, memory, and disk usage patterns
// and suggests resource allocation changes for optimal performance.
type ResourceAllocator struct {
	mu       sync.RWMutex
	history  []resourceSnapshot
	maxSize  int
}

// resourceSnapshot captures resource utilization at a point in time.
type resourceSnapshot struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	GoroutineCount int
	Timestamp     time.Time
}

// ResourceAllocation describes the recommended resource allocation for a node.
type ResourceAllocation struct {
	CPUPriority      string  `json:"cpu_priority"`       // "high" | "normal" | "low"
	MemoryLimitMB    int     `json:"memory_limit_mb"`
	MaxGoroutines    int     `json:"max_goroutines"`
	CacheSizeMB      int     `json:"cache_size_mb"`
	Recommendation   string  `json:"recommendation"`
	Confidence       float64 `json:"confidence"`
}

// NewResourceAllocator creates a new resource allocator.
func NewResourceAllocator() *ResourceAllocator {
	return &ResourceAllocator{
		history: make([]resourceSnapshot, 0, 100),
		maxSize: 100,
	}
}

// RecordSnapshot records current resource utilization.
func (ra *ResourceAllocator) RecordSnapshot(cpuPercent, memPercent, diskPercent float64, goroutines int) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.history = append(ra.history, resourceSnapshot{
		CPUPercent:     cpuPercent,
		MemoryPercent:  memPercent,
		DiskPercent:    diskPercent,
		GoroutineCount: goroutines,
		Timestamp:      time.Now(),
	})

	if len(ra.history) > ra.maxSize {
		ra.history = ra.history[1:]
	}
}

// Recommend returns a resource allocation recommendation based on current usage patterns.
func (ra *ResourceAllocator) Recommend(_ context.Context) (*ResourceAllocation, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	if len(ra.history) == 0 {
		return &ResourceAllocation{
			CPUPriority:    "normal",
			MemoryLimitMB:  4096,
			MaxGoroutines:  10000,
			CacheSizeMB:    512,
			Recommendation: "no data collected yet — using default allocation",
			Confidence:     0.5,
		}, nil
	}

	// Compute averages over recent history
	recentStart := 0
	if len(ra.history) > 20 {
		recentStart = len(ra.history) - 20
	}
	recent := ra.history[recentStart:]

	var avgCPU, avgMem, avgDisk float64
	var avgGoroutines float64
	for _, s := range recent {
		avgCPU += s.CPUPercent
		avgMem += s.MemoryPercent
		avgDisk += s.DiskPercent
		avgGoroutines += float64(s.GoroutineCount)
	}
	n := float64(len(recent))
	avgCPU /= n
	avgMem /= n
	avgDisk /= n
	avgGoroutines /= n

	// Compute peak values
	var peakCPU, peakMem float64
	for _, s := range recent {
		if s.CPUPercent > peakCPU {
			peakCPU = s.CPUPercent
		}
		if s.MemoryPercent > peakMem {
			peakMem = s.MemoryPercent
		}
	}

	// Generate recommendation
	alloc := &ResourceAllocation{
		CPUPriority:   "normal",
		MemoryLimitMB: 4096,
		MaxGoroutines: 10000,
		CacheSizeMB:   512,
		Confidence:    0.85,
	}

	var reasons []string

	// CPU recommendation
	if avgCPU > 80 {
		alloc.CPUPriority = "high"
		reasons = append(reasons, "CPU consistently high — recommend increasing CPU allocation or reducing load")
	} else if avgCPU < 20 {
		alloc.CPUPriority = "low"
		reasons = append(reasons, "CPU underutilized — can reduce allocation")
	}

	// Memory recommendation
	if avgMem > 80 {
		alloc.MemoryLimitMB = 8192
		reasons = append(reasons, "memory pressure detected — recommend doubling memory limit")
	} else if avgMem < 30 {
		alloc.MemoryLimitMB = 2048
		reasons = append(reasons, "memory underutilized — can reduce to 2GB")
	}

	// Goroutine recommendation
	if avgGoroutines > 8000 {
		alloc.MaxGoroutines = int(avgGoroutines * 1.5)
		reasons = append(reasons, "high goroutine count — increasing limit")
	}

	// Cache recommendation based on disk I/O pressure
	if avgDisk > 70 {
		alloc.CacheSizeMB = 1024
		reasons = append(reasons, "disk pressure high — increasing cache to reduce I/O")
	}

	// Confidence based on data quality
	if len(ra.history) < 10 {
		alloc.Confidence = 0.6
	} else if peakCPU > 95 || peakMem > 95 {
		alloc.Confidence = 0.9 // Very clear signal
	}

	if len(reasons) == 0 {
		alloc.Recommendation = "resource usage within normal bounds — no changes recommended"
	} else {
		alloc.Recommendation = reasons[0]
		for _, r := range reasons[1:] {
			alloc.Recommendation += "; " + r
		}
	}

	// Clamp confidence
	alloc.Confidence = math.Min(alloc.Confidence, 0.95)

	return alloc, nil
}
