package metrics

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// Gauge is a metric that can increase or decrease
type Gauge struct {
	name   string
	help   string
	value  uint64 // Using uint64 to store float64 bits atomically
	labels map[string]*labeledGauge
	mu     sync.RWMutex
}

// labeledGauge holds a gauge value for a specific label combination
type labeledGauge struct {
	value  uint64
	labels map[string]string
}

// NewGauge creates a new gauge metric
func NewGauge(name, help string) *Gauge {
	return &Gauge{
		name:   name,
		help:   help,
		labels: make(map[string]*labeledGauge),
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	atomic.StoreUint64(&g.value, math.Float64bits(value))
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(delta float64) {
	for {
		oldBits := atomic.LoadUint64(&g.value)
		oldValue := math.Float64frombits(oldBits)
		newValue := oldValue + delta
		newBits := math.Float64bits(newValue)

		if atomic.CompareAndSwapUint64(&g.value, oldBits, newBits) {
			break
		}
	}
}

// WithLabels returns a labeled gauge
func (g *Gauge) WithLabels(labels map[string]string) *LabeledGauge {
	key := labelsToKey(labels)

	g.mu.RLock()
	lg, exists := g.labels[key]
	g.mu.RUnlock()

	if !exists {
		g.mu.Lock()
		// Check again after acquiring write lock
		lg, exists = g.labels[key]
		if !exists {
			lg = &labeledGauge{labels: labels}
			g.labels[key] = lg
		}
		g.mu.Unlock()
	}

	return &LabeledGauge{gauge: lg}
}

// Value returns the current gauge value
func (g *Gauge) Value() interface{} {
	bits := atomic.LoadUint64(&g.value)
	return math.Float64frombits(bits)
}

// Name returns the metric name
func (g *Gauge) Name() string {
	return g.name
}

// String returns a string representation of the gauge
func (g *Gauge) String() string {
	return fmt.Sprintf("%s: %.2f", g.name, g.Value())
}

// Reset resets the gauge to zero
func (g *Gauge) Reset() {
	atomic.StoreUint64(&g.value, 0)

	g.mu.Lock()
	g.labels = make(map[string]*labeledGauge)
	g.mu.Unlock()
}

// LabeledGauge is a gauge with labels
type LabeledGauge struct {
	gauge *labeledGauge
}

// Set sets the labeled gauge to the given value
func (lg *LabeledGauge) Set(value float64) {
	atomic.StoreUint64(&lg.gauge.value, math.Float64bits(value))
}

// Inc increments the labeled gauge by 1
func (lg *LabeledGauge) Inc() {
	lg.Add(1)
}

// Dec decrements the labeled gauge by 1
func (lg *LabeledGauge) Dec() {
	lg.Add(-1)
}

// Add adds the given value to the labeled gauge
func (lg *LabeledGauge) Add(delta float64) {
	for {
		oldBits := atomic.LoadUint64(&lg.gauge.value)
		oldValue := math.Float64frombits(oldBits)
		newValue := oldValue + delta
		newBits := math.Float64bits(newValue)

		if atomic.CompareAndSwapUint64(&lg.gauge.value, oldBits, newBits) {
			break
		}
	}
}

// Value returns the current value
func (lg *LabeledGauge) Value() float64 {
	bits := atomic.LoadUint64(&lg.gauge.value)
	return math.Float64frombits(bits)
}
