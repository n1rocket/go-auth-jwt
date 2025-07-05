package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	client "github.com/n1rocket/go-auth-jwt/examples/clients/go"
)

func main() {
	// Create client with configuration
	authClient := client.NewClient(client.Config{
		BaseURL:     getEnv("AUTH_SERVICE_URL", "http://localhost:8080"),
		APIPath:     "/api/v1",
		Timeout:     30 * time.Second,
		AutoRefresh: true,
	})
	defer authClient.Close()

	// Add retry support
	authClient = client.WithRetry(authClient, 3, 1*time.Second)

	ctx := context.Background()

	// Example flow
	email := "test@example.com"
	password := "SecurePassword123!"

	// 1. Signup
	fmt.Println("1. Signing up new user...")
	if err := authClient.Signup(ctx, email, password); err != nil {
		// Check if it's an API error
		if apiErr, ok := err.(*client.APIError); ok {
			if apiErr.Code == "DUPLICATE_EMAIL" {
				fmt.Println("   User already exists, proceeding to login")
			} else {
				log.Fatalf("Signup failed: %v", apiErr)
			}
		} else {
			log.Fatalf("Signup failed: %v", err)
		}
	} else {
		fmt.Println("   ✓ Signup successful")
	}

	// 2. Login
	fmt.Println("\n2. Logging in...")
	authResp, err := authClient.Login(ctx, email, password)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Println("   ✓ Login successful")
	fmt.Printf("   Access token: %s...\n", authResp.AccessToken[:20])
	fmt.Printf("   Expires in: %d seconds\n", authResp.ExpiresIn)

	// 3. Get profile
	fmt.Println("\n3. Getting user profile...")
	profile, err := authClient.GetProfile(ctx)
	if err != nil {
		log.Fatalf("Get profile failed: %v", err)
	}
	fmt.Println("   ✓ Profile retrieved")
	fmt.Printf("   ID: %s\n", profile.ID)
	fmt.Printf("   Email: %s\n", profile.Email)
	fmt.Printf("   Verified: %v\n", profile.EmailVerified)
	fmt.Printf("   Created: %s\n", profile.CreatedAt.Format(time.RFC3339))

	// 4. Make authenticated request
	fmt.Println("\n4. Making authenticated request...")
	resp, err := authClient.AuthenticatedRequest(ctx, "GET", "/some-endpoint", nil)
	if err != nil {
		// This might fail if endpoint doesn't exist
		fmt.Printf("   Request failed (expected): %v\n", err)
	} else {
		fmt.Printf("   Response: %s\n", string(resp))
	}

	// 5. Token persistence example
	fmt.Println("\n5. Testing token persistence...")
	accessToken, refreshToken := authClient.GetTokens()
	fmt.Println("   Tokens saved")

	// Create new client and restore tokens
	newClient := client.NewClient(client.DefaultConfig())
	newClient.SetTokens(accessToken, refreshToken, 3600)
	fmt.Println("   Tokens restored to new client")

	// Verify restoration worked
	if newClient.IsAuthenticated() {
		fmt.Println("   ✓ New client is authenticated")
	}

	// 6. Wait for auto-refresh (optional)
	if false { // Set to true to test auto-refresh
		fmt.Println("\n6. Waiting for auto-refresh...")
		time.Sleep(2 * time.Minute)
		fmt.Println("   Check logs for auto-refresh")
	}

	// 7. Logout
	fmt.Println("\n7. Logging out...")
	if err := authClient.Logout(ctx); err != nil {
		log.Printf("Logout failed: %v", err)
	} else {
		fmt.Println("   ✓ Logout successful")
	}

	// Verify logout
	if !authClient.IsAuthenticated() {
		fmt.Println("   ✓ Client is no longer authenticated")
	}

	fmt.Println("\nExample completed successfully!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example of using the client in a web service
type AuthService struct {
	client *client.Client
}

func NewAuthService(authServiceURL string) *AuthService {
	return &AuthService{
		client: client.NewClient(client.Config{
			BaseURL:     authServiceURL,
			AutoRefresh: true,
		}),
	}
}

func (s *AuthService) AuthenticateUser(ctx context.Context, email, password string) (*client.UserProfile, error) {
	// Login
	_, err := s.client.Login(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Get profile
	profile, err := s.client.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

func (s *AuthService) GetAuthTokens() (accessToken, refreshToken string) {
	return s.client.GetTokens()
}

func (s *AuthService) Close() {
	s.client.Close()
}
