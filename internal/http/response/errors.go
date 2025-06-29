package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/abueno/go-auth-jwt/internal/domain"
	"github.com/abueno/go-auth-jwt/internal/token"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// WriteError writes an error response to the client
func WriteError(w http.ResponseWriter, err error) {
	var errorResponse ErrorResponse
	var statusCode int

	// Check for JSON parsing errors first
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "failed to decode JSON") || 
		   strings.Contains(errStr, "Content-Type") ||
		   strings.Contains(errStr, "unexpected EOF") ||
		   strings.Contains(errStr, "invalid character") {
			statusCode = http.StatusBadRequest
			errorResponse = ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request format",
				Code:    "INVALID_REQUEST",
			}
			WriteJSON(w, statusCode, errorResponse)
			return
		}
	}
	
	// Map domain errors to HTTP status codes
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		statusCode = http.StatusNotFound
		errorResponse = ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
			Code:    "USER_NOT_FOUND",
		}
	case errors.Is(err, domain.ErrDuplicateEmail):
		statusCode = http.StatusConflict
		errorResponse = ErrorResponse{
			Error:   "conflict",
			Message: "Email already exists",
			Code:    "DUPLICATE_EMAIL",
		}
	case errors.Is(err, domain.ErrInvalidEmail):
		statusCode = http.StatusBadRequest
		errorResponse = ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid email format",
			Code:    "INVALID_EMAIL",
		}
	case errors.Is(err, domain.ErrWeakPassword):
		statusCode = http.StatusBadRequest
		errorResponse = ErrorResponse{
			Error:   "validation_error",
			Message: "Password does not meet requirements",
			Code:    "WEAK_PASSWORD",
			Details: map[string]string{
				"requirements": "Password must be at least 8 characters long",
			},
		}
	case errors.Is(err, domain.ErrInvalidCredentials):
		statusCode = http.StatusUnauthorized
		errorResponse = ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid email or password",
			Code:    "INVALID_CREDENTIALS",
		}
	case errors.Is(err, domain.ErrInvalidToken):
		statusCode = http.StatusUnauthorized
		errorResponse = ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired token",
			Code:    "INVALID_TOKEN",
		}
	case errors.Is(err, domain.ErrEmailNotVerified):
		statusCode = http.StatusForbidden
		errorResponse = ErrorResponse{
			Error:   "forbidden",
			Message: "Email not verified",
			Code:    "EMAIL_NOT_VERIFIED",
		}
	case errors.Is(err, token.ErrInvalidToken):
		statusCode = http.StatusUnauthorized
		errorResponse = ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid token format",
			Code:    "INVALID_TOKEN_FORMAT",
		}
	case errors.Is(err, token.ErrExpiredToken):
		statusCode = http.StatusUnauthorized
		errorResponse = ErrorResponse{
			Error:   "unauthorized",
			Message: "Token has expired",
			Code:    "EXPIRED_TOKEN",
		}
	default:
		statusCode = http.StatusInternalServerError
		errorResponse = ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
			Code:    "INTERNAL_ERROR",
		}
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// WriteJSON writes a JSON response to the client
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// ValidationError represents a validation error with field-specific details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, errors []ValidationError) {
	errorResponse := ErrorResponse{
		Error:   "validation_error",
		Message: "Request validation failed",
		Code:    "VALIDATION_FAILED",
		Details: make(map[string]string),
	}

	// Add field-specific errors to details
	for _, ve := range errors {
		errorResponse.Details[ve.Field] = ve.Message
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(errorResponse)
}