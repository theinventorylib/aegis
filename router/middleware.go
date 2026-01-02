package router

import "net/http"

// SecurityHeadersMiddleware adds security headers to all HTTP responses.
//
// This middleware implements OWASP security best practices by adding headers that
// protect against common web vulnerabilities:
//
// Headers added:
//
//   - X-Content-Type-Options: nosniff
//     Prevents MIME type sniffing (prevents browsers from interpreting files as different MIME type)
//     Protects against: Drive-by download attacks
//
//   - X-Frame-Options: DENY
//     Prevents the page from being embedded in <iframe>, <frame>, or <object>
//     Protects against: Clickjacking attacks
//
//   - X-XSS-Protection: 1; mode=block
//     Enables browser XSS filter (legacy, mostly replaced by CSP)
//     Protects against: Reflected XSS attacks in older browsers
//
//   - Referrer-Policy: strict-origin-when-cross-origin
//     Controls how much referrer information is sent with requests
//     Protects against: Information leakage via Referer header
//
//   - Content-Security-Policy: default-src 'self'
//     Basic CSP that only allows resources from the same origin
//     Protects against: XSS, data injection, malicious scripts
//     Note: This is a basic policy - override with custom CSP for your needs
//
// Usage:
//
//	r.Use(router.SecurityHeadersMiddleware)
//
// Note: The CSP header can be customized by setting it before this middleware
// runs, or by using a custom middleware that overrides it.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Basic CSP - users can override for their specific needs
		if w.Header().Get("Content-Security-Policy") == "" {
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
		}
		next.ServeHTTP(w, r)
	})
}

// MaxRequestBodySize is the default maximum request body size (1MB).
//
// This limit prevents denial-of-service attacks via extremely large request bodies
// that could exhaust server memory or network bandwidth.
//
// Adjust this based on your use case:
//   - API endpoints: 1MB is usually sufficient
//   - File uploads: Increase to 10MB or more
//   - Webhooks: May need larger limits depending on payload size
const MaxRequestBodySize = 1 << 20 // 1MB

// MaxRequestBodyMiddleware limits the size of HTTP request bodies.
//
// This middleware wraps the request body with http.MaxBytesReader, which enforces
// a size limit. If a request exceeds the limit, reading the body will fail and
// the connection will be closed.
//
// Benefits:
//   - Prevents memory exhaustion from large requests
//   - Prevents network bandwidth abuse
//   - Protects against ZIP bomb attacks
//   - Limits upload file sizes
//
// Parameters:
//   - maxBytes: Maximum allowed request body size in bytes
//
// Usage:
//
//	// Apply 1MB limit to all routes
//	r.Use(router.MaxRequestBodyMiddleware(1 << 20))
//
//	// Apply 10MB limit for file upload routes
//	uploadRouter.Use(router.MaxRequestBodyMiddleware(10 << 20))
func MaxRequestBodyMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DefaultSecurityMiddleware applies a standard set of security protections.
//
// This is a convenience middleware that combines:
//   - SecurityHeadersMiddleware (OWASP security headers)
//   - MaxRequestBodyMiddleware with 1MB limit (DoS protection)
//
// This provides a good security baseline for most applications. Apply this
// middleware early in your middleware chain (before authentication) to protect
// all routes.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(router.DefaultSecurityMiddleware)
//	r.Use(core.AegisContextMiddleware())
//	// ... rest of your routes
func DefaultSecurityMiddleware(next http.Handler) http.Handler {
	return SecurityHeadersMiddleware(
		MaxRequestBodyMiddleware(MaxRequestBodySize)(next),
	)
}
