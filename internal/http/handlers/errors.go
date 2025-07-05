package handlers

import "errors"

// Common errors
var (
	// ErrUnauthorized is returned when the user is not authenticated
	ErrUnauthorized = errors.New("unauthorized")

	// ErrMissingAuthHeader is returned when the Authorization header is missing
	ErrMissingAuthHeader = errors.New("missing authorization header")
)
