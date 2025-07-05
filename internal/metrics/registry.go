package metrics

// MetricRegistry defines the interface for metric registration
type MetricRegistry interface {
	Register(metric Metric)
}