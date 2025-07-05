package handlers

import (
	"net"
	"net/http"
	"strings"

	"github.com/n1rocket/go-auth-jwt/internal/http/request"
	"github.com/n1rocket/go-auth-jwt/internal/http/response"
	"github.com/n1rocket/go-auth-jwt/internal/service"
)

// LoginHandler handles user authentication
type LoginHandler struct {
	authService *service.AuthService
	decoder     *request.Decoder
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(authService *service.AuthService) *LoginHandler {
	return &LoginHandler{
		authService: authService,
		decoder:     request.NewDecoder(),
	}
}

// Handle processes login requests
func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request
	var req request.LoginRequest
	if err := h.decoder.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	// Get client info
	clientIP := getLoginClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Create service input
	input := service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: &clientIP,
		UserAgent: &userAgent,
	}

	// Call service
	output, err := h.authService.Login(r.Context(), input)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// Build response
	response.NewBuilder(w).Success(loginResponseData{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    output.ExpiresIn,
	})
}

// loginResponseData represents the login API response data
type loginResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// getLoginClientIP attempts to get the real client IP address for login
func getLoginClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		ips := splitIPs(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// splitIPs splits a comma-separated list of IPs
func splitIPs(ips string) []string {
	var result []string
	for _, ip := range strings.Split(ips, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			result = append(result, ip)
		}
	}
	return result
}
