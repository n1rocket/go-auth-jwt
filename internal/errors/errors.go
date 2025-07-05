package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeAuthentication indicates an authentication error
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeAuthorization indicates an authorization error
	ErrorTypeAuthorization ErrorType = "authorization"
	// ErrorTypeNotFound indicates a resource not found error
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeConflict indicates a conflict error
	ErrorTypeConflict ErrorType = "conflict"
	// ErrorTypeInternal indicates an internal server error
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeRateLimit indicates a rate limit error
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeBadRequest indicates a bad request error
	ErrorTypeBadRequest ErrorType = "bad_request"
)

// AppError represents an application-specific error
type AppError struct {
	Type    ErrorType
	Message string
	Code    string
	Details interface{}
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}

// NewError creates a new AppError
func NewError(errType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Code:    string(errType),
	}
}

// WithCode adds a specific error code
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithDetails adds error details
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// Is checks if the error is of a specific type
func Is(err error, errType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

// GetHTTPStatus returns the appropriate HTTP status code for an error type
func GetHTTPStatus(errType ErrorType) int {
	switch errType {
	case ErrorTypeValidation, ErrorTypeBadRequest:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypeAuthorization:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Common errors
var (
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = NewError(ErrorTypeAuthentication, "invalid email or password").WithCode("INVALID_CREDENTIALS")

	// ErrUnauthorized is returned when a user is not authenticated
	ErrUnauthorized = NewError(ErrorTypeAuthentication, "authentication required").WithCode("UNAUTHORIZED")

	// ErrForbidden is returned when a user lacks permission
	ErrForbidden = NewError(ErrorTypeAuthorization, "insufficient permissions").WithCode("FORBIDDEN")

	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = NewError(ErrorTypeNotFound, "user not found").WithCode("USER_NOT_FOUND")

	// ErrUserExists is returned when trying to create a user that already exists
	ErrUserExists = NewError(ErrorTypeConflict, "user with this email already exists").WithCode("USER_EXISTS")

	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = NewError(ErrorTypeAuthentication, "invalid or expired token").WithCode("INVALID_TOKEN")

	// ErrTokenExpired is returned when a token has expired
	ErrTokenExpired = NewError(ErrorTypeAuthentication, "token has expired").WithCode("TOKEN_EXPIRED")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = NewError(ErrorTypeRateLimit, "rate limit exceeded").WithCode("RATE_LIMIT_EXCEEDED")
)

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
