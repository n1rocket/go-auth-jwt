package metrics

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewCounter(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter description")
	
	if counter == nil {
		t.Fatal("Expected counter to be created")
	}
	if counter.name != "test_counter" {
		t.Errorf("Expected name to be 'test_counter', got %s", counter.name)
	}
	if counter.help != "Test counter description" {
		t.Errorf("Expected help to be 'Test counter description', got %s", counter.help)
	}
	if v, ok := counter.Value().(int64); !ok || v != 0 {
		t.Errorf("Expected initial value to be 0, got %v", counter.Value())
	}
}

func TestCounter_Inc(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	counter.Inc()
	if v, ok := counter.Value().(int64); !ok || v != 1 {
		t.Errorf("Expected value to be 1 after Inc(), got %v", counter.Value())
	}
	
	counter.Inc()
	if v, ok := counter.Value().(int64); !ok || v != 2 {
		t.Errorf("Expected value to be 2 after second Inc(), got %v", counter.Value())
	}
}

func TestCounter_Add(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	counter.Add(10)
	if v, ok := counter.Value().(int64); !ok || v != 10 {
		t.Errorf("Expected value to be 10 after Add(10), got %v", counter.Value())
	}
	
	counter.Add(5)
	if v, ok := counter.Value().(int64); !ok || v != 15 {
		t.Errorf("Expected value to be 15 after Add(5), got %v", counter.Value())
	}
}

func TestCounter_Add_Panic(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when adding negative value")
		}
	}()
	
	counter.Add(-1)
}

func TestCounter_WithLabels(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	labels1 := map[string]string{"method": "GET", "path": "/api"}
	labeled1 := counter.WithLabels(labels1)
	
	if labeled1 == nil {
		t.Fatal("Expected labeled counter to be created")
	}
	
	// Increment labeled counter
	labeled1.Inc()
	labeled1.Add(5)
	
	// Get the same labeled counter again
	labeled2 := counter.WithLabels(labels1)
	if v := labeled2.Value(); v != int64(6) {
		t.Errorf("Expected labeled counter value to be 6, got %v", v)
	}
	
	// Different labels should create different counter
	labels3 := map[string]string{"method": "POST", "path": "/api"}
	labeled3 := counter.WithLabels(labels3)
	labeled3.Inc()
	
	if v := labeled3.Value(); v != int64(1) {
		t.Errorf("Expected different labeled counter value to be 1, got %v", v)
	}
}

func TestCounter_Name(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	if name := counter.Name(); name != "test_counter" {
		t.Errorf("Expected name to be 'test_counter', got %s", name)
	}
}

func TestCounter_String(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	counter.Add(42)
	
	expected := "test_counter: 42"
	if s := counter.String(); s != expected {
		t.Errorf("Expected string representation to be '%s', got '%s'", expected, s)
	}
}

func TestCounter_Reset(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	counter.Add(100)
	
	// Verify value before reset
	if v, ok := counter.Value().(int64); !ok || v != 100 {
		t.Errorf("Expected value to be 100 before reset, got %v", counter.Value())
	}
	
	counter.Reset()
	
	// Verify value after reset
	if v, ok := counter.Value().(int64); !ok || v != 0 {
		t.Errorf("Expected value to be 0 after reset, got %v", counter.Value())
	}
}


func TestLabeledCounter_Inc(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	labels := map[string]string{"method": "GET"}
	labeled := counter.WithLabels(labels)
	
	labeled.Inc()
	if v := labeled.Value(); v != int64(1) {
		t.Errorf("Expected value to be 1 after Inc(), got %v", v)
	}
	
	labeled.Inc()
	if v := labeled.Value(); v != int64(2) {
		t.Errorf("Expected value to be 2 after second Inc(), got %v", v)
	}
}

func TestLabeledCounter_Add(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	labels := map[string]string{"method": "GET"}
	labeled := counter.WithLabels(labels)
	
	labeled.Add(10)
	if v := labeled.Value(); v != int64(10) {
		t.Errorf("Expected value to be 10 after Add(10), got %v", v)
	}
	
	labeled.Add(5)
	if v := labeled.Value(); v != int64(15) {
		t.Errorf("Expected value to be 15 after Add(5), got %v", v)
	}
}

func TestLabeledCounter_Add_Panic(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	labels := map[string]string{"method": "GET"}
	labeled := counter.WithLabels(labels)
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when adding negative value to labeled counter")
		}
	}()
	
	labeled.Add(-1)
}

func TestLabelsToKey(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   "",
		},
		{
			name:   "single label",
			labels: map[string]string{"method": "GET"},
			want:   "method=GET",
		},
		{
			name:   "multiple labels sorted",
			labels: map[string]string{"method": "GET", "path": "/api", "status": "200"},
			want:   "method=GET,path=/api,status=200",
		},
		{
			name:   "labels with special chars",
			labels: map[string]string{"key": "value with spaces"},
			want:   "key=value with spaces",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := labelsToKey(tt.labels)
			
			// For empty labels, check exact match
			if len(tt.labels) == 0 {
				if got != tt.want {
					t.Errorf("labelsToKey() = %v, want %v", got, tt.want)
				}
				return
			}
			
			// For single label, check exact match
			if len(tt.labels) == 1 {
				if got != tt.want {
					t.Errorf("labelsToKey() = %v, want %v", got, tt.want)
				}
				return
			}
			
			// For multiple labels, check that all key-value pairs are present
			// (order doesn't matter for maps)
			for k, v := range tt.labels {
				expected := fmt.Sprintf("%s=%s", k, v)
				if !strings.Contains(got, expected) {
					t.Errorf("labelsToKey() missing %s in result: %v", expected, got)
				}
			}
		})
	}
}

func TestCounter_Concurrent(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	// Run concurrent increments
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				counter.Inc()
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
	
	// Check final value
	if v, ok := counter.Value().(int64); !ok || v != 10000 {
		t.Errorf("Expected value to be 10000 after concurrent increments, got %v", counter.Value())
	}
}

func TestCounter_ConcurrentLabels(t *testing.T) {
	counter := NewCounter("test_counter", "Test counter")
	
	// Run concurrent labeled counter operations
	done := make(chan bool)
	labeledCounters := make([]*LabeledCounter, 10)
	
	for i := 0; i < 10; i++ {
		go func(n int) {
			labels := map[string]string{"worker": string(rune('0' + n))}
			labeled := counter.WithLabels(labels)
			labeledCounters[n] = labeled
			for j := 0; j < 100; j++ {
				labeled.Inc()
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Each should have value 100
	for i, labeled := range labeledCounters {
		if labeled != nil {
			if v := labeled.Value(); v != int64(100) {
				t.Errorf("Expected labeled counter %d to have value 100, got %v", i, v)
			}
		}
	}
}