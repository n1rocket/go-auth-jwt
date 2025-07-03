package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/n1rocket/go-auth-jwt/internal/email"
	"github.com/n1rocket/go-auth-jwt/internal/http/middleware"
	"github.com/n1rocket/go-auth-jwt/internal/worker"
)

// Example demonstrates how to use Phase 4 features
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Example 1: Email Service Setup
	emailExample(logger)

	// Example 2: Worker Pool Setup
	workerPoolExample(logger)

	// Example 3: Rate Limiting Configuration
	rateLimitingExample(logger)

	// Example 4: CORS Configuration
	corsExample()

	// Example 5: Security Headers
	securityHeadersExample()
}

func emailExample(logger *slog.Logger) {
	logger.Info("=== Email Service Example ===")

	// Example SMTP configuration (would be used in production)
	// smtpConfig := email.SMTPConfig{
	// 	Host:        "smtp.gmail.com",
	// 	Port:        587,
	// 	Username:    "your-email@gmail.com",
	// 	Password:    "your-app-password",
	// 	FromAddress: "noreply@example.com",
	// 	FromName:    "Auth Service",
	// 	TLSEnabled:  true,
	// 	Timeout:     30 * time.Second,
	// }

	// Create email service (in production, you would use this)
	// emailService := email.NewSMTPService(smtpConfig, logger)

	// Prepare email template data
	templateData := email.TemplateData{
		BaseURL:           "https://example.com",
		AppName:           "My Auth App",
		SupportEmail:      "support@example.com",
		RecipientEmail:    "user@example.com",
		VerificationToken: "abc123xyz",
		VerificationURL:   "https://example.com/verify?token=abc123xyz",
		ExpirationHours:   24,
	}

	// Render email from template
	verificationEmail, err := email.RenderTemplate(email.VerificationEmailTemplate, templateData)
	if err != nil {
		logger.Error("Failed to render email", "error", err)
		return
	}

	logger.Info("Email rendered successfully",
		"to", verificationEmail.To,
		"subject", verificationEmail.Subject,
	)

	// In production, you would send the email:
	// ctx := context.Background()
	// err = emailService.Send(ctx, verificationEmail)
}

func workerPoolExample(logger *slog.Logger) {
	logger.Info("=== Worker Pool Example ===")

	// Create mock email service for demo
	mockService := email.NewMockService(logger)

	// Configure worker pool
	workerConfig := worker.Config{
		Workers:     5,
		QueueSize:   100,
		MaxRetries:  3,
		RetryDelay:  5 * time.Second,
		SendTimeout: 30 * time.Second,
	}

	// Create email dispatcher
	dispatcher := worker.NewEmailDispatcher(mockService, workerConfig, logger)

	// Start dispatcher
	dispatcher.Start()
	defer dispatcher.Stop(10 * time.Second)

	// Enqueue emails
	for i := 0; i < 10; i++ {
		testEmail := email.Email{
			To:      "user@example.com",
			Subject: "Test Email",
			Body:    "This is a test email from the worker pool",
		}

		if err := dispatcher.Enqueue(testEmail); err != nil {
			logger.Error("Failed to enqueue email", "error", err)
		}
	}

	// Check stats
	stats := dispatcher.GetStats()
	logger.Info("Worker pool stats",
		"workers", stats.Workers,
		"queue_size", stats.QueueSize,
		"queue_capacity", stats.QueueCapacity,
		"running", stats.Running,
	)

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Check sent emails
	sentEmails := mockService.GetSentEmails()
	logger.Info("Emails sent", "count", len(sentEmails))
}

func rateLimitingExample(logger *slog.Logger) {
	logger.Info("=== Rate Limiting Example ===")

	// Example 1: Auth endpoint rate limiter (strict)
	authLimiter := middleware.NewRateLimiter(
		middleware.AuthEndpointLimiter,
		logger,
	)

	// Simulate requests
	testKey := "192.168.1.1"
	for i := 0; i < 10; i++ {
		allowed, remaining, resetTime := authLimiter.Allow(testKey)
		logger.Info("Auth rate limit check",
			"request", i+1,
			"allowed", allowed,
			"remaining", remaining,
			"reset_in", time.Until(resetTime).Seconds(),
		)
		
		if !allowed {
			break
		}
	}

	// Example 2: Custom rate limiter
	customConfig := middleware.RateLimitConfig{
		Rate:    60,        // 60 requests
		Burst:   10,        // burst of 10
		Window:  time.Hour, // per hour
		KeyFunc: middleware.PathKeyFunc(),
	}

	customLimiter := middleware.NewRateLimiter(customConfig, logger)
	
	// Test with path-based key
	pathKey := "192.168.1.1:/api/v1/users"
	allowed, _, _ := customLimiter.Allow(pathKey)
	logger.Info("Custom rate limit check",
		"key", pathKey,
		"allowed", allowed,
	)
}

func corsExample() {
	slog.Info("=== CORS Configuration Example ===")

	// Development CORS (allow all)
	devCORS := middleware.DefaultCORSConfig()
	slog.Info("Dev CORS config",
		"allowed_origins", devCORS.AllowedOrigins,
		"allow_credentials", devCORS.AllowCredentials,
	)

	// Production CORS (strict)
	prodCORS := middleware.StrictCORSConfig([]string{
		"https://app.example.com",
		"https://admin.example.com",
	})
	slog.Info("Production CORS config",
		"allowed_origins", prodCORS.AllowedOrigins,
		"allowed_methods", prodCORS.AllowedMethods,
		"max_age", prodCORS.MaxAge,
	)

	// Custom CORS
	customCORS := middleware.CORSConfig{
		AllowedOrigins: []string{"https://*.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		ExposedHeaders: []string{"X-Request-ID", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge: 3600,
	}
	slog.Info("Custom CORS config",
		"allowed_origins", customCORS.AllowedOrigins,
		"exposed_headers", customCORS.ExposedHeaders,
	)
}

func securityHeadersExample() {
	slog.Info("=== Security Headers Example ===")

	// API security headers
	apiConfig := middleware.APISecurityConfig()
	slog.Info("API security config",
		"x_content_type_options", apiConfig.XContentTypeOptions,
		"x_frame_options", apiConfig.XFrameOptions,
		"force_https", apiConfig.ForceHTTPS,
	)

	// Strict security headers
	strictConfig := middleware.StrictSecurityConfig()
	slog.Info("Strict security config",
		"csp", strictConfig.ContentSecurityPolicy,
		"hsts", strictConfig.StrictTransportSecurity,
		"referrer_policy", strictConfig.ReferrerPolicy,
	)

	// Custom CSP builder
	customCSP := middleware.NewCSPBuilder().
		DefaultSrc(middleware.CSPSelf).
		ScriptSrc(middleware.CSPSelf, "https://cdn.example.com").
		StyleSrc(middleware.CSPSelf, middleware.CSPUnsafeInline).
		ImgSrc(middleware.CSPSelf, "data:", "https:").
		ConnectSrc(middleware.CSPSelf, "https://api.example.com").
		FontSrc(middleware.CSPSelf, "https://fonts.gstatic.com").
		FrameAncestors(middleware.CSPNone).
		UpgradeInsecureRequests().
		Build()

	slog.Info("Custom CSP", "policy", customCSP)

	// Custom security config
	customSecurity := middleware.SecurityConfig{
		ContentSecurityPolicy:   customCSP,
		StrictTransportSecurity: "max-age=31536000; includeSubDomains; preload",
		ForceHTTPS:              true,
		XContentTypeOptions:     "nosniff",
		XFrameOptions:           "SAMEORIGIN",
		ReferrerPolicy:          "strict-origin",
		CustomHeaders: map[string]string{
			"X-Permitted-Cross-Domain-Policies": "none",
			"X-Download-Options":                "noopen",
		},
	}
	slog.Info("Custom security config",
		"force_https", customSecurity.ForceHTTPS,
		"custom_headers", customSecurity.CustomHeaders,
	)
}