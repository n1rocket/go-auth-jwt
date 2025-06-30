package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Counter is a metric that can only increase
type Counter struct {
	name   string
	help   string
	value  int64
	labels map[string]*labeledCounter
	mu     sync.RWMutex
}

// labeledCounter holds a counter value for a specific label combination
type labeledCounter struct {
	value  int64
	labels map[string]string
}

// NewCounter creates a new counter metric
func NewCounter(name, help string) *Counter {
	return &Counter{
		name:   name,
		help:   help,
		labels: make(map[string]*labeledCounter),
	}
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter
func (c *Counter) Add(delta int64) {
	if delta < 0 {
		panic("counter cannot decrease")
	}
	atomic.AddInt64(&c.value, delta)
}

// WithLabels returns a labeled counter
func (c *Counter) WithLabels(labels map[string]string) *LabeledCounter {
	key := labelsToKey(labels)
	
	c.mu.RLock()
	lc, exists := c.labels[key]
	c.mu.RUnlock()
	
	if !exists {
		c.mu.Lock()
		lc = &labeledCounter{labels: labels}
		c.labels[key] = lc
		c.mu.Unlock()
	}
	
	return &LabeledCounter{counter: lc}
}

// Value returns the current counter value
func (c *Counter) Value() interface{} {
	return atomic.LoadInt64(&c.value)
}

// Name returns the metric name
func (c *Counter) Name() string {
	return c.name
}

// String returns a string representation of the counter
func (c *Counter) String() string {
	return fmt.Sprintf("%s: %d", c.name, c.Value())
}

// Reset resets the counter to zero
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
	
	c.mu.Lock()
	c.labels = make(map[string]*labeledCounter)
	c.mu.Unlock()
}

// LabeledCounter is a counter with labels
type LabeledCounter struct {
	counter *labeledCounter
}

// Inc increments the labeled counter by 1
func (lc *LabeledCounter) Inc() {
	atomic.AddInt64(&lc.counter.value, 1)
}

// Add adds the given value to the labeled counter
func (lc *LabeledCounter) Add(delta int64) {
	if delta < 0 {
		panic("counter cannot decrease")
	}
	atomic.AddInt64(&lc.counter.value, delta)
}

// Value returns the current value
func (lc *LabeledCounter) Value() int64 {
	return atomic.LoadInt64(&lc.counter.value)
}

// labelsToKey converts labels to a string key
func labelsToKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	
	key := ""
	for k, v := range labels {
		if key != "" {
			key += ","
		}
		key += fmt.Sprintf("%s=%s", k, v)
	}
	return key
}