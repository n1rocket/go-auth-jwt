package request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/n1rocket/go-auth-jwt/internal/http/response"
)

// MaxRequestBodySize is the maximum allowed request body size (1MB)
const MaxRequestBodySize = 1 << 20 // 1 MB

// DecodeJSON decodes a JSON request body into the provided destination
func DecodeJSON(r *http.Request, dst interface{}) error {
	// Limit the request body size
	r.Body = http.MaxBytesReader(nil, r.Body, MaxRequestBodySize)

	// Decode the JSON
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject unknown fields

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Ensure only one JSON object was sent
	if decoder.More() {
		return fmt.Errorf("request body must contain only one JSON object")
	}

	return nil
}

// ValidateContentType validates that the request has the expected content type
func ValidateContentType(r *http.Request, expectedType string) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("Content-Type header is required")
	}

	// Extract the media type (ignore parameters like charset)
	mediaType := strings.Split(contentType, ";")[0]
	mediaType = strings.TrimSpace(mediaType)

	if mediaType != expectedType {
		return fmt.Errorf("Content-Type must be %s, got %s", expectedType, mediaType)
	}

	return nil
}

// ValidateJSONRequest validates that a request has JSON content type and decodes it
func ValidateJSONRequest(r *http.Request, dst interface{}) error {
	// Validate content type
	if err := ValidateContentType(r, "application/json"); err != nil {
		return err
	}

	// Decode JSON body
	return DecodeJSON(r, dst)
}

// ValidateRequiredFields checks if required string fields are not empty
func ValidateRequiredFields(fields map[string]string) []response.ValidationError {
	var errors []response.ValidationError

	for field, value := range fields {
		value = strings.TrimSpace(value)
		if value == "" {
			errors = append(errors, response.ValidationError{
				Field:   field,
				Message: fmt.Sprintf("%s is required", field),
				Code:    "REQUIRED_FIELD",
			})
		}
	}

	return errors
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	// Basic validation - check for @ and at least one dot after @
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format")
	}

	if !strings.Contains(parts[1], ".") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// ValidatePassword performs basic password validation
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 characters")
	}

	return nil
}

// SanitizeString removes leading/trailing whitespace and limits string length
func SanitizeString(s string, maxLength int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLength {
		s = s[:maxLength]
	}
	return s
}

// ValidateToken validates a bearer token format
func ValidateToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}

	// Check for reasonable token length
	if len(token) < 10 || len(token) > 1000 {
		return fmt.Errorf("invalid token format")
	}

	return nil
}

// ExtractBearerToken extracts the token from the Authorization header
func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header is required")
	}

	// Check for Bearer prefix
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("Authorization header must use Bearer scheme")
	}

	token := parts[1]
	if err := ValidateToken(token); err != nil {
		return "", err
	}

	return token, nil
}