package metrics

import (
	"runtime"
	"time"
)

// SystemMetrics contains all system-related metrics
type SystemMetrics struct {
	GoRoutines      *Gauge
	MemoryAllocated *Gauge
	MemoryTotal     *Gauge
	GCPauses        *Histogram
}

// NewSystemMetrics creates a new SystemMetrics instance
func NewSystemMetrics() *SystemMetrics {
	return &SystemMetrics{
		GoRoutines:      NewGauge("go_goroutines", "Number of goroutines"),
		MemoryAllocated: NewGauge("go_memory_allocated_bytes", "Allocated memory in bytes"),
		MemoryTotal:     NewGauge("go_memory_total_bytes", "Total memory obtained from OS"),
		GCPauses:        NewHistogram("go_gc_pause_seconds", "GC pause durations in seconds"),
	}
}

// Register registers all system metrics
func (s *SystemMetrics) Register(registry MetricRegistry) {
	registry.Register(s.GoRoutines)
	registry.Register(s.MemoryAllocated)
	registry.Register(s.MemoryTotal)
	registry.Register(s.GCPauses)
}

// Update updates all system metrics
func (s *SystemMetrics) Update() {
	// Collect memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Update gauges
	s.GoRoutines.Set(float64(runtime.NumGoroutine()))
	s.MemoryAllocated.Set(float64(memStats.Alloc))
	s.MemoryTotal.Set(float64(memStats.Sys))

	// Record GC pauses
	for i := 0; i < int(memStats.NumGC); i++ {
		if i < len(memStats.PauseNs) {
			pauseNs := memStats.PauseNs[(memStats.NumGC+uint32(i))%uint32(len(memStats.PauseNs))]
			s.GCPauses.Observe(float64(pauseNs) / float64(time.Second))
		}
	}
}

// StartCollector starts a goroutine to periodically update system metrics
func (s *SystemMetrics) StartCollector(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial update
	s.Update()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.Update()
		}
	}
}