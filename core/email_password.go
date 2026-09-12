package core

import (
	"context"
	"strings"
	"sync"

	"github.com/theinventorylib/aegis/v2/auth"
)

// EmailPasswordHandlers provides HTTP handlers and programmatic functions for
// traditional email+password authentication (Login / Register).
//
// IP address and user agent are automatically extracted from the request context
// (populated by AegisContextMiddleware). For non-HTTP usage, populate the context
// with WithRequestMeta.
type EmailPasswordHandlers struct {
	authService *AuthService
}

// NewEmailPasswordHandlers creates a new set of email+password authentication handlers.
func NewEmailPasswordHandlers(authService *AuthService) *EmailPasswordHandlers {
	return &EmailPasswordHandlers{
		authService: authService,
	}
}

// recordFailure counts a failed attempt for lockout tracking and audits it.
// identifier is the normalized login key (lowercased email or username).
// Single owner of that repeated block in Login.
func (h *EmailPasswordHandlers) recordFailure(ctx context.Context, identifier, userID, reason string) {
	if h.authService.loginAttemptTracker != nil {
		_, _, _ = h.authService.loginAttemptTracker.RecordFailedAttempt(ctx, identifier)
	}
	h.auditFailure(ctx, identifier, userID, reason)
}

// auditFailure records a failed-login audit event without touching the lockout
// counter (used for non-credential failures such as an unverified email).
func (h *EmailPasswordHandlers) auditFailure(ctx context.Context, identifier, userID, reason string) {
	logAuthEvent(ctx, nil, h.authService.auditLogger, AuditEventLoginFailed, userID, false, map[string]any{
		"identifier": redactForLog(identifier),
		"reason":     reason,
	})
}

// LoginResult contains the result of an email+password login.
type LoginResult struct {
	// User is the authenticated user
	User auth.User
	// Session is the newly created session
	Session *auth.Session
	// Token is the session token
	Token string
}

// dummyVerifyHash lazily computes a throwaway Argon2id hash (default
// parameters) used to equalize response timing for unknown users.
var dummyVerifyHash = sync.OnceValues(func() (string, error) {
	return HashPassword("aegis-timing-equalizer", 0, 0, 0, 0)
})

// Login authenticates a user with a password. identifier may be either the
// user's email address or their username (when one was set at registration).
// IP address and user agent are automatically extracted from the request context.
func (h *EmailPasswordHandlers) Login(ctx context.Context, identifier, password string) (*LoginResult, error) {
	identifier = strings.TrimSpace(identifier)
	// Lockout key is normalized so casing/spacing cannot bypass the counter.
	loginKey := strings.ToLower(identifier)

	// Check if account is locked out
	if h.authService.loginAttemptTracker != nil {
		locked, remaining, err := h.authService.loginAttemptTracker.IsLockedOut(ctx, loginKey)
		if err != nil {
			return nil, err
		}
		if locked {
			logAuthEvent(ctx, nil, h.authService.auditLogger, AuditEventLoginFailed, "", false, map[string]any{
				"identifier": redactForLog(loginKey),
				"reason":     "account_locked",
				"remaining":  remaining.String(),
			})
			return nil, NewAuthError(AuthErrorCodeRateLimit, "Account is temporarily locked")
		}
	}

	// Resolve the identifier to a user (email first, then username).
	user, err := h.resolveUser(ctx, identifier)
	if err != nil {
		// Burn an Argon2id round on a constant hash so the unknown-user
		// path takes the same time as the wrong-password path; otherwise
		// the response-time difference reveals whether the identifier exists.
		// The dummy hash is generated lazily so API-only deployments that
		// never login don't pay the hashing cost at startup.
		if dummyHash, hashErr := dummyVerifyHash(); hashErr == nil {
			_, _ = VerifyPassword(password, dummyHash)
		}
		h.recordFailure(ctx, loginKey, "", "user_not_found")
		return nil, ErrInvalidCredentials
	}

	uid := user.GetID()

	// Verify password
	valid, err := h.authService.Account.VerifyPassword(ctx, uid, password)
	if err != nil || !valid {
		h.recordFailure(ctx, loginKey, uid, "invalid_password")
		return nil, ErrInvalidCredentials
	}

	// Email verification gate. Fail closed when verification is required but no
	// checker is wired, so the setting is never silently ignored. These are
	// audited but deliberately not counted toward the lockout counter: the
	// password was correct and a legitimate unverified user must not be locked
	// out for trying to log in.
	if h.authService.authConfig.RequireEmailVerification {
		if h.authService.emailVerificationCheck == nil {
			h.auditFailure(ctx, loginKey, uid, "email_not_verified")
			return nil, ErrEmailNotVerified
		}
		verified, verr := h.authService.emailVerificationCheck(ctx, user)
		if verr != nil || !verified {
			h.auditFailure(ctx, loginKey, uid, "email_not_verified")
			return nil, ErrEmailNotVerified
		}
	}

	// Clear failed attempts on successful login
	if h.authService.loginAttemptTracker != nil {
		_ = h.authService.loginAttemptTracker.ClearAttempts(ctx, loginKey)
	}

	// Create session
	session, err := h.authService.Session.CreateSession(ctx, &user)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:    user,
		Session: session,
		Token:   session.Token,
	}, nil
}

// resolveUser resolves an email-or-username identifier to a user. Email is
// tried first; a username is matched against the credentials account's
// provider account ID.
func (h *EmailPasswordHandlers) resolveUser(ctx context.Context, identifier string) (auth.User, error) {
	if email := SanitizeEmail(identifier); email != "" {
		if user, err := h.authService.User.GetUserByEmail(ctx, email); err == nil {
			return user, nil
		}
	}
	if username := SanitizeUsername(identifier, 0); username != "" {
		account, err := h.authService.Account.GetAccountByProvider(ctx, PasswordProvider, username)
		if err == nil {
			return h.authService.User.GetUserByID(ctx, account.UserID)
		}
	}
	return auth.User{}, ErrUserNotFound
}

// RegisterResult contains the result of an email+password registration.
type RegisterResult struct {
	// User is the newly created user
	User auth.User
	// Session is the newly created session (auto-login). Nil when
	// VerificationRequired is true.
	Session *auth.Session
	// Token is the session token. Empty when VerificationRequired is true.
	Token string
	// VerificationRequired reports that the account was created but no
	// session was issued because email verification is required first.
	VerificationRequired bool
}

// Register registers a new user with email and password programmatically.
// IP address and user agent are automatically extracted from the request context.
func (h *EmailPasswordHandlers) Register(ctx context.Context, name, email, password string) (*RegisterResult, error) {
	return h.RegisterWithUsername(ctx, name, email, "", password)
}

// RegisterWithUsername registers a user with an optional unique username that
// can also be used to log in. When AuthConfig.RequireEmailVerification is set,
// no session is issued; the caller must complete email verification first.
func (h *EmailPasswordHandlers) RegisterWithUsername(ctx context.Context, name, email, username, password string) (*RegisterResult, error) {
	name = SanitizeString(name, nil)
	email = SanitizeEmail(email)

	if name == "" {
		return nil, ValidationError{Field: "name", Message: "is required"}
	}
	if email == "" {
		return nil, ValidationError{Field: "email", Message: "is required"}
	}
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	user, err := h.authService.User.CreateUserWithUsername(ctx, name, email, username, password)
	if err != nil {
		return nil, err
	}

	result := &RegisterResult{
		User:                 user,
		VerificationRequired: h.authService.authConfig.RequireEmailVerification,
	}
	if result.VerificationRequired {
		return result, nil
	}

	// Create session (auto-login)
	session, err := h.authService.Session.CreateSession(ctx, &user)
	if err != nil {
		return nil, err
	}
	result.Session = session
	result.Token = session.Token
	return result, nil
}

// ========== REQUEST STRUCTS ==========

// LoginRequest represents the JSON payload for email+password login. Provide
// either Email or Username (Email is tried first when both are set).
type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents the JSON payload for email+password registration.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
}

// ========== END OF PROGRAMMATIC FUNCTIONS ==========
