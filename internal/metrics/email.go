package metrics

// EmailMetrics contains all email-related metrics
type EmailMetrics struct {
	EmailsSent       *Counter
	EmailsFailed     *Counter
	EmailQueue       *Gauge
	EmailSendLatency *Histogram
}

// NewEmailMetrics creates a new EmailMetrics instance
func NewEmailMetrics() *EmailMetrics {
	return &EmailMetrics{
		EmailsSent:       NewCounter("email_sent_total", "Total number of emails sent"),
		EmailsFailed:     NewCounter("email_failed_total", "Total number of failed email attempts"),
		EmailQueue:       NewGauge("email_queue_size", "Number of emails in queue"),
		EmailSendLatency: NewHistogram("email_send_duration_seconds", "Email send latencies in seconds"),
	}
}

// Register registers all email metrics
func (e *EmailMetrics) Register(registry MetricRegistry) {
	registry.Register(e.EmailsSent)
	registry.Register(e.EmailsFailed)
	registry.Register(e.EmailQueue)
	registry.Register(e.EmailSendLatency)
}

// RecordEmailSent records a sent email
func (e *EmailMetrics) RecordEmailSent(duration float64) {
	e.EmailsSent.Inc()
	e.EmailSendLatency.Observe(duration)
}

// RecordEmailFailed records a failed email
func (e *EmailMetrics) RecordEmailFailed() {
	e.EmailsFailed.Inc()
}

// SetQueueSize sets the current email queue size
func (e *EmailMetrics) SetQueueSize(size float64) {
	e.EmailQueue.Set(size)
}