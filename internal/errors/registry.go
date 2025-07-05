package errors

import (
	"errors"
	"net/http"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Success bool                   `json:"success"`
	Error   ErrorDetail            `json:"error"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Registry manages error mappings and responses
type Registry struct {
	handlers map[ErrorType]ErrorHandler
}

// ErrorHandler handles specific error types
type ErrorHandler func(err error) (int, ErrorResponse)

// NewRegistry creates a new error registry
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[ErrorType]ErrorHandler),
	}

	// Register default handlers
	r.RegisterDefaults()

	return r
}

// Register registers an error handler for a specific error type
func (r *Registry) Register(errType ErrorType, handler ErrorHandler) {
	r.handlers[errType] = handler
}

// RegisterDefaults registers default error handlers
func (r *Registry) RegisterDefaults() {
	// Validation errors
	r.Register(ErrorTypeValidation, func(err error) (int, ErrorResponse) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    appErr.Code,
					Message: appErr.Message,
					Details: appErr.Details,
				},
			}
		}
		return http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		}
	})

	// Authentication errors
	r.Register(ErrorTypeAuthentication, func(err error) (int, ErrorResponse) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    appErr.Code,
					Message: appErr.Message,
				},
			}
		}
		return http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "AUTHENTICATION_ERROR",
				Message: "Authentication failed",
			},
		}
	})

	// Not found errors
	r.Register(ErrorTypeNotFound, func(err error) (int, ErrorResponse) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    appErr.Code,
					Message: appErr.Message,
				},
			}
		}
		return http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "Resource not found",
			},
		}
	})

	// Conflict errors
	r.Register(ErrorTypeConflict, func(err error) (int, ErrorResponse) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    appErr.Code,
					Message: appErr.Message,
				},
			}
		}
		return http.StatusConflict, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONFLICT",
				Message: "Resource conflict",
			},
		}
	})

	// Rate limit errors
	r.Register(ErrorTypeRateLimit, func(err error) (int, ErrorResponse) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return http.StatusTooManyRequests, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    appErr.Code,
					Message: appErr.Message,
					Details: appErr.Details,
				},
			}
		}
		return http.StatusTooManyRequests, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "RATE_LIMIT_EXCEEDED",
				Message: "Too many requests",
			},
		}
	})

	// Internal errors
	r.Register(ErrorTypeInternal, func(err error) (int, ErrorResponse) {
		return http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "An internal error occurred",
			},
		}
	})
}

// Handle processes an error and returns the appropriate HTTP status and response
func (r *Registry) Handle(err error) (int, ErrorResponse) {
	if err == nil {
		return http.StatusOK, ErrorResponse{Success: true}
	}

	// Check if it's an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		if handler, ok := r.handlers[appErr.Type]; ok {
			return handler(err)
		}
	}

	// Default to internal error
	return http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
	}
}

// DefaultRegistry is the default error registry
var DefaultRegistry = NewRegistry()
