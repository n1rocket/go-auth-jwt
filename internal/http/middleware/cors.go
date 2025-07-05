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
			http.MethodHead,
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
		MaxAge:           86400, // 24 hours
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
		MaxAge:           3600, // 1 hour
	}
}

// NewCORS creates a new CORS middleware with the given configuration
func NewCORS(config CORSConfig) func(http.Handler) http.Handler {
	// Pre-compute header values
	allowedMethods := strings.Join(config.AllowedMethods, ",")
	exposedHeaders := strings.Join(config.ExposedHeaders, ",")
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
				// If wildcard is used, return "*" unless credentials are required
				if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" && !config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}

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
				requestedMethod := r.Header.Get("Access-Control-Request-Method")
				if requestedMethod != "" {
					// Check if the requested method is allowed
					if !isMethodAllowed(requestedMethod, config.AllowedMethods) {
						w.WriteHeader(http.StatusForbidden)
						return
					}

					// Handle requested headers
					requestedHeaders := r.Header.Get("Access-Control-Request-Headers")
					if requestedHeaders != "" {
						// Check if all requested headers are allowed
						if !areHeadersAllowed(requestedHeaders, config.AllowedHeaders) {
							w.WriteHeader(http.StatusForbidden)
							return
						}
						w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
					}

					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
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
		if strings.Contains(allowed, "*.") {
			// Extract the parts: scheme and domain
			wildcardParts := strings.SplitN(allowed, "://", 2)
			if len(wildcardParts) == 2 {
				scheme := wildcardParts[0]
				wildcardDomain := wildcardParts[1]

				// Extract origin parts
				originParts := strings.SplitN(origin, "://", 2)
				if len(originParts) == 2 && originParts[0] == scheme {
					originDomain := originParts[1]

					// Remove the "*" and check if origin matches the pattern
					if strings.HasPrefix(wildcardDomain, "*.") {
						baseDomain := wildcardDomain[1:] // Keeps the dot

						// Check if origin ends with the base domain and has a subdomain
						if strings.HasSuffix(originDomain, baseDomain) {
							// Make sure there's a subdomain (not just the base domain)
							subdomainPart := strings.TrimSuffix(originDomain, baseDomain)
							if len(subdomainPart) > 0 && !strings.Contains(subdomainPart, ".") {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// isMethodAllowed checks if a method is in the allowed list
func isMethodAllowed(method string, allowed []string) bool {
	for _, m := range allowed {
		if m == method {
			return true
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
		"accept":           true,
		"accept-language":  true,
		"content-language": true,
		"content-type":     true, // with restrictions, but we'll allow it
	}
	return simpleHeaders[header]
}
