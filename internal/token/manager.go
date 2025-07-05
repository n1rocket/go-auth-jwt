package token

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidSigningMethod is returned when the signing method is invalid
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)

// Claims represents the JWT claims
type Claims struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

// Manager handles JWT token operations
type Manager struct {
	algorithm      string
	secret         []byte
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	issuer         string
	accessTokenTTL time.Duration
}

// NewManager creates a new token manager
func NewManager(algorithm, secret, privateKeyPath, publicKeyPath, issuer string, accessTokenTTL time.Duration) (*Manager, error) {
	m := &Manager{
		algorithm:      algorithm,
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
	}

	switch algorithm {
	case "HS256":
		if secret == "" {
			return nil, fmt.Errorf("secret is required for HS256 algorithm")
		}
		m.secret = []byte(secret)

	case "RS256":
		if privateKeyPath == "" || publicKeyPath == "" {
			return nil, fmt.Errorf("private and public key paths are required for RS256 algorithm")
		}

		// Load private key
		privateKeyData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		m.privateKey = privateKey

		// Load public key
		publicKeyData, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key: %w", err)
		}

		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
		m.publicKey = publicKey

	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return m, nil
}

// GenerateAccessToken generates a new access token
func (m *Manager) GenerateAccessToken(userID, email string, emailVerified bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:        userID,
		Email:         email,
		EmailVerified: emailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	var token *jwt.Token
	switch m.algorithm {
	case "HS256":
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	case "RS256":
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", m.algorithm)
	}

	// Add key ID header for RS256
	if m.algorithm == "RS256" {
		token.Header["kid"] = "default"
	}

	tokenString, err := token.SignedString(m.getSigningKey())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		switch m.algorithm {
		case "HS256":
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidSigningMethod
			}
		case "RS256":
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, ErrInvalidSigningMethod
			}
		default:
			return nil, ErrInvalidSigningMethod
		}

		return m.getVerificationKey(), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetPublicKey returns the public key for RS256 algorithm
func (m *Manager) GetPublicKey() (*rsa.PublicKey, error) {
	if m.algorithm != "RS256" {
		return nil, fmt.Errorf("public key is only available for RS256 algorithm")
	}
	return m.publicKey, nil
}

// GetJWKS returns the JSON Web Key Set for the public keys
func (m *Manager) GetJWKS() (map[string]interface{}, error) {
	if m.algorithm != "RS256" {
		return nil, fmt.Errorf("JWKS is only available for RS256 algorithm")
	}

	// This is a simplified JWKS response
	// In production, you would want to properly encode the public key
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "default",
				"alg": "RS256",
				"n":   "", // Base64 URL encoded modulus
				"e":   "", // Base64 URL encoded exponent
			},
		},
	}

	return jwks, nil
}

// getSigningKey returns the key used for signing tokens
func (m *Manager) getSigningKey() interface{} {
	switch m.algorithm {
	case "HS256":
		return m.secret
	case "RS256":
		return m.privateKey
	default:
		return nil
	}
}

// getVerificationKey returns the key used for verifying tokens
func (m *Manager) getVerificationKey() interface{} {
	switch m.algorithm {
	case "HS256":
		return m.secret
	case "RS256":
		return m.publicKey
	default:
		return nil
	}
}
