package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/n1rocket/go-auth-jwt/internal/http/request"
	"github.com/n1rocket/go-auth-jwt/internal/http/response"
	"github.com/n1rocket/go-auth-jwt/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignupRequest represents the signup request payload
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignupResponse represents the signup response
type SignupResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// Signup handles user registration
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := request.ValidateJSONRequest(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Trim whitespace
	req.Email = strings.TrimSpace(req.Email)

	// Validate required fields
	validationErrors := request.ValidateRequiredFields(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if len(validationErrors) > 0 {
		response.WriteValidationError(w, validationErrors)
		return
	}

	// Call service
	output, err := h.authService.Signup(r.Context(), service.SignupInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusCreated, SignupResponse{
		UserID:  output.UserID,
		Message: "User created successfully. Please check your email to verify your account.",
	})
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := request.ValidateJSONRequest(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Trim whitespace
	req.Email = strings.TrimSpace(req.Email)

	// Validate required fields
	validationErrors := request.ValidateRequiredFields(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if len(validationErrors) > 0 {
		response.WriteValidationError(w, validationErrors)
		return
	}

	// Extract client info for refresh token metadata
	userAgent := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Call service
	output, err := h.authService.Login(r.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: &userAgent,
		IPAddress: &ipAddress,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    output.ExpiresIn,
	})
}

// RefreshRequest represents the refresh request payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := request.ValidateJSONRequest(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Validate required fields
	validationErrors := request.ValidateRequiredFields(map[string]string{
		"refresh_token": req.RefreshToken,
	})
	if len(validationErrors) > 0 {
		response.WriteValidationError(w, validationErrors)
		return
	}

	// Extract client info
	userAgent := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Call service
	output, err := h.authService.Refresh(r.Context(), service.RefreshInput{
		RefreshToken: req.RefreshToken,
		UserAgent:    &userAgent,
		IPAddress:    &ipAddress,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    output.ExpiresIn,
	})
}

// LogoutRequest represents the logout request payload
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := request.ValidateJSONRequest(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Validate required fields
	validationErrors := request.ValidateRequiredFields(map[string]string{
		"refresh_token": req.RefreshToken,
	})
	if len(validationErrors) > 0 {
		response.WriteValidationError(w, validationErrors)
		return
	}

	// Call service
	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// LogoutAll handles logout from all devices
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok {
		response.WriteError(w, http.ErrNotSupported)
		return
	}

	// Call service
	if err := h.authService.LogoutAll(r.Context(), userID); err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out from all devices successfully",
	})
}

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := request.ValidateJSONRequest(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Trim whitespace
	req.Email = strings.TrimSpace(req.Email)
	req.Token = strings.TrimSpace(req.Token)

	// Validate required fields
	validationErrors := request.ValidateRequiredFields(map[string]string{
		"email": req.Email,
		"token": req.Token,
	})
	if len(validationErrors) > 0 {
		response.WriteValidationError(w, validationErrors)
		return
	}

	// Call service
	if err := h.authService.VerifyEmail(r.Context(), service.VerifyEmailInput{
		Email: req.Email,
		Token: req.Token,
	}); err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// UserResponse represents the user information response
type UserResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     string `json:"created_at"`
}

// GetCurrentUser returns the current authenticated user's information
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok {
		response.WriteError(w, http.ErrNotSupported)
		return
	}

	// Get user from service
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Return response
	response.WriteJSON(w, http.StatusOK, UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// Context keys
type contextKey string

const (
	// UserIDContextKey is the context key for the authenticated user ID
	UserIDContextKey contextKey = "userID"
)

// WithUserID adds the user ID to the request context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDContextKey, userID)
}