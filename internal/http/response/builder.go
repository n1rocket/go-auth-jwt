package response

import (
	"encoding/json"
	"net/http"
)

// Builder provides a fluent interface for building HTTP responses
type Builder struct {
	w          http.ResponseWriter
	statusCode int
	headers    map[string]string
}

// NewBuilder creates a new response builder
func NewBuilder(w http.ResponseWriter) *Builder {
	return &Builder{
		w:          w,
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}

// Status sets the HTTP status code
func (b *Builder) Status(code int) *Builder {
	b.statusCode = code
	return b
}

// Header adds a header to the response
func (b *Builder) Header(key, value string) *Builder {
	b.headers[key] = value
	return b
}

// JSON sends a JSON response
func (b *Builder) JSON(data interface{}) error {
	b.headers["Content-Type"] = "application/json"

	// Write headers
	for key, value := range b.headers {
		b.w.Header().Set(key, value)
	}

	// Write status code
	b.w.WriteHeader(b.statusCode)

	// Write body
	if data != nil {
		return json.NewEncoder(b.w).Encode(data)
	}

	return nil
}

// Success sends a success response with data
func (b *Builder) Success(data interface{}) error {
	return b.Status(http.StatusOK).JSON(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// Created sends a created response with data
func (b *Builder) Created(data interface{}) error {
	return b.Status(http.StatusCreated).JSON(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a no content response
func (b *Builder) NoContent() error {
	b.w.WriteHeader(http.StatusNoContent)
	return nil
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}
