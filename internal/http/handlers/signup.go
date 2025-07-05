package handlers

import (
	"net/http"

	"github.com/n1rocket/go-auth-jwt/internal/http/request"
	"github.com/n1rocket/go-auth-jwt/internal/http/response"
	"github.com/n1rocket/go-auth-jwt/internal/service"
)

// SignupHandler handles user registration
type SignupHandler struct {
	authService *service.AuthService
	decoder     *request.Decoder
}

// NewSignupHandler creates a new signup handler
func NewSignupHandler(authService *service.AuthService) *SignupHandler {
	return &SignupHandler{
		authService: authService,
		decoder:     request.NewDecoder(),
	}
}

// Handle processes signup requests
func (h *SignupHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request
	var req request.SignupRequest
	if err := h.decoder.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Create service input
	input := service.SignupInput{
		Email:    req.Email,
		Password: req.Password,
	}

	// Call service
	output, err := h.authService.Signup(r.Context(), input)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).
		Status(http.StatusCreated).
		Success(signupResponseData{
			UserID:  output.UserID,
			Message: "User created successfully. Please check your email to verify your account.",
		})
}

// signupResponseData represents the signup API response data
type signupResponseData struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}
