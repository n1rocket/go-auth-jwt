package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()

	if m == nil {
		t.Fatal("Expected metrics to be created")
	}

	// Check that all metrics are initialized
	if m.RequestsTotal() == nil {
		t.Error("Expected RequestsTotal to be initialized")
	}
	if m.RequestsInFlight() == nil {
		t.Error("Expected RequestsInFlight to be initialized")
	}
	if m.RequestDuration() == nil {
		t.Error("Expected RequestDuration to be initialized")
	}
	if m.ResponseSize() == nil {
		t.Error("Expected ResponseSize to be initialized")
	}
	if m.ActiveSessions() == nil {
		t.Error("Expected ActiveSessions to be initialized")
	}
	if m.LoginSuccess() == nil {
		t.Error("Expected LoginSuccess to be initialized")
	}
	if m.LoginFailure() == nil {
		t.Error("Expected LoginFailure to be initialized")
	}
	if m.SignupSuccess() == nil {
		t.Error("Expected SignupSuccess to be initialized")
	}
	if m.SignupFailure() == nil {
		t.Error("Expected SignupFailure to be initialized")
	}
	if m.TokensIssued() == nil {
		t.Error("Expected TokensIssued to be initialized")
	}
	if m.TokensRefreshed() == nil {
		t.Error("Expected TokensRefreshed to be initialized")
	}
	if m.TokensRevoked() == nil {
		t.Error("Expected TokensRevoked to be initialized")
	}
	if m.EmailsSent() == nil {
		t.Error("Expected EmailsSent to be initialized")
	}
	if m.EmailsFailed() == nil {
		t.Error("Expected EmailsFailed to be initialized")
	}
	if m.EmailQueue() == nil {
		t.Error("Expected EmailQueue to be initialized")
	}
	if m.DBConnections() == nil {
		t.Error("Expected DBConnections to be initialized")
	}
	if m.DBQueriesTotal() == nil {
		t.Error("Expected DBQueriesTotal to be initialized")
	}
	if m.DBQueryDuration() == nil {
		t.Error("Expected DBQueryDuration to be initialized")
	}
	if m.DBErrors() == nil {
		t.Error("Expected DBErrors to be initialized")
	}
	if m.GoRoutines() == nil {
		t.Error("Expected GoRoutines to be initialized")
	}
	if m.MemoryAllocated() == nil {
		t.Error("Expected MemoryAllocated to be initialized")
	}
	if m.MemoryTotal() == nil {
		t.Error("Expected MemoryTotal to be initialized")
	}
	if m.GCPauses() == nil {
		t.Error("Expected GCPauses to be initialized")
	}

	// Check registry
	if len(m.registry) == 0 {
		t.Error("Expected metrics to be registered")
	}
}

func TestMetrics_Start(t *testing.T) {
	m := NewMetrics()
	ctx, cancel := context.WithCancel(context.Background())

	// Start metrics collection
	done := make(chan bool)
	go func() {
		m.Start(ctx)
		done <- true
	}()

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Start to finish
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Start did not return after context cancellation")
	}
}

func TestMetrics_Stop(t *testing.T) {
	m := NewMetrics()

	// First stop should work without panic
	m.Stop()

	// Second stop should also work without panic (channel already closed)
	m.Stop()
}

func TestMetrics_updateSystemMetrics(t *testing.T) {
	m := NewMetrics()

	// Update system metrics
	m.System.Update()

	// Check goroutines
	goRoutines := m.GoRoutines().Value().(float64)
	actualGoRoutines := float64(runtime.NumGoroutine())
	// Allow some variance as goroutines can change during test
	if goRoutines < 1 || goRoutines > actualGoRoutines*2 {
		t.Errorf("Expected goroutines to be around %f, got %f", actualGoRoutines, goRoutines)
	}

	// Check memory allocated
	memAllocated := m.MemoryAllocated().Value().(float64)
	if memAllocated <= 0 {
		t.Error("Expected memory allocated to be > 0")
	}

	// Check memory total
	memTotal := m.MemoryTotal().Value().(float64)
	if memTotal <= 0 {
		t.Error("Expected memory total to be > 0")
	}

	// Memory total should be >= memory allocated
	if memTotal < memAllocated {
		t.Errorf("Expected memory total (%f) to be >= memory allocated (%f)", memTotal, memAllocated)
	}
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := NewMetrics()

	// Record some requests
	m.RecordHTTPRequest("GET", "/api/users", "200", 100*time.Millisecond, 1024)
	m.RecordHTTPRequest("POST", "/api/users", "201", 200*time.Millisecond, 2048)
	m.RecordHTTPRequest("GET", "/api/users/123", "404", 50*time.Millisecond, 512)

	// Check base counter
	if v, ok := m.RequestsTotal().Value().(int64); !ok || v != 3 {
		t.Errorf("Expected RequestsTotal to be 3, got %v", m.RequestsTotal().Value())
	}
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	m := NewMetrics()

	// Record successful queries
	m.RecordDBQuery("SELECT", 10*time.Millisecond, nil)
	m.RecordDBQuery("INSERT", 20*time.Millisecond, nil)

	// Record failed query
	m.RecordDBQuery("UPDATE", 5*time.Millisecond, errTest)

	// Check base counter
	if v, ok := m.DBQueriesTotal().Value().(int64); !ok || v != 3 {
		t.Errorf("Expected DBQueriesTotal to be 3, got %v", m.DBQueriesTotal().Value())
	}

	// Check errors
	if v, ok := m.DBErrors().Value().(int64); !ok || v != 1 {
		t.Errorf("Expected DBErrors to be 1, got %v", m.DBErrors().Value())
	}
}

func TestMetrics_RecordEmailSent(t *testing.T) {
	m := NewMetrics()

	// Record successful emails
	m.RecordEmailSent("verification", 100*time.Millisecond, nil)
	m.RecordEmailSent("notification", 150*time.Millisecond, nil)

	// Record failed email
	m.RecordEmailSent("verification", 50*time.Millisecond, errTest)

	// Check labeled counters (base counter remains 0 when using labels)
	// The method uses labeled counters, so base counter should be 0
	if v, ok := m.EmailsSent().Value().(int64); !ok || v != 0 {
		t.Errorf("Expected base EmailsSent to be 0, got %v", m.EmailsSent().Value())
	}

	if v, ok := m.EmailsFailed().Value().(int64); !ok || v != 0 {
		t.Errorf("Expected base EmailsFailed to be 0, got %v", m.EmailsFailed().Value())
	}

	// Check labeled counters
	verificationSent := m.EmailsSent().WithLabels(map[string]string{"type": "verification"})
	if v := verificationSent.Value(); v != 1 {
		t.Errorf("Expected verification emails sent to be 1, got %v", v)
	}

	notificationSent := m.EmailsSent().WithLabels(map[string]string{"type": "notification"})
	if v := notificationSent.Value(); v != 1 {
		t.Errorf("Expected notification emails sent to be 1, got %v", v)
	}

	verificationFailed := m.EmailsFailed().WithLabels(map[string]string{"type": "verification"})
	if v := verificationFailed.Value(); v != 1 {
		t.Errorf("Expected verification emails failed to be 1, got %v", v)
	}
}

func TestMetrics_Handler(t *testing.T) {
	m := NewMetrics()

	// Set some metric values
	m.RequestsTotal().Add(100)
	m.ActiveSessions().Set(25.0)
	m.EmailQueue().Set(5.0)

	// Debug: check if metrics are actually set
	t.Logf("RequestsTotal value before handler: %v", m.RequestsTotal().Value())
	t.Logf("RequestsTotal name: %v", m.RequestsTotal().Name())
	t.Logf("Registry size: %d", len(m.registry))

	// Create test request
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := m.Handler()
	handler.ServeHTTP(rr, req)

	// Check response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}

	// Parse response
	body := rr.Body.String()
	t.Logf("Response body: %s", body)
	var metrics map[string]interface{}
	if err := json.Unmarshal([]byte(body), &metrics); err != nil {
		t.Fatalf("Failed to unmarshal response: %v (body: %s)", err, body)
	}

	// Check some metrics
	if v, ok := metrics["http_requests_total"].(float64); !ok || v != 100 {
		t.Errorf("Expected http_requests_total to be 100, got %v", metrics["http_requests_total"])
	}

	if v, ok := metrics["auth_active_sessions"].(float64); !ok || v != 25.0 {
		t.Errorf("Expected auth_active_sessions to be 25, got %v", metrics["auth_active_sessions"])
	}

	if v, ok := metrics["email_queue_size"].(float64); !ok || v != 5.0 {
		t.Errorf("Expected email_queue_size to be 5, got %v", metrics["email_queue_size"])
	}
}

func TestMetrics_PrometheusHandler(t *testing.T) {
	m := NewMetrics()

	// Set some metric values
	m.RequestsTotal().Add(100)
	m.ActiveSessions().Set(25.0)
	// Observe on base histogram to test export
	m.RequestDuration().Observe(0.1)

	// Create test request
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := m.PrometheusHandler()
	handler.ServeHTTP(rr, req)

	// Check response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "text/plain; version=0.0.4"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}

	// Check response body
	body := rr.Body.String()
	t.Logf("Prometheus response body:\n%s", body)

	// Should contain HELP lines
	if !strings.Contains(body, "# HELP http_requests_total") {
		t.Error("Expected HELP line for http_requests_total")
	}

	// Should contain TYPE lines
	if !strings.Contains(body, "# TYPE http_requests_total counter") {
		t.Error("Expected TYPE line for http_requests_total")
	}

	// Should contain metric values
	if !strings.Contains(body, "http_requests_total 100") {
		t.Error("Expected http_requests_total value")
	}

	if !strings.Contains(body, "auth_active_sessions 25") {
		t.Error("Expected auth_active_sessions value")
	}

	// Should contain histogram buckets with values
	if !strings.Contains(body, "http_request_duration_seconds_bucket{le=\"0.25\"} 1") {
		t.Error("Expected histogram bucket with count 1 for le=0.25")
	}
	if !strings.Contains(body, "http_request_duration_seconds_count 1") {
		t.Error("Expected histogram count to be 1")
	}
}

func TestMetrics_collectSystemMetrics_goroutine(t *testing.T) {
	m := NewMetrics()

	// collectSystemMetrics is already running from NewMetrics
	// Force an immediate update
	m.System.Update()

	// Check that metrics are being updated
	initialGoRoutines := m.GoRoutines().Value().(float64)

	// Create some goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			time.Sleep(200 * time.Millisecond)
			done <- true
		}()
	}

	// Force another update to capture new goroutines
	time.Sleep(50 * time.Millisecond) // Let goroutines start
	m.System.Update()

	// Check that goroutine count increased
	updatedGoRoutines := m.GoRoutines().Value().(float64)
	if updatedGoRoutines <= initialGoRoutines {
		t.Errorf("Expected goroutine count to increase from %f, got %f", initialGoRoutines, updatedGoRoutines)
	}

	// Clean up
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stop the metrics collector
	m.Stop()
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	m := NewMetrics()

	// Run concurrent operations
	done := make(chan bool)

	// HTTP requests
	go func() {
		for i := 0; i < 100; i++ {
			m.RecordHTTPRequest("GET", "/test", "200", 10*time.Millisecond, 1024)
		}
		done <- true
	}()

	// DB queries
	go func() {
		for i := 0; i < 100; i++ {
			m.RecordDBQuery("SELECT", 5*time.Millisecond, nil)
		}
		done <- true
	}()

	// Email operations
	go func() {
		for i := 0; i < 100; i++ {
			m.RecordEmailSent("test", 20*time.Millisecond, nil)
		}
		done <- true
	}()

	// Active requests
	go func() {
		for i := 0; i < 50; i++ {
			m.RequestsInFlight().Inc()
			time.Sleep(1 * time.Millisecond)
			m.RequestsInFlight().Dec()
		}
		done <- true
	}()

	// Registry access
	go func() {
		for i := 0; i < 100; i++ {
			_ = m.Handler()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify final state
	if v, ok := m.RequestsTotal().Value().(int64); !ok || v != 100 {
		t.Errorf("Expected RequestsTotal to be 100, got %v", m.RequestsTotal().Value())
	}

	if v, ok := m.DBQueriesTotal().Value().(int64); !ok || v != 100 {
		t.Errorf("Expected DBQueriesTotal to be 100, got %v", m.DBQueriesTotal().Value())
	}

	// RecordEmailSent uses labeled counters, so check the labeled counter
	testEmailsSent := m.EmailsSent().WithLabels(map[string]string{"type": "test"})
	if v := testEmailsSent.Value(); v != 100 {
		t.Errorf("Expected test emails sent to be 100, got %v", v)
	}
}

// Test helper
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string {
	return "test error"
}
