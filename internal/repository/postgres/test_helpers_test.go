package postgres

import "time"

// Helper functions for tests

// stringPtr creates a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// timePtr creates a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}