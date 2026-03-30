package core

import (
	"net/http"
	"time"
)

// CookieManager provides centralized cookie management for Aegis sessions.
//
// This manager encapsulates cookie security best practices:
//   - HTTPOnly: Prevents JavaScript access (XSS protection)
//   - Secure: Requires HTTPS in production (prevents MITM attacks)
//   - SameSite: Prevents CSRF attacks (Lax, Strict, or None)
//   - Configurable domain: Supports subdomain sharing
//   - Configurable path: Limits cookie scope
//
// The CookieManager is created by SessionService and uses settings from
// SessionConfig.CookieSettings. All session cookies are managed through
// this abstraction for consistency.
//
// Cookie Security Best Practices:
//   - Always enable HTTPOnly (prevents XSS from stealing cookies)
//   - Always enable Secure in production (requires HTTPS)
//   - Use SameSite=Lax for general APIs, Strict for sensitive operations
//   - Use SameSite=None only when needed for cross-site requests (requires Secure=true)
//
// Example:
//
//	cm := core.NewCookieManager(sessionConfig)
//	cm.SetSessionCookie(w, sessionToken) // Sets with configured security
//	token, err := cm.GetSessionCookie(r) // Reads session cookie
//	cm.ClearSessionCookie(w) // Deletes the session cookie
type CookieManager struct {
	config *SessionConfig
}

// CookieOptions defines options for setting a custom cookie.
// Used with SetCustomCookie for fine-grained control over cookie properties.
type CookieOptions struct {
	// Name is the cookie name
	Name string

	// Value is the cookie value (should not contain sensitive data unless encrypted)
	Value string

	// Path restricts the cookie to specific URL paths (default: "/")
	Path string

	// MaxAge is the cookie lifetime in seconds
	// Positive: Persistent cookie (survives browser restart)
	// Zero: Session cookie (deleted when browser closes)
	// Negative: Delete cookie immediately
	MaxAge int

	// Domain restricts the cookie to a specific domain or subdomain
	// Empty: Current domain only
	// ".example.com": All subdomains of example.com
	Domain string

	// Secure requires HTTPS for cookie transmission
	// Always true in production, can be false in local development
	Secure bool

	// HTTPOnly prevents JavaScript access to the cookie
	// Always true for session cookies to prevent XSS attacks
	HTTPOnly bool

	// SameSite controls cross-site cookie behavior (CSRF protection)
	// SameSiteLaxMode: Cookies sent with top-level navigation (default, good balance)
	// SameSiteStrictMode: Cookies never sent cross-site (maximum security)
	// SameSiteNoneMode: Cookies always sent (requires Secure=true, needed for OAuth flows)
	SameSite http.SameSite
}

// NewCookieManager creates a new CookieManager with the given configuration.
//
// If cfg is nil, uses DefaultSessionConfig() with secure defaults.
//
// The CookieManager will use the settings from cfg.CookieSettings for all
// cookie operations (HTTPOnly, Secure, SameSite, Domain, Path, Name).
func NewCookieManager(cfg *SessionConfig) *CookieManager {
	if cfg == nil {
		cfg = DefaultSessionConfig()
	}
	return &CookieManager{
		config: cfg,
	}
}

// parseSameSiteFromConfig converts the SameSite string from config to http.SameSite.
// Supports "Strict", "Lax", and "None". Defaults to Lax for invalid values.
func (cm *CookieManager) parseSameSiteFromConfig() http.SameSite {
	return parseSameSite(cm.config.CookieSettings.SameSite)
}

// GetConfig returns the underlying SessionConfig used by this CookieManager.
// Useful for inspecting current cookie settings.
func (cm *CookieManager) GetConfig() *SessionConfig {
	return cm.config
}

// SetCookie sets a cookie with the configured security defaults.
//
// This is a convenience method that applies CookieSettings from the config:
//   - Domain from config.CookieSettings.Domain
//   - Secure from config.CookieSettings.Secure
//   - HTTPOnly from config.CookieSettings.HTTPOnly
//   - SameSite from config.CookieSettings.SameSite
//   - Path is always "/" (DefaultCookiePath)
//
// Parameters:
//   - name: Cookie name
//   - value: Cookie value
//   - maxAge: Cookie lifetime (0 for session cookies, negative to delete)
//
// Example:
//
//	cm.SetCookie(w, "custom_data", "value", 24*time.Hour)
func (cm *CookieManager) SetCookie(w http.ResponseWriter, name, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure, HttpOnly, and SameSite are sourced from caller-supplied config
		Name:     name,
		Value:    value,
		Path:     DefaultCookiePath,
		Domain:   cm.config.CookieSettings.Domain,
		MaxAge:   int(maxAge.Seconds()),
		Secure:   cm.config.CookieSettings.Secure,
		HttpOnly: cm.config.CookieSettings.HTTPOnly,
		SameSite: cm.parseSameSiteFromConfig(),
	})
}

// GetSessionCookieName returns the configured session cookie name.
// Returns DefaultCookieName ("aegis_session") if not explicitly configured.
func (cm *CookieManager) GetSessionCookieName() string {
	if cm.config.CookieSettings.Name == "" {
		return DefaultCookieName
	}
	return cm.config.CookieSettings.Name
}

// SetSessionCookie sets the session cookie with configured security settings.
//
// This is the primary method for creating session cookies after successful login.
// The cookie expires according to config.SessionExpiry.
//
// Security settings applied:
//   - HTTPOnly: true (prevents XSS)
//   - Secure: from config (true in production)
//   - SameSite: from config (Lax/Strict for CSRF protection)
//   - Domain: from config (supports subdomain sharing)
//
// Parameters:
//   - token: The session token (random value from CreateSession)
//
// Example:
//
//	session, _ := sessionService.CreateSession(ctx, user)
//	cookieManager.SetSessionCookie(w, session.Token)
func (cm *CookieManager) SetSessionCookie(w http.ResponseWriter, token string) {
	cm.SetCookie(w, cm.GetSessionCookieName(), token, cm.config.SessionExpiry)
}

// ClearSessionCookie deletes the session cookie by setting MaxAge to -1.
//
// This is called during logout to invalidate the client-side session.
// The server-side session is deleted separately via SessionService.DeleteSession.
//
// Note: Even after clearing the cookie, the session token remains valid in the
// database until DeleteSession is called or the session expires naturally.
//
// Example:
//
//	sessionService.DeleteSession(ctx, sessionID) // Server-side cleanup
//	cookieManager.ClearSessionCookie(w) // Client-side cleanup
func (cm *CookieManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure, HttpOnly, and SameSite are sourced from caller-supplied config
		Name:     cm.GetSessionCookieName(),
		Value:    "",
		Path:     DefaultCookiePath,
		Domain:   cm.config.CookieSettings.Domain,
		MaxAge:   -1,
		Secure:   cm.config.CookieSettings.Secure,
		HttpOnly: cm.config.CookieSettings.HTTPOnly,
		SameSite: cm.parseSameSiteFromConfig(),
	})
}

// SetCustomCookie sets a custom cookie with fine-grained control over all options.
//
// This allows overriding specific cookie properties while still benefiting from
// config defaults for unspecified fields:
//   - SameSite: If zero, uses config.CookieSettings.SameSite
//   - Domain: If empty, uses config.CookieSettings.Domain
//
// Use this for plugin-specific cookies (CSRF tokens, OAuth state, etc.) that need
// different settings than the main session cookie.
//
// Example:
//
//	cm.SetCustomCookie(w, core.CookieOptions{
//		Name:     "csrf_token",
//		Value:    csrfToken,
//		Path:     "/",
//		MaxAge:   3600, // 1 hour
//		HTTPOnly: true,
//		Secure:   true,
//		SameSite: http.SameSiteStrictMode,
//	})
func (cm *CookieManager) SetCustomCookie(w http.ResponseWriter, opts CookieOptions) {
	// Apply defaults from config if not specified
	sameSite := opts.SameSite
	if sameSite == 0 {
		sameSite = cm.parseSameSiteFromConfig()
	}

	secure := opts.Secure
	httpOnly := opts.HTTPOnly
	domain := opts.Domain

	// Use config defaults if not explicitly set
	if domain == "" {
		domain = cm.config.CookieSettings.Domain
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure, HttpOnly, and SameSite are controlled by the caller via CookieOptions
		Name:     opts.Name,
		Value:    opts.Value,
		Path:     opts.Path,
		Domain:   domain,
		MaxAge:   opts.MaxAge,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: sameSite,
	})
}

// GetCookie retrieves a cookie value by name from the HTTP request.
//
// Returns an AuthError if:
//   - Cookie doesn't exist (AuthErrorCodeUnauthorized)
//   - Reading the cookie fails (AuthErrorCodeInternal)
//
// Example:
//
//	csrfToken, err := cm.GetCookie(r, "csrf_token")
//	if err != nil {
//		// Cookie missing or error
//	}
func (cm *CookieManager) GetCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", NewAuthError(AuthErrorCodeUnauthorized, "cookie not found")
		}
		return "", NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to get cookie", err)
	}
	return cookie.Value, nil
}

// GetSessionCookie retrieves the session token from the configured session cookie.
//
// This is the primary method used by AuthMiddleware to extract the session token
// for authentication. The cookie name is determined by GetSessionCookieName().
//
// Returns an AuthError if the cookie is missing or cannot be read.
//
// Example:
//
//	token, err := cm.GetSessionCookie(r)
//	if err != nil {
//		// User not authenticated via cookie
//	}
func (cm *CookieManager) GetSessionCookie(r *http.Request) (string, error) {
	return cm.GetCookie(r, cm.GetSessionCookieName())
}

// parseSameSite converts the SameSite string configuration to http.SameSite enum.
//
// Mappings:
//   - "Strict" → http.SameSiteStrictMode (never sent cross-site)
//   - "None" → http.SameSiteNoneMode (always sent, requires Secure=true)
//   - Anything else → http.SameSiteLaxMode (default, sent with top-level navigation)
//
// SameSite provides CSRF protection by controlling when cookies are sent in
// cross-site requests.
func parseSameSite(value string) http.SameSite {
	switch value {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
