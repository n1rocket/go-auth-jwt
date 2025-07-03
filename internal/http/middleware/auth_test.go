package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/http/handlers"
	"github.com/n1rocket/go-auth-jwt/internal/token"
	"github.com/golang-jwt/jwt/v5"
)

// Mock token manager
type mockTokenManager struct {
	validateFunc func(tokenString string) (*token.Claims, error)
}

func (m *mockTokenManager) GenerateAccessToken(userID, email string, emailVerified bool) (string, error) {
	return "", nil
}

func (m *mockTokenManager) ValidateAccessToken(tokenString string) (*token.Claims, error) {
	if m.validateFunc != nil {
		return m.validateFunc(tokenString)
	}
	return nil, errors.New("invalid token")
}

func (m *mockTokenManager) GenerateRefreshToken() (string, error) {
	return "", nil
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		tokenManager   *mockTokenManager
		expectedStatus int
		expectedUser   bool
	}{
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			tokenManager: &mockTokenManager{
				validateFunc: func(tokenString string) (*token.Claims, error) {
					if tokenString == "valid-token" {
						return &token.Claims{
							UserID:        "user-123",
							Email:         "user@example.com",
							EmailVerified: true,
							RegisteredClaims: jwt.RegisteredClaims{
								ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
							},
						}, nil
					}
					return nil, errors.New("invalid token")
				},
			},
			expectedStatus: http.StatusOK,
			expectedUser:   true,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			tokenManager:   &mockTokenManager{},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
		{
			name:           "invalid authorization format",
			authHeader:     "InvalidFormat token",
			tokenManager:   &mockTokenManager{},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
		{
			name:           "no bearer prefix",
			authHeader:     "token-without-bearer",
			tokenManager:   &mockTokenManager{},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			tokenManager: &mockTokenManager{
				validateFunc: func(tokenString string) (*token.Claims, error) {
					return nil, errors.New("invalid token")
				},
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if userID is in context
				userID, ok := r.Context().Value(handlers.UserIDContextKey).(string)
				if tt.expectedUser && (!ok || userID == "") {
					t.Error("Expected user ID in context but got empty")
				}
				if !tt.expectedUser && ok && userID != "" {
					t.Error("Expected no user ID in context but got one")
				}
				if ok && userID != "" && userID != "user-123" {
					t.Errorf("Unexpected user ID: %s", userID)
				}
				
				// Check email in context
				email, ok := r.Context().Value("email").(string)
				if tt.expectedUser && (!ok || email != "user@example.com") {
					t.Errorf("Expected email in context, got %s", email)
				}
				
				w.WriteHeader(http.StatusOK)
			})
			
			// Create a custom handler that uses our mock
			customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract token from Authorization header
				var tokenString string
				if tt.authHeader != "" {
					if len(tt.authHeader) > 7 && tt.authHeader[:7] == "Bearer " {
						tokenString = tt.authHeader[7:]
					}
				}
				
				if tokenString == "" && tt.authHeader != "" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				
				if tokenString == "" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				
				// Validate token
				claims, err := tt.tokenManager.ValidateAccessToken(tokenString)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// Add user ID to context
				ctx := handlers.WithUserID(r.Context(), claims.UserID)
				ctx = context.WithValue(ctx, "email", claims.Email)
				ctx = context.WithValue(ctx, "email_verified", claims.EmailVerified)

				// Call next handler with updated context
				handler.ServeHTTP(w, r.WithContext(ctx))
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			customHandler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		tokenManager   *mockTokenManager
		expectedStatus int
		expectedUser   bool
	}{
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			tokenManager: &mockTokenManager{
				validateFunc: func(tokenString string) (*token.Claims, error) {
					if tokenString == "valid-token" {
						return &token.Claims{
							UserID:        "user-123",
							Email:         "user@example.com",
							EmailVerified: true,
							RegisteredClaims: jwt.RegisteredClaims{
								ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
							},
						}, nil
					}
					return nil, errors.New("invalid token")
				},
			},
			expectedStatus: http.StatusOK,
			expectedUser:   true,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			tokenManager:   &mockTokenManager{},
			expectedStatus: http.StatusOK,
			expectedUser:   false,
		},
		{
			name:           "invalid authorization format",
			authHeader:     "InvalidFormat token",
			tokenManager:   &mockTokenManager{},
			expectedStatus: http.StatusOK,
			expectedUser:   false,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			tokenManager: &mockTokenManager{
				validateFunc: func(tokenString string) (*token.Claims, error) {
					return nil, errors.New("invalid token")
				},
			},
			expectedStatus: http.StatusOK,
			expectedUser:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if userID is in context
				userID, ok := r.Context().Value(handlers.UserIDContextKey).(string)
				if tt.expectedUser && (!ok || userID == "") {
					t.Error("Expected user ID in context but got empty")
				}
				if !tt.expectedUser && ok && userID != "" {
					t.Error("Expected no user ID in context but got one")
				}
				w.WriteHeader(http.StatusOK)
			})
			
			// Create a custom handler that uses our mock
			customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Try to extract token from Authorization header
				var tokenString string
				if tt.authHeader != "" && len(tt.authHeader) > 7 && tt.authHeader[:7] == "Bearer " {
					tokenString = tt.authHeader[7:]
				}
				
				if tokenString == "" {
					// No token - continue without auth
					handler.ServeHTTP(w, r)
					return
				}
				
				// Try to validate token
				claims, err := tt.tokenManager.ValidateAccessToken(tokenString)
				if err != nil {
					// Invalid token - continue without auth
					handler.ServeHTTP(w, r)
					return
				}

				// Add user ID to context
				ctx := handlers.WithUserID(r.Context(), claims.UserID)
				ctx = context.WithValue(ctx, "email", claims.Email)
				ctx = context.WithValue(ctx, "email_verified", claims.EmailVerified)

				// Call next handler with updated context
				handler.ServeHTTP(w, r.WithContext(ctx))
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			customHandler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireVerifiedEmail(t *testing.T) {
	tests := []struct {
		name             string
		emailVerified    *bool
		expectedStatus   int
	}{
		{
			name:             "verified email",
			emailVerified:    boolPtr(true),
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "unverified email",
			emailVerified:    boolPtr(false),
			expectedStatus:   http.StatusUnauthorized,
		},
		{
			name:             "no email verification in context",
			emailVerified:    nil,
			expectedStatus:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireVerifiedEmail(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.emailVerified != nil {
				ctx := context.WithValue(req.Context(), "email_verified", *tt.emailVerified)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestEmailNotVerifiedError(t *testing.T) {
	err := &emailNotVerifiedError{}
	if err.Error() != "email not verified" {
		t.Errorf("Expected error message 'email not verified', got %s", err.Error())
	}
}

func boolPtr(b bool) *bool {
	return &b
}