package email

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestMockService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mock := NewMockService(logger)
	
	t.Run("Send", func(t *testing.T) {
		ctx := context.Background()
		email := Email{
			To:      "test@example.com",
			Subject: "Test Subject",
			Body:    "Test Body",
		}
		
		err := mock.Send(ctx, email)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}
		
		// Check that email was stored
		emails := mock.GetSentEmails()
		if len(emails) != 1 {
			t.Errorf("Expected 1 email, got %d", len(emails))
		}
		
		if emails[0].To != email.To {
			t.Errorf("Expected To = %s, got %s", email.To, emails[0].To)
		}
	})
	
	t.Run("FailNext", func(t *testing.T) {
		mock.Clear()
		mock.FailNext()
		
		ctx := context.Background()
		email := Email{
			To:      "test@example.com",
			Subject: "Test",
			Body:    "Test",
		}
		
		// First send should fail
		err := mock.Send(ctx, email)
		if err == nil {
			t.Error("Expected error on first send")
		}
		
		// Second send should succeed
		err = mock.Send(ctx, email)
		if err != nil {
			t.Errorf("Second send error = %v", err)
		}
	})
	
	t.Run("GetLastEmail", func(t *testing.T) {
		mock.Clear()
		
		// No emails yet
		last, ok := mock.GetLastEmail()
		if ok {
			t.Error("Expected no emails")
		}
		
		// Send emails
		emails := []Email{
			{To: "first@example.com", Subject: "First"},
			{To: "second@example.com", Subject: "Second"},
			{To: "last@example.com", Subject: "Last"},
		}
		
		ctx := context.Background()
		for _, e := range emails {
			mock.Send(ctx, e)
		}
		
		last, ok = mock.GetLastEmail()
		if !ok {
			t.Fatal("Expected to get last email")
		}
		
		if last.To != "last@example.com" {
			t.Errorf("Expected last email to = last@example.com, got %s", last.To)
		}
	})
	
	t.Run("CountEmails", func(t *testing.T) {
		mock.Clear()
		
		if count := mock.CountEmails(); count != 0 {
			t.Errorf("Expected 0 emails, got %d", count)
		}
		
		ctx := context.Background()
		for i := 0; i < 5; i++ {
			mock.Send(ctx, Email{
				To:      "test@example.com",
				Subject: "Test",
			})
		}
		
		if count := mock.CountEmails(); count != 5 {
			t.Errorf("Expected 5 emails, got %d", count)
		}
	})
	
	t.Run("FindEmail", func(t *testing.T) {
		mock.Clear()
		
		ctx := context.Background()
		emails := []Email{
			{To: "alice@example.com", Subject: "Welcome"},
			{To: "bob@example.com", Subject: "Reset Password"},
			{To: "charlie@example.com", Subject: "Welcome"},
		}
		
		for _, e := range emails {
			mock.Send(ctx, e)
		}
		
		// Find by recipient
		found, ok := mock.FindEmail("bob@example.com")
		
		if !ok {
			t.Fatal("Expected to find bob's email")
		}
		
		if found.Subject != "Reset Password" {
			t.Errorf("Expected subject 'Reset Password', got '%s'", found.Subject)
		}
		
		// Find first email with Welcome subject
		found, ok = mock.FindEmail("alice@example.com")
		
		if !ok {
			t.Fatal("Expected to find alice's email")
		}
		
		if found.Subject != "Welcome" {
			t.Errorf("Expected subject 'Welcome', got '%s'", found.Subject)
		}
		
		// Find non-existent
		found, ok = mock.FindEmail("nonexistent@example.com")
		
		if ok {
			t.Error("Expected not to find non-existent email")
		}
	})
	
	t.Run("Clear", func(t *testing.T) {
		ctx := context.Background()
		
		// Add some emails
		for i := 0; i < 3; i++ {
			mock.Send(ctx, Email{To: "test@example.com", Subject: "Test"})
		}
		
		if count := mock.CountEmails(); count == 0 {
			t.Error("Expected emails before clear")
		}
		
		mock.Clear()
		
		if count := mock.CountEmails(); count != 0 {
			t.Errorf("Expected 0 emails after clear, got %d", count)
		}
	})
	
	t.Run("Concurrent access", func(t *testing.T) {
		mock.Clear()
		ctx := context.Background()
		
		// Send emails concurrently
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(i int) {
				email := Email{
					To:      "test@example.com",
					Subject: "Concurrent Test",
					Body:    "Test",
				}
				mock.Send(ctx, email)
				done <- true
			}(i)
		}
		
		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
		
		// Check count
		if count := mock.CountEmails(); count != 10 {
			t.Errorf("Expected 10 emails, got %d", count)
		}
	})
	
	t.Run("Send with attachments", func(t *testing.T) {
		mock.Clear()
		ctx := context.Background()
		
		email := Email{
			To:      "test@example.com",
			Subject: "With Attachments",
			Body:    "See attached",
			Attachments: []Attachment{
				{
					Filename: "test.pdf",
					Content:  []byte("PDF content"),
					MimeType: "application/pdf",
				},
			},
		}
		
		err := mock.Send(ctx, email)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}
		
		sent, ok := mock.GetLastEmail()
		if !ok {
			t.Fatal("Expected to get last email")
		}
		if len(sent.Attachments) != 1 {
			t.Errorf("Expected 1 attachment, got %d", len(sent.Attachments))
		}
	})
}

func TestMockService_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mock := NewMockService(logger)
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	email := Email{
		To:      "test@example.com",
		Subject: "Test",
		Body:    "Test",
	}
	
	// Should still succeed as mock doesn't check context
	err := mock.Send(ctx, email)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
	
	// Verify email was sent
	if count := mock.CountEmails(); count != 1 {
		t.Errorf("Expected 1 email, got %d", count)
	}
}