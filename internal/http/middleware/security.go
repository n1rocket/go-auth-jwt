package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// SecurityConfig holds security headers configuration
type SecurityConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string
	
	// Cross-Origin policies
	CrossOriginEmbedderPolicy   string
	CrossOriginOpenerPolicy     string
	CrossOriginResourcePolicy   string
	
	// HSTS
	StrictTransportSecurity string
	ForceHTTPS              bool
	
	// Other security headers
	XContentTypeOptions    string
	XFrameOptions          string
	XSSProtection          string
	ReferrerPolicy         string
	PermissionsPolicy      string
	
	// Custom headers
	CustomHeaders map[string]string
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",
		
		CrossOriginEmbedderPolicy:  "require-corp",
		CrossOriginOpenerPolicy:    "same-origin",
		CrossOriginResourcePolicy:  "same-origin",
		
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ForceHTTPS:              false, // Set to true in production
		
		XContentTypeOptions: "nosniff",
		XFrameOptions:       "DENY",
		XSSProtection:       "1; mode=block",
		ReferrerPolicy:      "strict-origin-when-cross-origin",
		PermissionsPolicy:   "geolocation=(), microphone=(), camera=()",
		
		CustomHeaders: make(map[string]string),
	}
}

// StrictSecurityConfig returns strict security configuration for production
func StrictSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentSecurityPolicy: "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; upgrade-insecure-requests;",
		
		CrossOriginEmbedderPolicy:  "require-corp",
		CrossOriginOpenerPolicy:    "same-origin",
		CrossOriginResourcePolicy:  "same-origin",
		
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		ForceHTTPS:              true,
		
		XContentTypeOptions: "nosniff",
		XFrameOptions:       "DENY",
		XSSProtection:       "0", // Disabled in modern browsers, can cause issues
		ReferrerPolicy:      "no-referrer",
		PermissionsPolicy:   "accelerometer=(), ambient-light-sensor=(), autoplay=(), battery=(), camera=(), cross-origin-isolated=(), display-capture=(), document-domain=(), encrypted-media=(), execution-while-not-rendered=(), execution-while-out-of-viewport=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), navigation-override=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), web-share=(), xr-spatial-tracking=()",
		
		CustomHeaders: map[string]string{
			"X-Permitted-Cross-Domain-Policies": "none",
		},
	}
}

// APISecurityConfig returns security configuration optimized for APIs
func APISecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentSecurityPolicy: "", // Not needed for APIs
		
		CrossOriginEmbedderPolicy:  "require-corp",
		CrossOriginOpenerPolicy:    "same-origin",
		CrossOriginResourcePolicy:  "cross-origin", // Allow API access from different origins
		
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ForceHTTPS:              true,
		
		XContentTypeOptions: "nosniff",
		XFrameOptions:       "DENY",
		XSSProtection:       "",
		ReferrerPolicy:      "strict-origin-when-cross-origin",
		PermissionsPolicy:   "",
		
		CustomHeaders: make(map[string]string),
	}
}

// SecurityHeaders returns a middleware that sets security headers
func SecurityHeaders(config SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Force HTTPS redirect if configured
			if config.ForceHTTPS && r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
				httpsURL := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}

			// Set security headers
			setHeader(w, "Content-Security-Policy", config.ContentSecurityPolicy)
			setHeader(w, "Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			setHeader(w, "Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			setHeader(w, "Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			
			// HSTS only on HTTPS connections
			if (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") && config.StrictTransportSecurity != "" {
				w.Header().Set("Strict-Transport-Security", config.StrictTransportSecurity)
			}
			
			setHeader(w, "X-Content-Type-Options", config.XContentTypeOptions)
			setHeader(w, "X-Frame-Options", config.XFrameOptions)
			setHeader(w, "X-XSS-Protection", config.XSSProtection)
			setHeader(w, "Referrer-Policy", config.ReferrerPolicy)
			setHeader(w, "Permissions-Policy", config.PermissionsPolicy)
			
			// Set custom headers
			for name, value := range config.CustomHeaders {
				w.Header().Set(name, value)
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// setHeader sets a header only if the value is not empty
func setHeader(w http.ResponseWriter, name, value string) {
	if value != "" {
		w.Header().Set(name, value)
	}
}

// CSPBuilder helps build Content Security Policy strings
type CSPBuilder struct {
	directives map[string][]string
}

// NewCSPBuilder creates a new CSP builder
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[string][]string),
	}
}

// DefaultSrc sets the default-src directive
func (b *CSPBuilder) DefaultSrc(sources ...string) *CSPBuilder {
	b.directives["default-src"] = sources
	return b
}

// ScriptSrc sets the script-src directive
func (b *CSPBuilder) ScriptSrc(sources ...string) *CSPBuilder {
	b.directives["script-src"] = sources
	return b
}

// StyleSrc sets the style-src directive
func (b *CSPBuilder) StyleSrc(sources ...string) *CSPBuilder {
	b.directives["style-src"] = sources
	return b
}

// ImgSrc sets the img-src directive
func (b *CSPBuilder) ImgSrc(sources ...string) *CSPBuilder {
	b.directives["img-src"] = sources
	return b
}

// ConnectSrc sets the connect-src directive
func (b *CSPBuilder) ConnectSrc(sources ...string) *CSPBuilder {
	b.directives["connect-src"] = sources
	return b
}

// FontSrc sets the font-src directive
func (b *CSPBuilder) FontSrc(sources ...string) *CSPBuilder {
	b.directives["font-src"] = sources
	return b
}

// FrameAncestors sets the frame-ancestors directive
func (b *CSPBuilder) FrameAncestors(sources ...string) *CSPBuilder {
	b.directives["frame-ancestors"] = sources
	return b
}

// BaseURI sets the base-uri directive
func (b *CSPBuilder) BaseURI(sources ...string) *CSPBuilder {
	b.directives["base-uri"] = sources
	return b
}

// FormAction sets the form-action directive
func (b *CSPBuilder) FormAction(sources ...string) *CSPBuilder {
	b.directives["form-action"] = sources
	return b
}

// UpgradeInsecureRequests adds the upgrade-insecure-requests directive
func (b *CSPBuilder) UpgradeInsecureRequests() *CSPBuilder {
	b.directives["upgrade-insecure-requests"] = []string{}
	return b
}

// Build creates the CSP string
func (b *CSPBuilder) Build() string {
	var parts []string
	
	// Ensure consistent order
	order := []string{
		"default-src", "script-src", "style-src", "img-src", "font-src",
		"connect-src", "media-src", "object-src", "frame-src", "worker-src",
		"form-action", "frame-ancestors", "base-uri", "upgrade-insecure-requests",
	}
	
	for _, directive := range order {
		if sources, ok := b.directives[directive]; ok {
			if len(sources) == 0 {
				// Directives like upgrade-insecure-requests don't have values
				parts = append(parts, directive)
			} else {
				parts = append(parts, fmt.Sprintf("%s %s", directive, strings.Join(sources, " ")))
			}
		}
	}
	
	// Add any directives not in the order list
	for directive, sources := range b.directives {
		found := false
		for _, ordered := range order {
			if ordered == directive {
				found = true
				break
			}
		}
		if !found {
			if len(sources) == 0 {
				parts = append(parts, directive)
			} else {
				parts = append(parts, fmt.Sprintf("%s %s", directive, strings.Join(sources, " ")))
			}
		}
	}
	
	return strings.Join(parts, "; ")
}

// Common CSP sources
const (
	CSPSelf          = "'self'"
	CSPNone          = "'none'"
	CSPUnsafeInline  = "'unsafe-inline'"
	CSPUnsafeEval    = "'unsafe-eval'"
	CSPStrictDynamic = "'strict-dynamic'"
	CSPReportSample  = "'report-sample'"
)