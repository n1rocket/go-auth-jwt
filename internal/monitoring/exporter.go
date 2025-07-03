package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/metrics"
)

// ExporterConfig holds exporter configuration
type ExporterConfig struct {
	// Export targets
	Stdout     bool
	File       string
	HTTPPush   string
	PushGateway string

	// Export settings
	Interval   time.Duration
	Format     string // "json", "prometheus", "influx"
	BatchSize  int
	BufferSize int

	// Labels to add to all metrics
	GlobalLabels map[string]string
}

// Exporter handles metrics export
type Exporter struct {
	config  ExporterConfig
	metrics *metrics.Metrics
	logger  *slog.Logger
	buffer  chan MetricSnapshot
	stopCh  chan struct{}
}

// MetricSnapshot represents a point-in-time metric value
type MetricSnapshot struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Value     interface{}            `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp int64                  `json:"timestamp"`
	Hostname  string                 `json:"hostname"`
}

// NewExporter creates a new metrics exporter
func NewExporter(config ExporterConfig, m *metrics.Metrics, logger *slog.Logger) *Exporter {
	if config.Interval == 0 {
		config.Interval = 10 * time.Second
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.Format == "" {
		config.Format = "json"
	}

	return &Exporter{
		config:  config,
		metrics: m,
		logger:  logger,
		buffer:  make(chan MetricSnapshot, config.BufferSize),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the exporter
func (e *Exporter) Start(ctx context.Context) error {
	// Start collector
	go e.collect(ctx)

	// Start exporters based on configuration
	if e.config.Stdout {
		go e.exportToStdout(ctx)
	}
	if e.config.File != "" {
		go e.exportToFile(ctx)
	}
	if e.config.HTTPPush != "" {
		go e.exportToHTTP(ctx)
	}
	if e.config.PushGateway != "" {
		go e.exportToPushGateway(ctx)
	}

	<-ctx.Done()
	close(e.stopCh)
	return nil
}

// collect periodically collects metrics
func (e *Exporter) collect(ctx context.Context) {
	ticker := time.NewTicker(e.config.Interval)
	defer ticker.Stop()

	hostname := GetHostname()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.collectMetrics(hostname)
		}
	}
}

// collectMetrics collects current metric values
func (e *Exporter) collectMetrics(hostname string) {
	timestamp := time.Now().Unix()

	// Collect all metrics
	snapshots := []MetricSnapshot{
		// HTTP metrics
		{
			Name:      "http_requests_total",
			Type:      "counter",
			Value:     e.metrics.RequestsTotal.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		{
			Name:      "http_requests_in_flight",
			Type:      "gauge",
			Value:     e.metrics.RequestsInFlight.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		// Auth metrics
		{
			Name:      "auth_active_sessions",
			Type:      "gauge",
			Value:     e.metrics.ActiveSessions.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		{
			Name:      "auth_login_success_total",
			Type:      "counter",
			Value:     e.metrics.LoginSuccess.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		{
			Name:      "auth_login_failure_total",
			Type:      "counter",
			Value:     e.metrics.LoginFailure.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		// System metrics
		{
			Name:      "go_goroutines",
			Type:      "gauge",
			Value:     e.metrics.GoRoutines.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
		{
			Name:      "go_memory_allocated_bytes",
			Type:      "gauge",
			Value:     e.metrics.MemoryAllocated.Value(),
			Labels:    e.config.GlobalLabels,
			Timestamp: timestamp,
			Hostname:  hostname,
		},
	}

	// Send to buffer
	for _, snapshot := range snapshots {
		select {
		case e.buffer <- snapshot:
		default:
			e.logger.Warn("Metrics buffer full, dropping metric",
				slog.String("metric", snapshot.Name))
		}
	}
}

// exportToStdout exports metrics to stdout
func (e *Exporter) exportToStdout(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-e.buffer:
			switch e.config.Format {
			case "json":
				data, _ := json.Marshal(snapshot)
				fmt.Println(string(data))
			case "prometheus":
				fmt.Printf("%s{hostname=\"%s\"} %v %d\n",
					snapshot.Name, snapshot.Hostname, snapshot.Value, snapshot.Timestamp)
			default:
				fmt.Printf("%s=%v\n", snapshot.Name, snapshot.Value)
			}
		}
	}
}

// exportToFile exports metrics to a file
func (e *Exporter) exportToFile(ctx context.Context) {
	file, err := os.OpenFile(e.config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		e.logger.Error("Failed to open metrics file",
			slog.String("file", e.config.File),
			slog.String("error", err.Error()))
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-e.buffer:
			if err := encoder.Encode(snapshot); err != nil {
				e.logger.Error("Failed to write metric to file",
					slog.String("error", err.Error()))
			}
		}
	}
}

// exportToHTTP exports metrics via HTTP push
func (e *Exporter) exportToHTTP(ctx context.Context) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	batch := make([]MetricSnapshot, 0, e.config.BatchSize)

	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-e.buffer:
			batch = append(batch, snapshot)

			if len(batch) >= e.config.BatchSize {
				e.sendHTTPBatch(client, batch)
				batch = batch[:0]
			}
		case <-time.After(5 * time.Second):
			if len(batch) > 0 {
				e.sendHTTPBatch(client, batch)
				batch = batch[:0]
			}
		}
	}
}

// sendHTTPBatch sends a batch of metrics via HTTP
func (e *Exporter) sendHTTPBatch(client *http.Client, batch []MetricSnapshot) {
	data, err := json.Marshal(batch)
	if err != nil {
		e.logger.Error("Failed to marshal metrics batch",
			slog.String("error", err.Error()))
		return
	}

	resp, err := client.Post(e.config.HTTPPush, "application/json", 
		bytes.NewReader(data))
	if err != nil {
		e.logger.Error("Failed to push metrics",
			slog.String("url", e.config.HTTPPush),
			slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		e.logger.Error("Metrics push failed",
			slog.String("url", e.config.HTTPPush),
			slog.Int("status", resp.StatusCode))
	}
}

// exportToPushGateway exports metrics to Prometheus Push Gateway
func (e *Exporter) exportToPushGateway(ctx context.Context) {
	// Implementation for Prometheus Push Gateway
	// This would format metrics in Prometheus exposition format
	// and push to the configured gateway
	e.logger.Info("Push Gateway exporter not implemented")
}