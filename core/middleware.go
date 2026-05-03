package core

import (
	"context"
	"net"
	"net/http"

	"github.com/theinventorylib/aegis/auth"
)

// Default request body size limits prevent denial-of-service attacks via
// extremely large request bodies. These limits can be overridden per-route.
const (
	// DefaultMaxBodySize is the default maximum request body size (1MB)
	// Suitable for most API endpoints with JSON payloads
	DefaultMaxBodySize int64 = 1 << 20 // 1 MB

	// MaxBodySizeSmall is for endpoints with small payloads like login (64KB)
	// Use for authentication endpoints to prevent abuse
	MaxBodySizeSmall int64 = 64 << 10 // 64 KB

	// MaxBodySizeLarge is for endpoints that may have larger payloads (10MB)
	// Use for file uploads or bulk operations
	MaxBodySizeLarge int64 = 10 << 20 // 10 MB
)

// AegisContextMiddleware initializes the Aegis request context for each HTTP request.
//
// This middleware is called automatically by the Aegis framework and sets up:
//   - Request ID: Unique identifier for tracing and logging
//   - Request metadata: IP address, user agent, method, path
//   - Plugin data store: Key-value storage for plugins to share data
//   - Initialization marker: Prevents double initialization
//
// The middleware is idempotent - if the context is already initialized (e.g., by
// a parent middleware), it skips initialization and passes through.
//
// Users typically don't need to call this directly - it's integrated into the
// Aegis HTTP stack automatically.
func AegisContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Skip if already initialized (avoid double initialization)
			if IsContextInitialized(ctx) {
				next.ServeHTTP(w, r)
				return
			}

			// Generate request ID for tracing
			requestID := GenerateID()
			ctx = WithRequestID(ctx, requestID)

			// Set request metadata
			meta := &RequestMeta{
				RequestID: requestID,
				IPAddress: GetClientIP(r),
				UserAgent: r.UserAgent(),
				Method:    r.Method,
				Path:      r.URL.Path,
			}
			ctx = WithRequestMeta(ctx, meta)

			// Initialize plugin data store
			ctx = WithPluginData(ctx, NewPluginData())

			// Mark context as initialized
			ctx = WithContextInitialized(ctx)

			// Continue with enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthMiddleware returns HTTP middleware that validates sessions and injects
// authenticated user data into the request context.
//
// Authentication flow:
//  1. Check if context already initialized (if not, initialize it)
//  2. Look for session in per-request cache (avoid redundant lookups)
//  3. Extract session token from cookie or Authorization header
//  4. Validate the session token (database + Redis cache)
//  5. Load the associated user
//  6. Create EnrichedUser and populate with plugin extensions
//  7. Inject user and session into context
//
// After this middleware, authenticated requests can access:
//   - core.GetUser(ctx) - Get the base user
//   - core.GetEnrichedUser(ctx) - Get user with plugin extensions
//   - core.GetSession(ctx) - Get the current session
//
// Unauthenticated requests continue normally (no user/session in context).
// Protected routes should check for authentication and return 401 if missing.
//
// The middleware caches the session in the request context to avoid redundant
// database/Redis lookups if multiple handlers need authentication data.
//
// Parameters:
//   - sessionService: Service for session validation and user lookup
//
// Example usage:
//
//	protectedRouter.Use(core.AuthMiddleware(sessionService))
func AuthMiddleware(sessionService *SessionService) func(http.Handler) http.Handler {
	// Get cookie manager for consistent cookie handling
	cookieManager := sessionService.GetCookieManager()

	// Pre-create the context middleware for inline use
	contextMiddleware := AegisContextMiddleware()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Ensure context is initialized (idempotent - skips if already done)
			if !IsContextInitialized(ctx) {
				// Wrap the rest of the handler in context middleware
				contextMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handleAuth(sessionService, cookieManager, next, w, r)
				})).ServeHTTP(w, r)
				return
			}

			handleAuth(sessionService, cookieManager, next, w, r)
		})
	}
}

// handleAuth performs the actual authentication logic for AuthMiddleware.
// Separated into a helper function to avoid code duplication between the
// pre-initialized and post-initialized context paths.
//
// This function:
//   - Checks for cached session in context (per-request cache)
//   - Extracts token from cookie or Authorization header
//   - Validates the token and loads user data
//   - Creates EnrichedUser for plugin extension
//   - Injects authenticated data into context
func handleAuth(sessionService *SessionService, cookieManager *CookieManager, next http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if user is already authenticated in context (per-request cache).
	// Using Authenticated instead of HasSession so that stateless token schemes
	// (e.g. JWT) — which store a nil session — also benefit from the early exit.
	if Authenticated(ctx) {
		next.ServeHTTP(w, r)
		return
	}

	// Try to get session from cookie first (uses configured cookie name)
	token, err := cookieManager.GetSessionCookie(r)
	if err == nil && token != "" {
		session, user, err := sessionService.ValidateSession(ctx, token)
		if err == nil && session != nil && user != nil {
			ctx = setAuthContext(ctx, user, session)
			r = r.WithContext(ctx)
		}
	} else if sessionService.IsBearerAuthEnabled() {
		// Only check Authorization header if bearer auth is explicitly enabled
		// This is enabled via config.WithBearerAuth(true) or config.WithAPIOnlyMode(true)
		token := r.Header.Get("Authorization")
		if token != "" {
			// Remove "Bearer " prefix if present
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}

			var (
				user    *auth.User
				session *auth.Session
				err     error
			)

			// Try each registered bearer token validator in order (e.g., JWT plugin,
			// API-key plugin, …). The first validator that returns a non-nil user
			// wins; remaining validators are skipped.
			for _, v := range sessionService.GetBearerTokenValidators() {
				user, session, err = v.ValidateBearerToken(ctx, token)
				if err == nil && user != nil {
					break
				}
			}

			// Fall back to opaque session-token database lookup when no validator
			// recognized the token.
			if err != nil || user == nil {
				session, user, err = sessionService.ValidateSession(ctx, token)
			}

			if err == nil && user != nil {
				ctx = setAuthContext(ctx, user, session)
				r = r.WithContext(ctx)
			}
		}
	}

	next.ServeHTTP(w, r)
}

// setAuthContext sets user, session, and enriched user in context.
func setAuthContext(ctx context.Context, user *auth.User, session *auth.Session) context.Context {
	ctx = WithUser(ctx, user)
	ctx = WithSession(ctx, session)
	ctx = WithEnrichedUser(ctx, NewEnrichedUser(user))
	return ctx
}

// RequireAuthMiddleware returns middleware that requires authentication.
func RequireAuthMiddleware(_ *SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Authenticated(r.Context()) {
				WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts the client IP address from the request.
//
// SECURITY: This function does NOT trust X-Forwarded-For or X-Real-IP
// headers. It returns r.RemoteAddr unchanged. Honoring proxy headers
// without an allowlist lets any client spoof their IP, which would
// break rate-limit / audit / lockout protection.
//
// If you run Aegis behind a reverse proxy, use GetClientIPTrusted with
// a parsed CIDR allowlist of your trusted proxy addresses, or rely on
// the rate-limiter's built-in TrustedProxies handling.
func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// GetClientIPTrusted returns the client IP, honoring X-Forwarded-For /
// X-Real-IP only when the immediate peer (r.RemoteAddr) is contained in
// one of the trustedProxies CIDR ranges. trustedProxies entries may be
// in CIDR form ("10.0.0.0/8") or bare IP form ("10.0.0.5"); invalid
// entries are silently ignored.
func GetClientIPTrusted(r *http.Request, trustedProxies []string) string {
	return clientIPFromRequest(r, parseTrustedProxies(trustedProxies, nil))
}

// MaxBodySizeMiddleware returns middleware that limits request body size.
// This helps prevent DoS attacks via large request bodies.
//
// Example:
//
//	router.Use(core.MaxBodySizeMiddleware(core.DefaultMaxBodySize))
func MaxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
