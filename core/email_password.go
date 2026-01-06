package core

import (
	"context"

	"github.com/theinventorylib/aegis/auth"
)

// EmailPasswordHandlers provides HTTP handlers and programmatic functions for traditional email+password authentication.
//
// This handler set implements the classic username/password authentication flow:
//   - Registration: Create a new user with email+password
//   - Login: Authenticate existing user with credentials
//
// HTTP handlers are private (lowercase) and automatically mounted. For programmatic
// use without HTTP, use the public methods:
//   - Login(ctx, email, password, ipAddress, userAgent)
//   - Register(ctx, name, email, password, ipAddress, userAgent)
type EmailPasswordHandlers struct {
	authService *AuthService
}

// NewEmailPasswordHandlers creates a new set of email+password authentication handlers.
func NewEmailPasswordHandlers(authService *AuthService) *EmailPasswordHandlers {
	return &EmailPasswordHandlers{
		authService: authService,
	}
}

// ========== PUBLIC PROGRAMMATIC FUNCTIONS ==========

// LoginResult contains the result of an email+password login.
type LoginResult struct {
	// User is the authenticated user
	User auth.User
	// Session is the newly created session
	Session *auth.Session
	// Token is the session token
	Token string
}

// Login authenticates a user with email and password programmatically.
func (h *EmailPasswordHandlers) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error) {
	// Check if account is locked out
	if h.authService.loginAttemptTracker != nil {
		locked, remaining, err := h.authService.loginAttemptTracker.IsLockedOut(ctx, email)
		if err != nil {
			return nil, err
		}
		if locked {
			_ = h.authService.auditLogger.LogAuthEvent(ctx, AuditEventLoginFailed, "", ipAddress, userAgent, false, map[string]interface{}{
				"email":     email,
				"reason":    "account_locked",
				"remaining": remaining.String(),
			})
			return nil, NewAuthError(AuthErrorCodeRateLimit, "Account is temporarily locked")
		}
	}

	// Get user by email
	user, err := h.authService.User.GetUserByEmail(ctx, email)
	if err != nil {
		if h.authService.loginAttemptTracker != nil {
			attempts, lockout, err := h.authService.loginAttemptTracker.RecordFailedAttempt(ctx, email)
			_ = attempts
			_ = lockout
			_ = err
		}
		_ = h.authService.auditLogger.LogAuthEvent(ctx, AuditEventLoginFailed, "", ipAddress, userAgent, false, map[string]interface{}{
			"email":  email,
			"reason": "user_not_found",
		})
		return nil, ErrInvalidCredentials
	}

	uid := user.GetID()

	// Verify password
	valid, err := h.authService.Account.VerifyPassword(ctx, uid, password)
	if err != nil || !valid {
		if h.authService.loginAttemptTracker != nil {
			attempts, lockout, err := h.authService.loginAttemptTracker.RecordFailedAttempt(ctx, email)
			_ = attempts
			_ = lockout
			_ = err
		}
		_ = h.authService.auditLogger.LogAuthEvent(ctx, AuditEventLoginFailed, uid, ipAddress, userAgent, false, map[string]interface{}{
			"email":  email,
			"reason": "invalid_password",
		})
		return nil, ErrInvalidCredentials
	}

	// Clear failed attempts on successful login
	if h.authService.loginAttemptTracker != nil {
		err := h.authService.loginAttemptTracker.ClearAttempts(ctx, email)
		_ = err
	}

	// Create session
	session, err := h.authService.Session.CreateSession(ctx, &user, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	token := ""
	if sa, ok := any(session).(interface{ GetToken() string }); ok {
		token = sa.GetToken()
	} else if s2, ok := any(session).(*auth.Session); ok {
		token = s2.Token
	}

	return &LoginResult{
		User:    user,
		Session: session,
		Token:   token,
	}, nil
}

// RegisterResult contains the result of an email+password registration.
type RegisterResult struct {
	// User is the newly created user
	User auth.User
	// Session is the newly created session (auto-login)
	Session *auth.Session
	// Token is the session token
	Token string
}

// Register registers a new user with email and password programmatically.
func (h *EmailPasswordHandlers) Register(ctx context.Context, name, email, password, ipAddress, userAgent string) (*RegisterResult, error) {
	// Create user with password
	user, err := h.authService.User.CreateUserWithEmail(ctx, name, email, password)
	if err != nil {
		return nil, err
	}

	// Set email (already done by CreateUserWithEmail, but if we want to be explicit or if there's extra logic in UpdateUserEmail)
	// Actually CreateUserWithEmail sets the email in the store.

	// Create session (auto-login)
	session, err := h.authService.Session.CreateSession(ctx, &user, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	token := ""
	if sa, ok := any(session).(interface{ GetToken() string }); ok {
		token = sa.GetToken()
	} else if s2, ok := any(session).(*auth.Session); ok {
		token = s2.Token
	}

	return &RegisterResult{
		User:    user,
		Session: session,
		Token:   token,
	}, nil
}

// ========== REQUEST STRUCTS ==========

// LoginRequest represents the JSON payload for email+password login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents the JSON payload for email+password registration.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ========== END OF PROGRAMMATIC FUNCTIONS ==========
