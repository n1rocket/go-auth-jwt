package metrics

// HTTPMetrics contains all HTTP-related metrics
type HTTPMetrics struct {
	RequestsTotal    *Counter
	RequestDuration  *Histogram
	RequestsInFlight *Gauge
	ResponseSize     *Histogram
}

// NewHTTPMetrics creates a new HTTPMetrics instance
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		RequestsTotal:    NewCounter("http_requests_total", "Total number of HTTP requests"),
		RequestDuration:  NewHistogram("http_request_duration_seconds", "HTTP request latencies in seconds"),
		RequestsInFlight: NewGauge("http_requests_in_flight", "Number of HTTP requests currently being processed"),
		ResponseSize:     NewHistogram("http_response_size_bytes", "HTTP response sizes in bytes"),
	}
}

// Register registers all HTTP metrics
func (h *HTTPMetrics) Register(registry MetricRegistry) {
	registry.Register(h.RequestsTotal)
	registry.Register(h.RequestDuration)
	registry.Register(h.RequestsInFlight)
	registry.Register(h.ResponseSize)
}

// RecordRequest records an HTTP request
func (h *HTTPMetrics) RecordRequest(method, path string, statusCode int, duration float64, size int64) {
	h.RequestsTotal.Inc()
	h.RequestDuration.Observe(duration)
	h.ResponseSize.Observe(float64(size))
}

// IncrementInFlight increments the in-flight requests gauge
func (h *HTTPMetrics) IncrementInFlight() {
	h.RequestsInFlight.Inc()
}

// DecrementInFlight decrements the in-flight requests gauge
func (h *HTTPMetrics) DecrementInFlight() {
	h.RequestsInFlight.Dec()
}