package metrics

// RateLimitMetrics contains all rate limiting-related metrics
type RateLimitMetrics struct {
	RateLimitHits     *Counter
	RateLimitExceeded *Counter
}

// NewRateLimitMetrics creates a new RateLimitMetrics instance
func NewRateLimitMetrics() *RateLimitMetrics {
	return &RateLimitMetrics{
		RateLimitHits:     NewCounter("rate_limit_hits_total", "Total number of rate limit checks"),
		RateLimitExceeded: NewCounter("rate_limit_exceeded_total", "Total number of rate limit exceeded events"),
	}
}

// Register registers all rate limit metrics
func (r *RateLimitMetrics) Register(registry MetricRegistry) {
	registry.Register(r.RateLimitHits)
	registry.Register(r.RateLimitExceeded)
}

// RecordHit records a rate limit check
func (r *RateLimitMetrics) RecordHit(exceeded bool) {
	r.RateLimitHits.Inc()
	if exceeded {
		r.RateLimitExceeded.Inc()
	}
}