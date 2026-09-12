// Package core provides the high-level authentication and authorization business logic.
// It orchestrates the low-level data operations from the auth package and adds
// features like password hashing, session management, rate limiting, audit logging,
// and more.
//
// The core package is organized around specialized service objects:
//   - AuthService: Main orchestrator that coordinates all sub-services
//   - UserService: User identity management
//   - AccountService: Provider-specific account management and authentication
//   - SessionService: Session creation, validation, and caching
//   - VerificationService: Token generation and validation for email/password flows
//
// All services accept context for cancellation and use the audit logger for
// security event tracking.
package core

import (
	"context"

	"github.com/theinventorylib/aegis/v2/auth"
)

// AuthService is the main orchestrator for authentication operations.
// It coordinates specialized sub-services and provides centralized access to
// authentication functionality throughout the application.
//
// AuthService manages:
//   - Password hashing configuration
//   - Audit logging
//   - Login attempt tracking for account lockout
//   - Authentication policy configuration
//
// It provides four sub-services that handle specific domains:
//   - User: User CRUD operations
//   - Account: Authentication account management
//   - Session: Session lifecycle and caching
//   - Verification: Token-based verification flows
//
// AuthService should be initialized once at application startup and shared
// across HTTP handlers and middleware.
type AuthService struct {
	// hashConfig defines Argon2id parameters for password hashing
	hashConfig *PasswordHasherConfig

	// auditLogger records security events (logins, failures, etc.)
	auditLogger AuditLogger

	// loginAttemptTracker prevents brute force attacks via account lockout
	loginAttemptTracker *LoginAttemptTracker

	// authConfig holds authentication policy settings
	authConfig *AuthConfig

	// Sub-services for specialized operations
	User          *UserService
	Account       *AccountService
	Session       *SessionService
	Verification  *VerificationService
	EmailPassword *EmailPasswordHandlers

	// Internal storage references (kept for service re-instantiation)
	userStore         auth.UserStore
	accountStore      auth.AccountStore
	sessionStore      auth.SessionStore
	verificationStore auth.VerificationStore

	// emailVerificationCheck, when set by an email plugin, reports whether a
	// user's email is verified. Consulted only when
	// authConfig.RequireEmailVerification is true.
	emailVerificationCheck func(ctx context.Context, user auth.User) (bool, error)
}

// SetEmailVerificationCheck wires the email-verification checker used when
// AuthConfig.RequireEmailVerification is enabled. Email plugins call this
// during Init. Passing nil disables the check.
func (as *AuthService) SetEmailVerificationCheck(fn func(ctx context.Context, user auth.User) (bool, error)) {
	as.emailVerificationCheck = fn
}

// ValidatePassword checks password against the configured password policy.
// Exposed so plugins and applications that create credentials outside
// UserService enforce the same policy.
func (as *AuthService) ValidatePassword(password string) error {
	return validatePassword(password, as.authConfig.PasswordPolicy)
}

// SetEmailVerificationResetter wires the callback invoked when a user's email
// changes, so the email plugin can clear its verification flag. Email plugins
// call this during Init. Passing nil disables the reset.
func (as *AuthService) SetEmailVerificationResetter(fn func(ctx context.Context, userID, email string) error) {
	as.User.setEmailVerificationReset(fn)
}

// NewAuthService creates a new AuthService with all sub-services initialized.
//
// Parameters:
//   - authConfig: Authentication policy configuration (session duration, password
//     policy, etc.). If nil, defaults are used.
//   - authConn: Connection to the auth storage layer providing access to stores.
//   - hashConfig: Argon2id password hashing parameters. If nil, secure OWASP-
//     recommended defaults are used.
//   - auditLogger: Interface for logging security events. If nil, a no-op logger
//     is used (events are silently discarded).
//   - loginAttemptTracker: Tracks failed login attempts for account lockout.
//     Can be nil if brute force protection is not needed.
//
// The function ensures all nil inputs are replaced with safe defaults, so it
// will never return a partially-configured service.
func NewAuthService(authConfig *AuthConfig, authConn *auth.Auth, hashConfig *PasswordHasherConfig, auditLogger AuditLogger, loginAttemptTracker *LoginAttemptTracker, logger Logger) *AuthService {
	if hashConfig == nil {
		hashConfig = defaultPasswordHasherConfig()
	}
	if authConfig == nil {
		authConfig = DefaultAuthConfig()
	}
	if authConfig.PasswordPolicy == nil {
		authConfig.PasswordPolicy = defaultPasswordPolicyConfig()
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}
	if logger == nil {
		logger = noopLogger{}
	}

	as := &AuthService{
		hashConfig:          hashConfig,
		auditLogger:         auditLogger,
		loginAttemptTracker: loginAttemptTracker,
		authConfig:          authConfig,
		userStore:           authConn.UserStore(),
		accountStore:        authConn.AccountStore(),
		sessionStore:        authConn.SessionStore(),
		verificationStore:   authConn.VerificationStore(),
	}

	// Initialize sub-services
	as.Account = newAccountService(as.accountStore, as.sessionStore, hashConfig, authConfig, auditLogger, authConn.Transactor(), logger)
	as.User = newUserService(as.userStore, as.accountStore, as.sessionStore, hashConfig, authConfig, auditLogger, authConn.Transactor(), logger)
	as.Verification = newVerificationService(as.verificationStore, auditLogger)
	as.Session = newSessionService(as.userStore, as.sessionStore, nil, auditLogger, logger)
	as.Account.setSessionInvalidator(as.Session.purgeUserSessionCache)
	as.User.setSessionCachePurger(as.Session.purgeUserSessionCache)
	as.EmailPassword = NewEmailPasswordHandlers(as)

	return as
}

// GetUserFieldsConfig returns the user fields configuration which controls
// which user fields are included or excluded in API responses.
//
// Returns nil if not configured, meaning all fields are included by default.
func (as *AuthService) GetUserFieldsConfig() *UserFieldsConfig {
	if as.authConfig == nil {
		return nil
	}
	return as.authConfig.UserFields
}
