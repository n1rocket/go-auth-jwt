package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	client "github.com/n1rocket/go-auth-jwt/examples/clients/go"
)

const (
	configFileName = ".jwt-auth-cli"
	version        = "1.0.0"
)

type Config struct {
	APIBaseURL   string `json:"api_base_url"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Email        string `json:"email,omitempty"`
}

func main() {
	// Command line flags
	var (
		apiURL      = flag.String("api", "http://localhost:8080", "API base URL")
		configPath  = flag.String("config", "", "Config file path")
		showVersion = flag.Bool("version", false, "Show version")
	)

	// Subcommands
	signupCmd := flag.NewFlagSet("signup", flag.ExitOnError)
	loginCmd := flag.NewFlagSet("login", flag.ExitOnError)
	logoutCmd := flag.NewFlagSet("logout", flag.ExitOnError)
	profileCmd := flag.NewFlagSet("profile", flag.ExitOnError)
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	refreshCmd := flag.NewFlagSet("refresh", flag.ExitOnError)

	flag.Parse()

	if *showVersion {
		fmt.Printf("jwt-auth-cli version %s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	// Load config
	config := loadConfig(*configPath)
	if *apiURL != "http://localhost:8080" {
		config.APIBaseURL = *apiURL
	}

	// Create client
	authClient := client.NewClient(client.Config{
		BaseURL:     config.APIBaseURL,
		AutoRefresh: false, // Manual control in CLI
	})
	defer authClient.Close()

	// Restore tokens if available
	if config.AccessToken != "" && config.RefreshToken != "" {
		authClient.SetTokens(config.AccessToken, config.RefreshToken, 3600)
	}

	ctx := context.Background()

	switch flag.Arg(0) {
	case "signup":
		signupCmd.Parse(flag.Args()[1:])
		handleSignup(ctx, authClient, config)

	case "login":
		loginCmd.Parse(flag.Args()[1:])
		handleLogin(ctx, authClient, config)

	case "logout":
		logoutCmd.Parse(flag.Args()[1:])
		handleLogout(ctx, authClient, config)

	case "profile":
		profileCmd.Parse(flag.Args()[1:])
		handleProfile(ctx, authClient)

	case "verify":
		verifyCmd.Parse(flag.Args()[1:])
		handleVerify(ctx, authClient)

	case "refresh":
		refreshCmd.Parse(flag.Args()[1:])
		handleRefresh(ctx, authClient, config)

	default:
		fmt.Printf("Unknown command: %s\n", flag.Arg(0))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("JWT Auth CLI - Command line client for JWT Authentication Service")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  jwt-auth-cli [options] <command> [arguments]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -api string     API base URL (default \"http://localhost:8080\")")
	fmt.Println("  -config string  Config file path")
	fmt.Println("  -version        Show version")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  signup          Create a new account")
	fmt.Println("  login           Login to your account")
	fmt.Println("  logout          Logout from your account")
	fmt.Println("  profile         Show user profile")
	fmt.Println("  verify          Verify email address")
	fmt.Println("  refresh         Refresh access token")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  jwt-auth-cli signup")
	fmt.Println("  jwt-auth-cli -api https://api.example.com login")
	fmt.Println("  jwt-auth-cli profile")
}

func handleSignup(ctx context.Context, authClient *client.Client, config *Config) {
	email := promptEmail("Email: ")
	password := promptPassword("Password: ")
	confirmPassword := promptPassword("Confirm Password: ")

	if password != confirmPassword {
		fmt.Println("Error: Passwords do not match")
		os.Exit(1)
	}

	fmt.Println("Creating account...")
	if err := authClient.Signup(ctx, email, password); err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			fmt.Printf("Signup failed: %s\n", apiErr.Message)
			if apiErr.Code == "DUPLICATE_EMAIL" {
				fmt.Println("This email is already registered. Please login instead.")
			}
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Account created successfully!")
	fmt.Println("Please check your email to verify your account.")
	
	// Save email for convenience
	config.Email = email
	saveConfig(config)
}

func handleLogin(ctx context.Context, authClient *client.Client, config *Config) {
	email := config.Email
	if email == "" {
		email = promptEmail("Email: ")
	} else {
		fmt.Printf("Email [%s]: ", email)
		if input := readLine(); input != "" {
			email = input
		}
	}
	
	password := promptPassword("Password: ")

	fmt.Println("Logging in...")
	authResp, err := authClient.Login(ctx, email, password)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			fmt.Printf("Login failed: %s\n", apiErr.Message)
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Login successful!")
	
	// Save tokens and email
	config.Email = email
	config.AccessToken = authResp.AccessToken
	config.RefreshToken = authResp.RefreshToken
	saveConfig(config)
	
	// Show expiration time
	expiresAt := time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	fmt.Printf("Token expires at: %s\n", expiresAt.Format(time.RFC3339))
}

func handleLogout(ctx context.Context, authClient *client.Client, config *Config) {
	if !authClient.IsAuthenticated() {
		fmt.Println("You are not logged in.")
		return
	}

	fmt.Println("Logging out...")
	if err := authClient.Logout(ctx); err != nil {
		fmt.Printf("Warning: Logout API call failed: %v\n", err)
	}

	// Clear stored tokens
	config.AccessToken = ""
	config.RefreshToken = ""
	saveConfig(config)
	
	fmt.Println("✓ Logged out successfully!")
}

func handleProfile(ctx context.Context, authClient *client.Client) {
	if !authClient.IsAuthenticated() {
		fmt.Println("Error: You must be logged in to view your profile.")
		fmt.Println("Run 'jwt-auth-cli login' to authenticate.")
		os.Exit(1)
	}

	fmt.Println("Fetching profile...")
	profile, err := authClient.GetProfile(ctx)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 401 {
			fmt.Println("Error: Session expired. Please login again.")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("\n--- User Profile ---")
	fmt.Printf("ID:       %s\n", profile.ID)
	fmt.Printf("Email:    %s\n", profile.Email)
	fmt.Printf("Verified: %v\n", profile.EmailVerified)
	fmt.Printf("Created:  %s\n", profile.CreatedAt.Format(time.RFC3339))
}

func handleVerify(ctx context.Context, authClient *client.Client) {
	email := promptEmail("Email: ")
	token := prompt("Verification token: ")

	fmt.Println("Verifying email...")
	if err := authClient.VerifyEmail(ctx, email, token); err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			fmt.Printf("Verification failed: %s\n", apiErr.Message)
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Email verified successfully!")
}

func handleRefresh(ctx context.Context, authClient *client.Client, config *Config) {
	if !authClient.IsAuthenticated() {
		fmt.Println("Error: You must be logged in to refresh your token.")
		os.Exit(1)
	}

	fmt.Println("Refreshing token...")
	authResp, err := authClient.Refresh(ctx)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			fmt.Printf("Refresh failed: %s\n", apiErr.Message)
			if apiErr.StatusCode == 401 {
				fmt.Println("Your session has expired. Please login again.")
			}
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✓ Token refreshed successfully!")
	
	// Save new tokens
	config.AccessToken = authResp.AccessToken
	config.RefreshToken = authResp.RefreshToken
	saveConfig(config)
	
	// Show new expiration time
	expiresAt := time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	fmt.Printf("Token expires at: %s\n", expiresAt.Format(time.RFC3339))
}

// Helper functions

func loadConfig(customPath string) *Config {
	configPath := customPath
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		configPath = filepath.Join(home, configFileName)
	}

	config := &Config{
		APIBaseURL: "http://localhost:8080",
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config // Return default config if file doesn't exist
	}

	if err := json.Unmarshal(data, config); err != nil {
		fmt.Printf("Warning: Failed to parse config file: %v\n", err)
		return config
	}

	return config
}

func saveConfig(config *Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: Failed to get home directory: %v\n", err)
		return
	}

	configPath := filepath.Join(home, configFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Failed to marshal config: %v\n", err)
		return
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		fmt.Printf("Warning: Failed to save config: %v\n", err)
	}
}

func prompt(label string) string {
	fmt.Print(label)
	return readLine()
}

func promptEmail(label string) string {
	for {
		email := prompt(label)
		if strings.Contains(email, "@") && strings.Contains(email, ".") {
			return email
		}
		fmt.Println("Please enter a valid email address.")
	}
}

func promptPassword(label string) string {
	fmt.Print(label)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input
	if err != nil {
		log.Fatal(err)
	}
	
	pwd := string(password)
	if len(pwd) < 8 {
		fmt.Println("Password must be at least 8 characters long.")
		return promptPassword(label)
	}
	
	return pwd
}

func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}