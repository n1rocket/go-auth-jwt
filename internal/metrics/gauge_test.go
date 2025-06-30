package metrics

import (
	"fmt"
	"testing"
)

func TestNewGauge(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge description")
	
	if gauge == nil {
		t.Fatal("Expected gauge to be created")
	}
	if gauge.name != "test_gauge" {
		t.Errorf("Expected name to be 'test_gauge', got %s", gauge.name)
	}
	if gauge.help != "Test gauge description" {
		t.Errorf("Expected help to be 'Test gauge description', got %s", gauge.help)
	}
	if v, ok := gauge.Value().(float64); !ok || v != 0.0 {
		t.Errorf("Expected initial value to be 0.0, got %v", gauge.Value())
	}
}

func TestGauge_Set(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	gauge.Set(42.5)
	if v, ok := gauge.Value().(float64); !ok || v != 42.5 {
		t.Errorf("Expected value to be 42.5 after Set(42.5), got %v", gauge.Value())
	}
	
	gauge.Set(-10.3)
	if v, ok := gauge.Value().(float64); !ok || v != -10.3 {
		t.Errorf("Expected value to be -10.3 after Set(-10.3), got %v", gauge.Value())
	}
}

func TestGauge_Inc(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	gauge.Inc()
	if v, ok := gauge.Value().(float64); !ok || v != 1.0 {
		t.Errorf("Expected value to be 1.0 after Inc(), got %v", gauge.Value())
	}
	
	gauge.Inc()
	if v, ok := gauge.Value().(float64); !ok || v != 2.0 {
		t.Errorf("Expected value to be 2.0 after second Inc(), got %v", gauge.Value())
	}
}

func TestGauge_Dec(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	gauge.Set(10.0)
	
	gauge.Dec()
	if v, ok := gauge.Value().(float64); !ok || v != 9.0 {
		t.Errorf("Expected value to be 9.0 after Dec(), got %v", gauge.Value())
	}
	
	gauge.Dec()
	if v, ok := gauge.Value().(float64); !ok || v != 8.0 {
		t.Errorf("Expected value to be 8.0 after second Dec(), got %v", gauge.Value())
	}
}

func TestGauge_Add(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	gauge.Add(10.5)
	if v, ok := gauge.Value().(float64); !ok || v != 10.5 {
		t.Errorf("Expected value to be 10.5 after Add(10.5), got %v", gauge.Value())
	}
	
	gauge.Add(-5.3)
	if v, ok := gauge.Value().(float64); !ok || v != 5.2 {
		t.Errorf("Expected value to be 5.2 after Add(-5.3), got %v", gauge.Value())
	}
}

func TestGauge_WithLabels(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	labels1 := map[string]string{"type": "memory", "unit": "bytes"}
	labeled1 := gauge.WithLabels(labels1)
	
	if labeled1 == nil {
		t.Fatal("Expected labeled gauge to be created")
	}
	
	// Set labeled gauge
	labeled1.Set(1024.0)
	labeled1.Add(512.0)
	
	// Get the same labeled gauge again
	labeled2 := gauge.WithLabels(labels1)
	// First check that we got the same gauge instance
	if labeled2 == nil {
		t.Fatal("Expected labeled gauge to be returned")
	}
	// Check the value
	v := labeled2.Value()
	if v != 1536.0 {
		// Debug: print the key to see if it's the same
		key1 := fmt.Sprintf("%v", labels1)
		t.Errorf("Expected labeled gauge value to be 1536.0, got %v (labels: %s)", v, key1)
	}
	
	// Different labels should create different gauge
	labels3 := map[string]string{"type": "cpu", "unit": "percent"}
	labeled3 := gauge.WithLabels(labels3)
	labeled3.Set(50.5)
	
	if v := labeled3.Value(); v != 50.5 {
		t.Errorf("Expected different labeled gauge value to be 50.5, got %v", v)
	}
}

func TestGauge_Name(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	if name := gauge.Name(); name != "test_gauge" {
		t.Errorf("Expected name to be 'test_gauge', got %s", name)
	}
}

func TestGauge_String(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	gauge.Set(3.14)
	
	expected := "test_gauge: 3.14"
	if s := gauge.String(); s != expected {
		t.Errorf("Expected string representation to be '%s', got '%s'", expected, s)
	}
}


func TestLabeledGauge_Set(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	labels := map[string]string{"type": "test"}
	labeled := gauge.WithLabels(labels)
	
	labeled.Set(100.0)
	if v := labeled.Value(); v != 100.0 {
		t.Errorf("Expected value to be 100.0 after Set(100.0), got %v", v)
	}
	
	labeled.Set(-50.0)
	if v := labeled.Value(); v != -50.0 {
		t.Errorf("Expected value to be -50.0 after Set(-50.0), got %v", v)
	}
}

func TestLabeledGauge_Inc(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	labels := map[string]string{"type": "test"}
	labeled := gauge.WithLabels(labels)
	
	labeled.Inc()
	if v := labeled.Value(); v != 1.0 {
		t.Errorf("Expected value to be 1.0 after Inc(), got %v", v)
	}
	
	labeled.Inc()
	if v := labeled.Value(); v != 2.0 {
		t.Errorf("Expected value to be 2.0 after second Inc(), got %v", v)
	}
}

func TestLabeledGauge_Dec(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	labels := map[string]string{"type": "test"}
	labeled := gauge.WithLabels(labels)
	labeled.Set(5.0)
	
	labeled.Dec()
	if v := labeled.Value(); v != 4.0 {
		t.Errorf("Expected value to be 4.0 after Dec(), got %v", v)
	}
	
	labeled.Dec()
	if v := labeled.Value(); v != 3.0 {
		t.Errorf("Expected value to be 3.0 after second Dec(), got %v", v)
	}
}

func TestLabeledGauge_Add(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	labels := map[string]string{"type": "test"}
	labeled := gauge.WithLabels(labels)
	
	labeled.Add(10.0)
	if v := labeled.Value(); v != 10.0 {
		t.Errorf("Expected value to be 10.0 after Add(10.0), got %v", v)
	}
	
	labeled.Add(-3.0)
	if v := labeled.Value(); v != 7.0 {
		t.Errorf("Expected value to be 7.0 after Add(-3.0), got %v", v)
	}
}

func TestGauge_Concurrent(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	// Run concurrent operations
	done := make(chan bool)
	
	// Incrementers
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				gauge.Inc()
			}
			done <- true
		}()
	}
	
	// Decrementers
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				gauge.Dec()
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
	
	// Since we have equal increments and decrements, value should be 0
	if v, ok := gauge.Value().(float64); !ok || v != 0.0 {
		t.Errorf("Expected value to be 0.0 after concurrent inc/dec, got %v", gauge.Value())
	}
}

func TestGauge_ConcurrentLabels(t *testing.T) {
	gauge := NewGauge("test_gauge", "Test gauge")
	
	// Run concurrent labeled gauge operations
	done := make(chan bool)
	labeledGauges := make([]*LabeledGauge, 10)
	
	for i := 0; i < 10; i++ {
		go func(n int) {
			labels := map[string]string{"worker": string(rune('0' + n))}
			labeled := gauge.WithLabels(labels)
			labeledGauges[n] = labeled
			labeled.Set(float64(n * 10))
			for j := 0; j < 100; j++ {
				labeled.Add(1.0)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Each should have value n*10 + 100
	for i, labeled := range labeledGauges {
		if labeled != nil {
			expected := float64(i*10 + 100)
			if v := labeled.Value(); v != expected {
				t.Errorf("Expected labeled gauge %d to have value %f, got %v", i, expected, v)
			}
		}
	}
}