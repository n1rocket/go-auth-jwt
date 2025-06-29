package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abueno/go-auth-jwt/internal/http/handlers"
	"github.com/abueno/go-auth-jwt/internal/http/middleware"
	"github.com/abueno/go-auth-jwt/internal/service"
	"github.com/abueno/go-auth-jwt/internal/token"
)

// Routes configures and returns the HTTP routes
func Routes(authService *service.AuthService, tokenManager *token.Manager) http.Handler {
	mux := http.NewServeMux()
	logger := slog.Default()

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Create rate limiters
	authLimiter := middleware.RateLimit(middleware.AuthEndpointLimiter, logger)
	apiLimiter := middleware.RateLimit(middleware.APIEndpointLimiter, logger)

	// Public routes with strict rate limiting
	mux.Handle("POST /api/v1/auth/signup", authLimiter(http.HandlerFunc(authHandler.Signup)))
	mux.Handle("POST /api/v1/auth/login", authLimiter(http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /api/v1/auth/refresh", authLimiter(http.HandlerFunc(authHandler.Refresh)))
	mux.Handle("POST /api/v1/auth/verify-email", authLimiter(http.HandlerFunc(authHandler.VerifyEmail)))

	// Protected routes with API rate limiting
	mux.Handle("POST /api/v1/auth/logout", 
		apiLimiter(middleware.RequireAuth(tokenManager, http.HandlerFunc(authHandler.Logout))))
	mux.Handle("POST /api/v1/auth/logout-all", 
		apiLimiter(middleware.RequireAuth(tokenManager, http.HandlerFunc(authHandler.LogoutAll))))
	mux.Handle("GET /api/v1/auth/me", 
		apiLimiter(middleware.RequireAuth(tokenManager, http.HandlerFunc(authHandler.GetCurrentUser))))

	// Health check
	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("GET /ready", handlers.Ready)

	// Configure CORS
	corsConfig := middleware.DefaultCORSConfig()
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		// Development mode - allow all origins
		corsConfig.AllowedOrigins = []string{"*"}
	} else {
		// Production mode - restrict origins
		corsConfig.AllowedOrigins = []string{
			"https://yourdomain.com",
			"https://app.yourdomain.com",
		}
	}

	// Configure security headers
	securityConfig := middleware.APISecurityConfig()

	// Add common middleware
	handler := middleware.RequestID(mux)
	handler = middleware.Logger(handler)
	handler = middleware.Recover(handler)
	handler = middleware.NewCORS(corsConfig)(handler)
	handler = middleware.SecurityHeaders(securityConfig)(handler)

	return handler
}