package middleware

import (
	"context"
	"net/http"

	"github.com/abueno/go-auth-jwt/internal/http/request"
	"github.com/abueno/go-auth-jwt/internal/http/response"
	"github.com/abueno/go-auth-jwt/internal/http/handlers"
	"github.com/abueno/go-auth-jwt/internal/token"
)

// RequireAuth is a middleware that validates JWT tokens
func RequireAuth(tokenManager *token.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		tokenString, err := request.ExtractBearerToken(r)
		if err != nil {
			response.WriteError(w, token.ErrInvalidToken)
			return
		}

		// Validate token
		claims, err := tokenManager.ValidateAccessToken(tokenString)
		if err != nil {
			response.WriteError(w, err)
			return
		}

		// Add user ID to context
		ctx := handlers.WithUserID(r.Context(), claims.UserID)
		
		// Add additional claims to context if needed
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "email_verified", claims.EmailVerified)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is a middleware that validates JWT tokens if present but doesn't require them
func OptionalAuth(tokenManager *token.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract token from Authorization header
		tokenString, err := request.ExtractBearerToken(r)
		if err != nil {
			// No token or invalid format - continue without auth
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate token
		claims, err := tokenManager.ValidateAccessToken(tokenString)
		if err != nil {
			// Invalid token - continue without auth
			next.ServeHTTP(w, r)
			return
		}

		// Add user ID to context
		ctx := handlers.WithUserID(r.Context(), claims.UserID)
		
		// Add additional claims to context
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "email_verified", claims.EmailVerified)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireVerifiedEmail is a middleware that requires the user to have a verified email
func RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if email is verified (set by RequireAuth middleware)
		emailVerified, ok := r.Context().Value("email_verified").(bool)
		if !ok || !emailVerified {
			response.WriteError(w, &emailNotVerifiedError{})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// emailNotVerifiedError is a custom error for unverified emails
type emailNotVerifiedError struct{}

func (e *emailNotVerifiedError) Error() string {
	return "email not verified"
}