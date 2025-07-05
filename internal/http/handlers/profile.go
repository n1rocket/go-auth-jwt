package handlers

import (
	"errors"
	"net/http"

	httpcontext "github.com/n1rocket/go-auth-jwt/internal/http/context"
	"github.com/n1rocket/go-auth-jwt/internal/http/request"
	"github.com/n1rocket/go-auth-jwt/internal/http/response"
	"github.com/n1rocket/go-auth-jwt/internal/service"
)

// ProfileHandler handles user profile operations
type ProfileHandler struct {
	authService *service.AuthService
	decoder     *request.Decoder
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(authService *service.AuthService) *ProfileHandler {
	return &ProfileHandler{
		authService: authService,
		decoder:     request.NewDecoder(),
	}
}

// HandleGetProfile gets the current user's profile
func (h *ProfileHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user context
	userID, ok := r.Context().Value(httpcontext.UserIDKey).(string)
	if !ok {
		response.WriteError(w, errors.New("user not found in context"))
		return
	}

	email, _ := r.Context().Value(httpcontext.UserEmailKey).(string)
	emailVerified, _ := r.Context().Value(httpcontext.UserEmailVerifiedKey).(bool)

	// Build response
	response.NewBuilder(w).Success(profileResponseData{
		UserID:        userID,
		Email:         email,
		EmailVerified: emailVerified,
	})
}

// HandleVerifyEmail processes email verification requests
func (h *ProfileHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request
	var req request.VerifyEmailRequest
	if err := h.decoder.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Create service input
	input := service.VerifyEmailInput{
		Token: req.Token,
	}

	// Call service
	if err := h.authService.VerifyEmail(r.Context(), input); err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).Success(map[string]string{
		"message": "Email verified successfully",
	})
}

// profileResponseData represents the profile API response data
type profileResponseData struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}
