package core

import (
	"net/http"
)

// AuthMiddleware returns middleware that validates sessions and injects user into context
func AuthMiddleware(sessionService *SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try to get session from cookie first
			cookie, err := r.Cookie("aegis_session")
			if err == nil && cookie.Value != "" {
				session, user, err := sessionService.ValidateSession(ctx, cookie.Value)
				if err == nil && session != nil && user != nil {
					ctx = WithUser(ctx, user)
					r = r.WithContext(ctx)
				}
			} else if sessionService.IsBearerAuthEnabled() {
				// Only check Authorization header if bearer auth is explicitly enabled
				// This requires the bearer plugin to be registered
				token := r.Header.Get("Authorization")
				if token != "" {
					// Remove "Bearer " prefix if present
					if len(token) > 7 && token[:7] == "Bearer " {
						token = token[7:]
					}

					session, user, err := sessionService.ValidateSession(ctx, token)
					if err == nil && session != nil && user != nil {
						ctx = WithUser(ctx, user)
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthMiddleware returns middleware that requires authentication.
func RequireAuthMiddleware(_ *SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Authenticated(r.Context()) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts the client IP address from the request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// Check X-Real-IP header
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// SetSessionCookie sets a session cookie
func SetSessionCookie(w http.ResponseWriter, token string, cfg *SessionConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     "aegis_session",
		Value:    token,
		Path:     "/",
		Domain:   cfg.CookieSettings.Domain,
		MaxAge:   int(cfg.SessionExpiry.Seconds()),
		Secure:   cfg.CookieSettings.Secure,
		HttpOnly: cfg.CookieSettings.HTTPOnly,
		SameSite: parseSameSite(cfg.CookieSettings.SameSite),
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(w http.ResponseWriter, cfg *SessionConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     "aegis_session",
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieSettings.Domain,
		MaxAge:   -1,
		Secure:   cfg.CookieSettings.Secure,
		HttpOnly: cfg.CookieSettings.HTTPOnly,
		SameSite: parseSameSite(cfg.CookieSettings.SameSite),
	})
}

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
