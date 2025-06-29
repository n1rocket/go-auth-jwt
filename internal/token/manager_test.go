package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewManager_HS256(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{
			name:    "valid secret",
			secret:  "my-secret-key",
			wantErr: false,
		},
		{
			name:    "empty secret",
			secret:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager("HS256", tt.secret, "", "", "test-issuer", 15*time.Minute)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && manager == nil {
				t.Error("NewManager() returned nil manager without error")
			}
		})
	}
}

func TestNewManager_RS256(t *testing.T) {
	// Create temporary key files for testing
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")

	// Generate test keys
	generateTestKeys(t, privateKeyPath, publicKeyPath)

	tests := []struct {
		name           string
		privateKeyPath string
		publicKeyPath  string
		wantErr        bool
	}{
		{
			name:           "valid keys",
			privateKeyPath: privateKeyPath,
			publicKeyPath:  publicKeyPath,
			wantErr:        false,
		},
		{
			name:           "missing private key",
			privateKeyPath: "",
			publicKeyPath:  publicKeyPath,
			wantErr:        true,
		},
		{
			name:           "missing public key",
			privateKeyPath: privateKeyPath,
			publicKeyPath:  "",
			wantErr:        true,
		},
		{
			name:           "non-existent private key",
			privateKeyPath: "/non/existent/private.pem",
			publicKeyPath:  publicKeyPath,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager("RS256", "", tt.privateKeyPath, tt.publicKeyPath, "test-issuer", 15*time.Minute)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && manager == nil {
				t.Error("NewManager() returned nil manager without error")
			}
		})
	}
}

func TestNewManager_UnsupportedAlgorithm(t *testing.T) {
	_, err := NewManager("HS512", "secret", "", "", "test-issuer", 15*time.Minute)
	if err == nil {
		t.Error("NewManager() should return error for unsupported algorithm")
	}
}

func TestManager_GenerateAndValidateToken_HS256(t *testing.T) {
	manager, err := NewManager("HS256", "test-secret", "", "", "test-issuer", 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	userID := "user-123"
	email := "test@example.com"
	emailVerified := true

	// Generate token
	tokenString, err := manager.GenerateAccessToken(userID, email, emailVerified)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if tokenString == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Validate token
	claims, err := manager.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	// Verify claims
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Email = %v, want %v", claims.Email, email)
	}
	if claims.EmailVerified != emailVerified {
		t.Errorf("EmailVerified = %v, want %v", claims.EmailVerified, emailVerified)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer = %v, want %v", claims.Issuer, "test-issuer")
	}
	if claims.Subject != userID {
		t.Errorf("Subject = %v, want %v", claims.Subject, userID)
	}
}

func TestManager_GenerateAndValidateToken_RS256(t *testing.T) {
	// Create temporary key files
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	generateTestKeys(t, privateKeyPath, publicKeyPath)

	manager, err := NewManager("RS256", "", privateKeyPath, publicKeyPath, "test-issuer", 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	userID := "user-456"
	email := "rs256@example.com"
	emailVerified := false

	// Generate token
	tokenString, err := manager.GenerateAccessToken(userID, email, emailVerified)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Validate token
	claims, err := manager.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	// Verify claims
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Email = %v, want %v", claims.Email, email)
	}
	if claims.EmailVerified != emailVerified {
		t.Errorf("EmailVerified = %v, want %v", claims.EmailVerified, emailVerified)
	}
}

func TestManager_ValidateAccessToken_Errors(t *testing.T) {
	manager, _ := NewManager("HS256", "test-secret", "", "", "test-issuer", 15*time.Minute)

	tests := []struct {
		name        string
		tokenString string
		wantErr     error
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     ErrInvalidToken,
		},
		{
			name:        "invalid token",
			tokenString: "invalid.token.string",
			wantErr:     ErrInvalidToken,
		},
		{
			name:        "malformed token",
			tokenString: "not-a-jwt",
			wantErr:     ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateAccessToken(tt.tokenString)
			if err == nil {
				t.Error("ValidateAccessToken() should return error")
			}
		})
	}
}

func TestManager_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	manager, err := NewManager("HS256", "test-secret", "", "", "test-issuer", 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Generate token
	tokenString, err := manager.GenerateAccessToken("user-123", "test@example.com", true)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = manager.ValidateAccessToken(tokenString)
	if err != ErrExpiredToken {
		t.Errorf("ValidateAccessToken() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestManager_ValidateAccessToken_WrongSigningMethod(t *testing.T) {
	// Create HS256 manager
	manager, _ := NewManager("HS256", "test-secret", "", "", "test-issuer", 15*time.Minute)

	// Create a token with RS256 algorithm (wrong for this manager)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": "test",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})

	// This will create an invalid signature, but that's OK for this test
	tokenString, _ := token.SignedString([]byte("wrong-key"))

	// Try to validate with HS256 manager
	_, err := manager.ValidateAccessToken(tokenString)
	if err == nil {
		t.Error("ValidateAccessToken() should return error for wrong signing method")
	}
}

func TestManager_GetPublicKey(t *testing.T) {
	// Test with HS256 manager
	hs256Manager, _ := NewManager("HS256", "test-secret", "", "", "test-issuer", 15*time.Minute)
	_, err := hs256Manager.GetPublicKey()
	if err == nil {
		t.Error("GetPublicKey() should return error for HS256 algorithm")
	}

	// Test with RS256 manager
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	generateTestKeys(t, privateKeyPath, publicKeyPath)

	rs256Manager, _ := NewManager("RS256", "", privateKeyPath, publicKeyPath, "test-issuer", 15*time.Minute)
	publicKey, err := rs256Manager.GetPublicKey()
	if err != nil {
		t.Errorf("GetPublicKey() error = %v", err)
	}
	if publicKey == nil {
		t.Error("GetPublicKey() returned nil public key")
	}
}

// Helper function to generate test RSA keys
func generateTestKeys(t *testing.T, privateKeyPath, publicKeyPath string) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Save private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateFile, err := os.Create(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create private key file: %v", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}

	// Save public key
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyPKIX,
	}
	publicFile, err := os.Create(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}
	defer publicFile.Close()

	if err := pem.Encode(publicFile, publicKeyPEM); err != nil {
		t.Fatalf("Failed to write public key: %v", err)
	}
}