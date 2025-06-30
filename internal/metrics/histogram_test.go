package metrics

import (
	"testing"
)

func TestNewHistogram(t *testing.T) {
	histogram := NewHistogram("test_histogram", "Test histogram description")
	
	if histogram == nil {
		t.Fatal("Expected histogram to be created")
	}
	if histogram.name != "test_histogram" {
		t.Errorf("Expected name to be 'test_histogram', got %s", histogram.name)
	}
	if histogram.help != "Test histogram description" {
		t.Errorf("Expected help to be 'Test histogram description', got %s", histogram.help)
	}
	// Check that default buckets are used
	if len(histogram.buckets) == 0 {
		t.Error("Expected histogram to have default buckets")
	}
	
	// Verify bucket counts are initialized to 0
	for i, count := range histogram.counts {
		if count != 0 {
			t.Errorf("Expected bucket %d count to be 0, got %d", i, count)
		}
	}
	
	if histogram.count != 0 {
		t.Errorf("Expected count to be 0, got %d", histogram.count)
	}
}

func TestNewHistogramWithBuckets(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0, 2.5, 5.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram description", buckets)
	
	if histogram == nil {
		t.Fatal("Expected histogram to be created")
	}
	if len(histogram.buckets) != len(buckets) {
		t.Errorf("Expected %d buckets, got %d", len(buckets), len(histogram.buckets))
	}
}

func TestHistogram_Observe(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0, 2.5, 5.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	// Observe values that fall into different buckets
	histogram.Observe(0.05)  // <= 0.1
	histogram.Observe(0.3)   // <= 0.5
	histogram.Observe(0.7)   // <= 1.0
	histogram.Observe(1.5)   // <= 2.5
	histogram.Observe(3.0)   // <= 5.0
	histogram.Observe(10.0)  // > 5.0 (inf bucket)
	
	// Check bucket counts (current implementation uses non-cumulative buckets)
	expectedCounts := []uint64{1, 1, 1, 1, 1, 1} // Each bucket gets one value
	for i, expected := range expectedCounts {
		if histogram.counts[i] != expected {
			t.Errorf("Expected bucket %d count to be %d, got %d", i, expected, histogram.counts[i])
		}
	}
	
	// Check count
	if histogram.count != 6 {
		t.Errorf("Expected count to be 6, got %d", histogram.count)
	}
}

func TestHistogram_WithLabels(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	labels1 := map[string]string{"endpoint": "/api/users", "method": "GET"}
	labeled1 := histogram.WithLabels(labels1)
	
	if labeled1 == nil {
		t.Fatal("Expected labeled histogram to be created")
	}
	
	// Observe values in labeled histogram
	labeled1.Observe(0.2)
	labeled1.Observe(0.8)
	
	// Get the same labeled histogram again and observe more
	labeled2 := histogram.WithLabels(labels1)
	labeled2.Observe(1.0)
	
	// Different labels should create different histogram
	labels3 := map[string]string{"endpoint": "/api/users", "method": "POST"}
	labeled3 := histogram.WithLabels(labels3)
	labeled3.Observe(0.5)
}

func TestHistogram_Value(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	histogram.Observe(0.2)
	histogram.Observe(0.4)
	histogram.Observe(0.8)
	
	value := histogram.Value()
	m, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{} type, got %T", value)
	}
	
	// Check count
	if count, ok := m["count"].(uint64); !ok || count != 3 {
		t.Errorf("Expected count to be 3, got %v", m["count"])
	}
	
	// Check sum
	expectedSum := 0.2 + 0.4 + 0.8
	if sum, ok := m["sum"].(float64); !ok || (sum < expectedSum-0.01 || sum > expectedSum+0.01) {
		t.Errorf("Expected sum to be approximately %f, got %v", expectedSum, m["sum"])
	}
	
	// Check buckets
	bucketList, ok := m["buckets"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected buckets to be []map[string]interface{}, got %T", m["buckets"])
	}
	
	// Check specific bucket counts (cumulative)
	expectedBuckets := map[float64]uint64{
		0.1: 0, // No values <= 0.1
		0.5: 2, // Values 0.2, 0.4 <= 0.5
		1.0: 3, // All values <= 1.0
	}
	
	// Convert list to map for easier checking
	bucketMap := make(map[float64]uint64)
	for _, b := range bucketList {
		// Handle both float64 and string (for +Inf)
		var le float64
		switch v := b["le"].(type) {
		case float64:
			le = v
		case string:
			if v == "+Inf" {
				continue // Skip +Inf for this test
			}
		}
		count := b["count"].(uint64)
		bucketMap[le] = count
	}
	
	for bucket, expectedCount := range expectedBuckets {
		if count, ok := bucketMap[bucket]; !ok || count != expectedCount {
			t.Errorf("Bucket %f: expected count %d, got %d", bucket, expectedCount, count)
		}
	}
}

func TestHistogram_Name(t *testing.T) {
	histogram := NewHistogram("test_histogram", "Test histogram")
	
	if name := histogram.Name(); name != "test_histogram" {
		t.Errorf("Expected name to be 'test_histogram', got %s", name)
	}
}

func TestHistogram_String(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	histogram.Observe(0.2)
	histogram.Observe(0.4)
	
	s := histogram.String()
	expected := "test_histogram: count=2, sum=0.60"
	if s != expected {
		t.Errorf("Expected string representation to be '%s', got '%s'", expected, s)
	}
}


func TestLabeledHistogram_Observe(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	labels := map[string]string{"type": "test"}
	labeled := histogram.WithLabels(labels)
	
	// Just test that we can observe values
	labeled.Observe(0.2)
	labeled.Observe(0.7)
	labeled.Observe(1.5)
	
	// The labeled histogram doesn't expose its internal state directly,
	// so we just verify that observation doesn't panic
}


func TestHistogram_Concurrent(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0, 5.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	// Run concurrent observations
	done := make(chan bool)
	observationsPerGoroutine := 100
	numGoroutines := 50
	
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			for j := 0; j < observationsPerGoroutine; j++ {
				// Observe values between 0 and 1
				value := float64(j%10) / 10.0
				histogram.Observe(value)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Check final count
	expectedCount := uint64(numGoroutines * observationsPerGoroutine)
	if count := histogram.Count(); count != expectedCount {
		t.Errorf("Expected count to be %d, got %d", expectedCount, count)
	}
}

func TestHistogram_ConcurrentLabels(t *testing.T) {
	buckets := []float64{0.1, 0.5, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	// Run concurrent labeled histogram operations
	done := make(chan bool)
	labeledHistograms := make([]*LabeledHistogram, 10)
	
	for i := 0; i < 10; i++ {
		go func(n int) {
			labels := map[string]string{"worker": string(rune('0' + n))}
			labeled := histogram.WithLabels(labels)
			labeledHistograms[n] = labeled
			for j := 0; j < 100; j++ {
				labeled.Observe(float64(j%10) / 10.0)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// We can't check the count of labeled histograms directly,
	// so just verify the code runs without panicking
}

func TestHistogram_EdgeCases(t *testing.T) {
	// Test with no buckets (should still have +Inf bucket)
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", []float64{})
	histogram.Observe(1.0)
	histogram.Observe(100.0)
	
	if count := histogram.Count(); count != 2 {
		t.Errorf("Expected count to be 2, got %d", count)
	}
}

func TestHistogram_NegativeValues(t *testing.T) {
	buckets := []float64{-1.0, 0.0, 1.0}
	histogram := NewHistogramWithBuckets("test_histogram", "Test histogram", buckets)
	
	histogram.Observe(-2.0)  // < -1.0
	histogram.Observe(-0.5) // <= 0.0
	histogram.Observe(0.5)  // <= 1.0
	histogram.Observe(2.0)  // > 1.0
	
	// Check that we have the right total count
	if count := histogram.Count(); count != 4 {
		t.Errorf("Expected count to be 4, got %d", count)
	}
}