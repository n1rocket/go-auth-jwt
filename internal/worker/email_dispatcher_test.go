package worker

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/abueno/go-auth-jwt/internal/email"
)

func TestEmailDispatcher(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := email.NewMockService(logger)
	
	config := Config{
		Workers:     3,
		QueueSize:   10,
		MaxRetries:  2,
		RetryDelay:  10 * time.Millisecond,
		SendTimeout: 1 * time.Second,
	}
	
	dispatcher := NewEmailDispatcher(mockService, config, logger)
	
	// Start the dispatcher
	dispatcher.Start()
	defer dispatcher.Stop(5 * time.Second)
	
	t.Run("enqueue and process emails", func(t *testing.T) {
		// Clear any previous emails
		mockService.Clear()
		
		// Enqueue some emails
		emails := []email.Email{
			{To: "user1@example.com", Subject: "Test 1", Body: "Body 1"},
			{To: "user2@example.com", Subject: "Test 2", Body: "Body 2"},
			{To: "user3@example.com", Subject: "Test 3", Body: "Body 3"},
		}
		
		for _, e := range emails {
			if err := dispatcher.Enqueue(e); err != nil {
				t.Fatalf("Failed to enqueue email: %v", err)
			}
		}
		
		// Wait for processing
		time.Sleep(100 * time.Millisecond)
		
		// Check that all emails were sent
		sentEmails := mockService.GetSentEmails()
		if len(sentEmails) != len(emails) {
			t.Errorf("Expected %d emails, got %d", len(emails), len(sentEmails))
		}
		
		// Verify each email
		for _, original := range emails {
			found := false
			for _, sent := range sentEmails {
				if sent.To == original.To && sent.Subject == original.Subject {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Email to %s not found in sent emails", original.To)
			}
		}
	})
	
	t.Run("retry on failure", func(t *testing.T) {
		mockService.Clear()
		
		// Make the next send fail
		mockService.FailNext()
		
		testEmail := email.Email{
			To:      "retry@example.com",
			Subject: "Retry Test",
			Body:    "This should be retried",
		}
		
		if err := dispatcher.Enqueue(testEmail); err != nil {
			t.Fatalf("Failed to enqueue email: %v", err)
		}
		
		// Wait for retries
		time.Sleep(200 * time.Millisecond)
		
		// Should eventually succeed after retry
		sentEmails := mockService.GetSentEmails()
		if len(sentEmails) != 1 {
			t.Errorf("Expected 1 email after retry, got %d", len(sentEmails))
		}
	})
	
	t.Run("queue full error", func(t *testing.T) {
		// Create a dispatcher with no workers to prevent processing
		noWorkerConfig := Config{
			Workers:     0,
			QueueSize:   5,
			MaxRetries:  2,
			RetryDelay:  10 * time.Millisecond,
			SendTimeout: 1 * time.Second,
		}
		
		noWorkerDispatcher := NewEmailDispatcher(mockService, noWorkerConfig, logger)
		noWorkerDispatcher.Start()
		defer noWorkerDispatcher.Stop(1 * time.Second)
		
		// Fill the queue
		for i := 0; i < noWorkerConfig.QueueSize; i++ {
			e := email.Email{
				To:      "test@example.com",
				Subject: "Queue Test",
				Body:    "Test",
			}
			err := noWorkerDispatcher.Enqueue(e)
			if err != nil {
				t.Errorf("Failed to enqueue email %d: %v", i, err)
			}
		}
		
		// Next enqueue should fail
		e := email.Email{
			To:      "test@example.com",
			Subject: "Queue Full Test",
			Body:    "This should fail",
		}
		err := noWorkerDispatcher.Enqueue(e)
		if err == nil {
			t.Error("Expected queue full error")
		}
	})
	
	t.Run("concurrent enqueue", func(t *testing.T) {
		mockService.Clear()
		
		// Wait for queue to clear
		time.Sleep(100 * time.Millisecond)
		
		var wg sync.WaitGroup
		emailCount := 20
		
		for i := 0; i < emailCount; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				e := email.Email{
					To:      "concurrent@example.com",
					Subject: "Concurrent Test",
					Body:    "Test",
				}
				dispatcher.Enqueue(e)
			}(i)
		}
		
		wg.Wait()
		
		// Wait for processing
		time.Sleep(200 * time.Millisecond)
		
		// Check stats
		stats := dispatcher.GetStats()
		if stats.Workers != config.Workers {
			t.Errorf("Expected %d workers, got %d", config.Workers, stats.Workers)
		}
	})
}

func TestEmailDispatcher_Stop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := email.NewMockService(logger)
	
	config := DefaultConfig()
	config.Workers = 2
	
	dispatcher := NewEmailDispatcher(mockService, config, logger)
	dispatcher.Start()
	
	// Enqueue an email
	testEmail := email.Email{
		To:      "stop@example.com",
		Subject: "Stop Test",
		Body:    "Test",
	}
	dispatcher.Enqueue(testEmail)
	
	// Stop with timeout
	err := dispatcher.Stop(2 * time.Second)
	if err != nil {
		t.Errorf("Failed to stop dispatcher: %v", err)
	}
	
	// Verify dispatcher is stopped
	stats := dispatcher.GetStats()
	if stats.Running {
		t.Error("Dispatcher should not be running after stop")
	}
}

func TestEmailDispatcher_EnqueueWithContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := email.NewMockService(logger)
	
	config := DefaultConfig()
	dispatcher := NewEmailDispatcher(mockService, config, logger)
	dispatcher.Start()
	defer dispatcher.Stop(2 * time.Second)
	
	t.Run("context cancelled", func(t *testing.T) {
		// Create a dispatcher with a full queue to ensure context cancellation is detected
		fullQueueConfig := Config{
			Workers:     0, // No workers to prevent processing
			QueueSize:   1,
			MaxRetries:  2,
			RetryDelay:  10 * time.Millisecond,
			SendTimeout: 1 * time.Second,
		}
		fullDispatcher := NewEmailDispatcher(mockService, fullQueueConfig, logger)
		fullDispatcher.Start()
		defer fullDispatcher.Stop(1 * time.Second)
		
		// Fill the queue
		fullDispatcher.Enqueue(email.Email{To: "filler@example.com", Subject: "Filler", Body: "Fill queue"})
		
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		testEmail := email.Email{
			To:      "cancelled@example.com",
			Subject: "Cancelled",
			Body:    "Test",
		}
		
		err := fullDispatcher.EnqueueWithContext(ctx, testEmail)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
	
	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		
		// Wait for timeout
		time.Sleep(5 * time.Millisecond)
		
		testEmail := email.Email{
			To:      "timeout@example.com",
			Subject: "Timeout",
			Body:    "Test",
		}
		
		err := dispatcher.EnqueueWithContext(ctx, testEmail)
		if err == nil {
			t.Error("Expected context deadline exceeded error")
		}
	})
}