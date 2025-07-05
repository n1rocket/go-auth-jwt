package handlers

import (
	"errors"
	"net/http"

	httpcontext "github.com/n1rocket/go-auth-jwt/internal/http/context"
	"github.com/n1rocket/go-auth-jwt/internal/http/request"
	"github.com/n1rocket/go-auth-jwt/internal/http/response"
	"github.com/n1rocket/go-auth-jwt/internal/service"
)

// TokenHandler handles token-related operations
type TokenHandler struct {
	authService *service.AuthService
	decoder     *request.Decoder
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(authService *service.AuthService) *TokenHandler {
	return &TokenHandler{
		authService: authService,
		decoder:     request.NewDecoder(),
	}
}

// HandleRefresh processes token refresh requests
func (h *TokenHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request
	var req request.RefreshTokenRequest
	if err := h.decoder.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Create service input
	input := service.RefreshInput{
		RefreshToken: req.RefreshToken,
	}

	// Call service
	output, err := h.authService.Refresh(r.Context(), input)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).Success(refreshResponseData{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    output.ExpiresIn,
	})
}

// HandleLogout processes logout requests
func (h *TokenHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request
	var req request.RefreshTokenRequest
	if err := h.decoder.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Create service input
	input := service.LogoutInput{
		RefreshToken: req.RefreshToken,
	}

	// Call service
	if err := h.authService.Logout(r.Context(), input); err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).Success(map[string]string{
		"message": "Successfully logged out",
	})
}

// HandleLogoutAll processes logout from all devices requests
func (h *TokenHandler) HandleLogoutAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(httpcontext.UserIDKey).(string)
	if !ok {
		response.WriteError(w, errors.New("user not found in context"))
		return
	}

	// Call service
	if err := h.authService.LogoutAll(r.Context(), userID); err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).Success(map[string]string{
		"message": "Successfully logged out from all devices",
	})
}

// refreshResponseData represents the refresh API response data
type refreshResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}
