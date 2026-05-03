package core

import "net/http"

// SecurityHeadersConfig controls which response headers SecurityHeadersMiddleware
// emits. The zero value is a sensible production default; create one with
// DefaultSecurityHeadersConfig() and tweak fields as needed.
//
// All fields except HSTSMaxAgeSeconds are raw header values that are sent
// verbatim. Setting a field to the empty string disables the header
// entirely.
type SecurityHeadersConfig struct {
	// XContentTypeOptions sets the X-Content-Type-Options header.
	// Recommended: "nosniff" — disables MIME sniffing.
	XContentTypeOptions string

	// XFrameOptions sets the X-Frame-Options header.
	// Recommended: "DENY" — blocks the response from being framed
	// (clickjacking protection). Use "SAMEORIGIN" if your own pages
	// must frame each other.
	XFrameOptions string

	// ReferrerPolicy sets the Referrer-Policy header.
	// Recommended: "strict-origin-when-cross-origin".
	ReferrerPolicy string

	// ContentSecurityPolicy sets the Content-Security-Policy header.
	// Empty by default because a CSP that breaks the application is
	// worse than no CSP — applications should opt-in with a value
	// tailored to their HTML/asset sources.
	ContentSecurityPolicy string

	// PermissionsPolicy sets the Permissions-Policy header (formerly
	// Feature-Policy). Empty by default; set per application.
	PermissionsPolicy string

	// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
	// Recommended: "same-origin" for sites that need cross-origin
	// isolation (e.g. SharedArrayBuffer).
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
	// Recommended: "same-origin" for API responses that should not be
	// embedded by other origins.
	CrossOriginResourcePolicy string

	// HSTSMaxAgeSeconds, when positive, emits a Strict-Transport-Security
	// header — but ONLY when the request was served over TLS
	// (r.TLS != nil). Sending HSTS over plain HTTP is a no-op per RFC
	// 6797 and risks confusing intermediaries; gating on TLS also
	// prevents a misconfigured local-dev deployment from accidentally
	// pinning HSTS for users.
	//
	// Common value: 31536000 (1 year).
	HSTSMaxAgeSeconds int

	// HSTSIncludeSubdomains adds the includeSubDomains directive.
	HSTSIncludeSubdomains bool

	// HSTSPreload adds the preload directive (only set this if you
	// understand the implications: https://hstspreload.org/).
	HSTSPreload bool
}

// DefaultSecurityHeadersConfig returns a security-headers configuration
// with safe production defaults. CSP and Permissions-Policy are left
// empty because they require per-application tuning.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		XContentTypeOptions:       "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		HSTSMaxAgeSeconds:         31536000, // 1 year
		HSTSIncludeSubdomains:     true,
	}
}

// SecurityHeadersMiddleware returns middleware that emits common HTTP
// security headers on every response.
//
// Pass nil to use DefaultSecurityHeadersConfig().
//
// The middleware writes headers BEFORE calling next.ServeHTTP, so the
// downstream handler may overwrite any header it manages explicitly
// (e.g. a route that needs a custom CSP).
//
// Example:
//
//	router.Use(core.SecurityHeadersMiddleware(nil))
//
//	cfg := core.DefaultSecurityHeadersConfig()
//	cfg.ContentSecurityPolicy = "default-src 'self'"
//	router.Use(core.SecurityHeadersMiddleware(&cfg))
func SecurityHeadersMiddleware(cfg *SecurityHeadersConfig) func(http.Handler) http.Handler {
	effective := DefaultSecurityHeadersConfig()
	if cfg != nil {
		effective = *cfg
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			if effective.XContentTypeOptions != "" {
				h.Set("X-Content-Type-Options", effective.XContentTypeOptions)
			}
			if effective.XFrameOptions != "" {
				h.Set("X-Frame-Options", effective.XFrameOptions)
			}
			if effective.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", effective.ReferrerPolicy)
			}
			if effective.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", effective.ContentSecurityPolicy)
			}
			if effective.PermissionsPolicy != "" {
				h.Set("Permissions-Policy", effective.PermissionsPolicy)
			}
			if effective.CrossOriginOpenerPolicy != "" {
				h.Set("Cross-Origin-Opener-Policy", effective.CrossOriginOpenerPolicy)
			}
			if effective.CrossOriginResourcePolicy != "" {
				h.Set("Cross-Origin-Resource-Policy", effective.CrossOriginResourcePolicy)
			}

			if effective.HSTSMaxAgeSeconds > 0 && r.TLS != nil {
				h.Set("Strict-Transport-Security", buildHSTS(effective))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildHSTS assembles the Strict-Transport-Security header value from
// the configured directives.
func buildHSTS(cfg SecurityHeadersConfig) string {
	out := "max-age=" + itoaPositive(cfg.HSTSMaxAgeSeconds)
	if cfg.HSTSIncludeSubdomains {
		out += "; includeSubDomains"
	}
	if cfg.HSTSPreload {
		out += "; preload"
	}
	return out
}

// itoaPositive is a tiny strconv-free integer formatter used only for
// the HSTS max-age value; keeps this file dependency-light.
func itoaPositive(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
