package handlers

import (
	"net/http"

	"github.com/n1rocket/go-auth-jwt/internal/http/response"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// Health handles the health check endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// Ready handles the readiness check endpoint
func Ready(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual service checks (database, etc.)
	response.WriteJSON(w, http.StatusOK, ReadyResponse{
		Status: "ready",
		Services: map[string]string{
			"database": "ok",
			"auth":     "ok",
		},
	})
}
