package metrics

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

// Histogram tracks the distribution of values
type Histogram struct {
	name    string
	help    string
	buckets []float64
	counts  []uint64
	sum     uint64 // Stores float64 as bits
	count   uint64
	labels  map[string]*labeledHistogram
	mu      sync.RWMutex
}

// labeledHistogram holds histogram data for a specific label combination
type labeledHistogram struct {
	buckets []float64
	counts  []uint64
	sum     uint64
	count   uint64
	labels  map[string]string
	mu      sync.Mutex
}

// DefaultBuckets are the default histogram buckets (in seconds)
var DefaultBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
}

// NewHistogram creates a new histogram metric
func NewHistogram(name, help string) *Histogram {
	return NewHistogramWithBuckets(name, help, DefaultBuckets)
}

// NewHistogramWithBuckets creates a new histogram with custom buckets
func NewHistogramWithBuckets(name, help string, buckets []float64) *Histogram {
	// Sort buckets in ascending order
	sortedBuckets := make([]float64, len(buckets))
	copy(sortedBuckets, buckets)
	sort.Float64s(sortedBuckets)
	
	return &Histogram{
		name:    name,
		help:    help,
		buckets: sortedBuckets,
		counts:  make([]uint64, len(sortedBuckets)+1), // +1 for +Inf bucket
		labels:  make(map[string]*labeledHistogram),
	}
}

// Observe adds a value to the histogram
func (h *Histogram) Observe(value float64) {
	// Update sum
	for {
		oldBits := atomic.LoadUint64(&h.sum)
		oldSum := math.Float64frombits(oldBits)
		newSum := oldSum + value
		newBits := math.Float64bits(newSum)
		
		if atomic.CompareAndSwapUint64(&h.sum, oldBits, newBits) {
			break
		}
	}
	
	// Update count
	atomic.AddUint64(&h.count, 1)
	
	// Find the appropriate bucket
	bucketIndex := len(h.buckets)
	for i, upper := range h.buckets {
		if value <= upper {
			bucketIndex = i
			break
		}
	}
	
	// Update bucket count
	atomic.AddUint64(&h.counts[bucketIndex], 1)
}

// WithLabels returns a labeled histogram
func (h *Histogram) WithLabels(labels map[string]string) *LabeledHistogram {
	key := labelsToKey(labels)
	
	h.mu.RLock()
	lh, exists := h.labels[key]
	h.mu.RUnlock()
	
	if !exists {
		h.mu.Lock()
		lh = &labeledHistogram{
			buckets: make([]float64, len(h.buckets)),
			counts:  make([]uint64, len(h.buckets)+1),
			labels:  labels,
		}
		copy(lh.buckets, h.buckets)
		h.labels[key] = lh
		h.mu.Unlock()
	}
	
	return &LabeledHistogram{histogram: lh}
}

// Buckets returns the bucket upper bounds and their counts
func (h *Histogram) Buckets() map[float64]uint64 {
	result := make(map[float64]uint64)
	
	cumulative := uint64(0)
	for i, bucket := range h.buckets {
		cumulative += atomic.LoadUint64(&h.counts[i])
		result[bucket] = cumulative
	}
	
	// Add +Inf bucket
	cumulative += atomic.LoadUint64(&h.counts[len(h.buckets)])
	result[math.Inf(1)] = cumulative
	
	return result
}

// Sum returns the sum of all observed values
func (h *Histogram) Sum() float64 {
	bits := atomic.LoadUint64(&h.sum)
	return math.Float64frombits(bits)
}

// Count returns the total number of observations
func (h *Histogram) Count() uint64 {
	return atomic.LoadUint64(&h.count)
}

// Value returns the histogram data
func (h *Histogram) Value() interface{} {
	return map[string]interface{}{
		"buckets": h.Buckets(),
		"sum":     h.Sum(),
		"count":   h.Count(),
	}
}

// Name returns the metric name
func (h *Histogram) Name() string {
	return h.name
}

// String returns a string representation of the histogram
func (h *Histogram) String() string {
	return fmt.Sprintf("%s: count=%d, sum=%.2f", h.name, h.Count(), h.Sum())
}

// Percentile calculates the given percentile value
func (h *Histogram) Percentile(p float64) float64 {
	if p < 0 || p > 100 {
		panic("percentile must be between 0 and 100")
	}
	
	count := h.Count()
	if count == 0 {
		return 0
	}
	
	threshold := uint64(float64(count) * p / 100.0)
	cumulative := uint64(0)
	
	for i, bucketCount := range h.counts[:len(h.counts)-1] {
		cumulative += atomic.LoadUint64(&bucketCount)
		if cumulative >= threshold {
			if i == 0 {
				return h.buckets[i] / 2
			}
			// Linear interpolation within the bucket
			prevBucket := 0.0
			if i > 0 {
				prevBucket = h.buckets[i-1]
			}
			return prevBucket + (h.buckets[i]-prevBucket)/2
		}
	}
	
	// Value is in the +Inf bucket
	if len(h.buckets) > 0 {
		return h.buckets[len(h.buckets)-1] * 2
	}
	return 0
}

// Reset resets the histogram
func (h *Histogram) Reset() {
	atomic.StoreUint64(&h.sum, 0)
	atomic.StoreUint64(&h.count, 0)
	
	for i := range h.counts {
		atomic.StoreUint64(&h.counts[i], 0)
	}
	
	h.mu.Lock()
	h.labels = make(map[string]*labeledHistogram)
	h.mu.Unlock()
}

// LabeledHistogram is a histogram with labels
type LabeledHistogram struct {
	histogram *labeledHistogram
}

// Observe adds a value to the labeled histogram
func (lh *LabeledHistogram) Observe(value float64) {
	lh.histogram.mu.Lock()
	defer lh.histogram.mu.Unlock()
	
	// Update sum
	oldSum := math.Float64frombits(lh.histogram.sum)
	newSum := oldSum + value
	lh.histogram.sum = math.Float64bits(newSum)
	
	// Update count
	lh.histogram.count++
	
	// Find the appropriate bucket
	bucketIndex := len(lh.histogram.buckets)
	for i, upper := range lh.histogram.buckets {
		if value <= upper {
			bucketIndex = i
			break
		}
	}
	
	// Update bucket count
	lh.histogram.counts[bucketIndex]++
}