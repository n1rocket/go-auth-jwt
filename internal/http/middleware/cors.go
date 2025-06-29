package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:          86400, // 24 hours
	}
}

// StrictCORSConfig returns a strict CORS configuration for production
func StrictCORSConfig(allowedOrigins []string) CORSConfig {
	return CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:          3600, // 1 hour
	}
}

// NewCORS creates a new CORS middleware with the given configuration
func NewCORS(config CORSConfig) func(http.Handler) http.Handler {
	// Pre-compute header values
	allowedMethods := strings.Join(config.AllowedMethods, ", ")
	allowedHeaders := strings.Join(config.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(config.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If no origin header, this is not a CORS request
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			if isAllowedOrigin(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if exposedHeaders != "" {
					w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				// Check if this is a preflight request
				if r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
					
					// Handle requested headers
					requestedHeaders := r.Header.Get("Access-Control-Request-Headers")
					if requestedHeaders != "" {
						// Check if all requested headers are allowed
						if areHeadersAllowed(requestedHeaders, config.AllowedHeaders) {
							w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
						} else {
							w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
						}
					}
					
					w.Header().Set("Access-Control-Max-Age", maxAge)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAllowedOrigin checks if an origin is in the allowed list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

// areHeadersAllowed checks if all requested headers are allowed
func areHeadersAllowed(requested string, allowed []string) bool {
	requestedHeaders := strings.Split(requested, ",")
	
	// Create a map for faster lookup
	allowedMap := make(map[string]bool)
	for _, h := range allowed {
		allowedMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	// Check each requested header
	for _, h := range requestedHeaders {
		header := strings.ToLower(strings.TrimSpace(h))
		if !allowedMap[header] && !isSimpleHeader(header) {
			return false
		}
	}

	return true
}

// isSimpleHeader checks if a header is a simple header per CORS spec
func isSimpleHeader(header string) bool {
	simpleHeaders := map[string]bool{
		"accept":          true,
		"accept-language": true,
		"content-language": true,
		"content-type":    true, // with restrictions, but we'll allow it
	}
	return simpleHeaders[header]
}