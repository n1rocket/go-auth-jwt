package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/abueno/go-auth-jwt/internal/http/handlers"
)

func TestSignupRequest_JSONSerialization(t *testing.T) {
	// Test marshaling
	req := handlers.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal SignupRequest: %v", err)
	}

	// Test unmarshaling
	var decoded handlers.SignupRequest
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SignupRequest: %v", err)
	}

	if decoded.Email != req.Email {
		t.Errorf("Expected email %s, got %s", req.Email, decoded.Email)
	}
	if decoded.Password != req.Password {
		t.Errorf("Expected password %s, got %s", req.Password, decoded.Password)
	}
}

func TestLoginRequest_JSONSerialization(t *testing.T) {
	jsonStr := `{"email":"user@example.com","password":"secret123"}`
	
	var req handlers.LoginRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal LoginRequest: %v", err)
	}

	if req.Email != "user@example.com" {
		t.Errorf("Expected email user@example.com, got %s", req.Email)
	}
	if req.Password != "secret123" {
		t.Errorf("Expected password secret123, got %s", req.Password)
	}
}

func TestRefreshRequest_JSONSerialization(t *testing.T) {
	jsonStr := `{"refresh_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"}`
	
	var req handlers.RefreshRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal RefreshRequest: %v", err)
	}

	if req.RefreshToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" {
		t.Errorf("Unexpected refresh token value")
	}
}

func TestLogoutRequest_JSONSerialization(t *testing.T) {
	jsonStr := `{"refresh_token":"logout-token"}`
	
	var req handlers.LogoutRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal LogoutRequest: %v", err)
	}

	if req.RefreshToken != "logout-token" {
		t.Errorf("Expected refresh token logout-token, got %s", req.RefreshToken)
	}
}

func TestVerifyEmailRequest_JSONSerialization(t *testing.T) {
	jsonStr := `{"email":"verify@example.com","token":"verification-token-123"}`
	
	var req handlers.VerifyEmailRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal VerifyEmailRequest: %v", err)
	}

	if req.Email != "verify@example.com" {
		t.Errorf("Expected email verify@example.com, got %s", req.Email)
	}
	if req.Token != "verification-token-123" {
		t.Errorf("Expected token verification-token-123, got %s", req.Token)
	}
}

func TestLoginResponse_JSONSerialization(t *testing.T) {
	resp := handlers.LoginResponse{
		AccessToken:  "access-token-abc",
		RefreshToken: "refresh-token-xyz",
		ExpiresIn:    7200,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal LoginResponse: %v", err)
	}

	// Verify JSON field names
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := m["access_token"]; !ok {
		t.Error("Expected field 'access_token' in JSON")
	}
	if _, ok := m["refresh_token"]; !ok {
		t.Error("Expected field 'refresh_token' in JSON")
	}
	if _, ok := m["expires_in"]; !ok {
		t.Error("Expected field 'expires_in' in JSON")
	}
}

func TestUserResponse_JSONSerialization(t *testing.T) {
	resp := handlers.UserResponse{
		ID:            "user-789",
		Email:         "user@example.com",
		EmailVerified: true,
		CreatedAt:     "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal UserResponse: %v", err)
	}

	// Verify JSON field names
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := m["id"]; !ok {
		t.Error("Expected field 'id' in JSON")
	}
	if _, ok := m["email"]; !ok {
		t.Error("Expected field 'email' in JSON")
	}
	if _, ok := m["email_verified"]; !ok {
		t.Error("Expected field 'email_verified' in JSON")
	}
	if _, ok := m["created_at"]; !ok {
		t.Error("Expected field 'created_at' in JSON")
	}
}