package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/metrics"
)

func TestNewExporter(t *testing.T) {
	tests := []struct {
		name             string
		config           ExporterConfig
		expectedInterval time.Duration
		expectedBuffer   int
		expectedFormat   string
	}{
		{
			name:             "default values",
			config:           ExporterConfig{},
			expectedInterval: 10 * time.Second,
			expectedBuffer:   1000,
			expectedFormat:   "json",
		},
		{
			name: "custom values",
			config: ExporterConfig{
				Interval:   5 * time.Second,
				BufferSize: 500,
				Format:     "prometheus",
			},
			expectedInterval: 5 * time.Second,
			expectedBuffer:   500,
			expectedFormat:   "prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.NewMetrics()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			exporter := NewExporter(tt.config, m, logger)

			if exporter == nil {
				t.Fatal("Expected exporter to be created")
			}
			if exporter.config.Interval != tt.expectedInterval {
				t.Errorf("Expected interval %v, got %v", tt.expectedInterval, exporter.config.Interval)
			}
			if exporter.config.BufferSize != tt.expectedBuffer {
				t.Errorf("Expected buffer size %d, got %d", tt.expectedBuffer, exporter.config.BufferSize)
			}
			if exporter.config.Format != tt.expectedFormat {
				t.Errorf("Expected format %s, got %s", tt.expectedFormat, exporter.config.Format)
			}
			if cap(exporter.buffer) != tt.expectedBuffer {
				t.Errorf("Expected buffer capacity %d, got %d", tt.expectedBuffer, cap(exporter.buffer))
			}
		})
	}
}

func TestExporter_CollectMetrics(t *testing.T) {
	config := ExporterConfig{
		BufferSize: 100,
		GlobalLabels: map[string]string{
			"env": "test",
		},
	}

	m := metrics.NewMetrics()
	// Set some test values
	m.RequestsTotal().Add(10)
	m.RequestsInFlight().Set(2.0)
	m.ActiveSessions().Set(5.0)
	m.LoginSuccess().Add(3)
	m.GoRoutines().Set(20)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	exporter := NewExporter(config, m, logger)

	hostname := GetHostname()
	exporter.collectMetrics(hostname)

	// Verify metrics were collected
	collectedCount := 0
	timeout := time.After(100 * time.Millisecond)

	for {
		select {
		case snapshot := <-exporter.buffer:
			collectedCount++

			// Verify snapshot properties
			if snapshot.Hostname != hostname {
				t.Errorf("Expected hostname %s, got %s", hostname, snapshot.Hostname)
			}
			if snapshot.Timestamp == 0 {
				t.Error("Expected timestamp to be set")
			}
			if snapshot.Labels["env"] != "test" {
				t.Error("Expected global label to be set")
			}

			// Check specific metrics
			switch snapshot.Name {
			case "http_requests_total":
				if snapshot.Type != "counter" {
					t.Errorf("Expected counter type for requests_total")
				}
				if v, ok := snapshot.Value.(int64); !ok || v != 10 {
					t.Errorf("Expected requests_total value 10, got %v", snapshot.Value)
				}
			case "http_requests_in_flight":
				if snapshot.Type != "gauge" {
					t.Errorf("Expected gauge type for requests_in_flight")
				}
			case "auth_active_sessions":
				if snapshot.Type != "gauge" {
					t.Errorf("Expected gauge type for active_sessions")
				}
			}

		case <-timeout:
			if collectedCount == 0 {
				t.Error("No metrics were collected")
			}
			return
		}
	}
}

func TestExporter_Start(t *testing.T) {
	// Create temp file for file export
	tmpFile, err := os.CreateTemp("", "metrics_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := ExporterConfig{
		Stdout:     false, // Disable stdout to avoid test output noise
		File:       tmpFile.Name(),
		Interval:   100 * time.Millisecond, // Fast interval for testing
		BufferSize: 10,
		Format:     "json",
	}

	m := metrics.NewMetrics()
	m.RequestsTotal().Inc()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	exporter := NewExporter(config, m, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start exporter
	done := make(chan error, 1)
	go func() {
		done <- exporter.Start(ctx)
	}()

	// Let it run for a bit
	time.Sleep(300 * time.Millisecond)

	// Stop exporter
	cancel()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start did not return after context cancellation")
	}

	// Check that metrics were written to file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read metrics file: %v", err)
	}

	if len(data) == 0 {
		t.Error("No metrics were written to file")
	}

	// Verify JSON format
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var snapshot MetricSnapshot
		if err := json.Unmarshal([]byte(line), &snapshot); err != nil {
			t.Errorf("Failed to unmarshal metric line: %v", err)
		}
	}
}

func TestExporter_ExportToStdout(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		snapshot MetricSnapshot
		contains []string
	}{
		{
			name:   "json format",
			format: "json",
			snapshot: MetricSnapshot{
				Name:      "test_metric",
				Type:      "counter",
				Value:     int64(42),
				Timestamp: 1234567890,
				Hostname:  "test-host",
			},
			contains: []string{`"name":"test_metric"`, `"value":42`},
		},
		{
			name:   "prometheus format",
			format: "prometheus",
			snapshot: MetricSnapshot{
				Name:      "test_metric",
				Type:      "gauge",
				Value:     float64(3.14),
				Timestamp: 1234567890,
				Hostname:  "test-host",
			},
			contains: []string{`test_metric{hostname="test-host"} 3.14 1234567890`},
		},
		{
			name:   "default format",
			format: "text",
			snapshot: MetricSnapshot{
				Name:  "test_metric",
				Value: "test_value",
			},
			contains: []string{`test_metric=test_value`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			config := ExporterConfig{
				Format:     tt.format,
				BufferSize: 1,
			}

			m := metrics.NewMetrics()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			exporter := NewExporter(config, m, logger)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start stdout exporter
			go exporter.exportToStdout(ctx)

			// Send metric
			exporter.buffer <- tt.snapshot
			time.Sleep(50 * time.Millisecond)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read output
			output, _ := io.ReadAll(r)
			outputStr := string(output)

			// Verify output
			for _, expected := range tt.contains {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %s, got: %s", expected, outputStr)
				}
			}
		})
	}
}

func TestExporter_SendHTTPBatch(t *testing.T) {
	// Create test server
	received := false
	var receivedBatch []MetricSnapshot

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type")
		}

		var batch []MetricSnapshot
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			t.Errorf("Failed to decode batch: %v", err)
		}

		received = true
		receivedBatch = batch
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := ExporterConfig{
		HTTPPush: server.URL,
	}

	m := metrics.NewMetrics()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	exporter := NewExporter(config, m, logger)

	// Create test batch
	batch := []MetricSnapshot{
		{
			Name:      "metric1",
			Type:      "counter",
			Value:     int64(10),
			Timestamp: time.Now().Unix(),
		},
		{
			Name:      "metric2",
			Type:      "gauge",
			Value:     float64(20.5),
			Timestamp: time.Now().Unix(),
		},
	}

	// Send batch
	client := &http.Client{Timeout: 1 * time.Second}
	exporter.sendHTTPBatch(client, batch)

	// Verify
	if !received {
		t.Error("HTTP batch was not received")
	}
	if len(receivedBatch) != 2 {
		t.Errorf("Expected 2 metrics in batch, got %d", len(receivedBatch))
	}
}

func TestExporter_ExportToHTTP(t *testing.T) {
	// Create test server
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := ExporterConfig{
		HTTPPush:  server.URL,
		BatchSize: 2,
	}

	m := metrics.NewMetrics()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	exporter := NewExporter(config, m, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start HTTP exporter
	go exporter.exportToHTTP(ctx)

	// Send metrics
	for i := 0; i < 3; i++ {
		exporter.buffer <- MetricSnapshot{
			Name:      fmt.Sprintf("metric%d", i),
			Value:     i,
			Timestamp: time.Now().Unix(),
		}
	}

	// Wait for batch to be sent
	time.Sleep(200 * time.Millisecond)

	// Should have sent one batch (first 2 metrics)
	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}
}

func TestExporter_BufferOverflow(t *testing.T) {
	config := ExporterConfig{
		BufferSize: 2, // Small buffer
	}

	m := metrics.NewMetrics()

	// Use a custom logger to capture warnings
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	exporter := NewExporter(config, m, logger)

	// Fill buffer beyond capacity
	for i := 0; i < 5; i++ {
		exporter.collectMetrics("test-host")
	}

	// Check that warning was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Metrics buffer full") {
		t.Error("Expected buffer full warning to be logged")
	}
}
