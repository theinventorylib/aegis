package core

// This file restores exported identifiers that existed through v1.6.0 but were
// unexported or removed during the internal API cleanup. They are thin
// compatibility shims so downstream v1 users keep compiling; new code should
// use the replacements noted on each symbol. Remove in v2.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// EmailRegexPattern is the RFC 5322 (simplified) email validation pattern.
//
// Deprecated: email validation uses ozzo-validation's is.Email; this constant
// is kept for v1 compatibility.
const EmailRegexPattern = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`

// ═══════════════════════════════════════════════════════════════════════════
// Service constructors
// ═══════════════════════════════════════════════════════════════════════════

// NewSessionService creates a session service.
//
// Deprecated: construct the stack with NewAuthService (or aegis.New); the
// session service is available as AuthService.Session.
func NewSessionService(userStore auth.UserStore, sessionStore auth.SessionStore, cfg *SessionConfig, auditLogger AuditLogger, logger Logger) *SessionService {
	return newSessionService(userStore, sessionStore, cfg, auditLogger, logger)
}

// NewAccountService creates an account service.
//
// Deprecated: construct the stack with NewAuthService (or aegis.New). A
// standalone account service has no session-cache invalidator wired, so
// password changes will not purge Redis session entries.
func NewAccountService(accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger, transactor auth.Transactor, logger Logger) *AccountService {
	return newAccountService(accountStore, sessionStore, hashConfig, authConfig, auditLogger, transactor, logger)
}

// NewUserService creates a user service.
//
// Deprecated: construct the stack with NewAuthService (or aegis.New); the user
// service is available as AuthService.User.
func NewUserService(userStore auth.UserStore, accountStore auth.AccountStore, sessionStore auth.SessionStore, hashConfig *PasswordHasherConfig, authConfig *AuthConfig, auditLogger AuditLogger) *UserService {
	return newUserService(userStore, accountStore, sessionStore, hashConfig, authConfig, auditLogger, nil, nil)
}

// NewVerificationService creates a verification service.
//
// Deprecated: construct the stack with NewAuthService (or aegis.New); the
// verification service is available as AuthService.Verification.
func NewVerificationService(store auth.VerificationStore, auditLogger AuditLogger) *VerificationService {
	return newVerificationService(store, auditLogger)
}

// NewPluginData creates an empty PluginData container.
//
// Deprecated: use NewAuthService; plugin data is requested via
// GetPluginData/GetUserExtension.
func NewPluginData() *PluginData {
	return newPluginData()
}

// ═══════════════════════════════════════════════════════════════════════════
// Config accessors and defaults
// ═══════════════════════════════════════════════════════════════════════════

// GetAuthConfig returns the authentication configuration used by the service.
//
// Deprecated: read configuration from the config.Config / AuthConfig you
// passed in; the service does not expose it anymore.
func (as *AuthService) GetAuthConfig() *AuthConfig {
	return as.authConfig
}

// DefaultPasswordHasherConfig returns default password hashing configuration.
//
// Deprecated: NewAuthService applies these defaults when hashConfig is nil.
func DefaultPasswordHasherConfig() *PasswordHasherConfig {
	return defaultPasswordHasherConfig()
}

// DefaultPasswordPolicyConfig returns default password policy configuration.
//
// Deprecated: NewAuthService applies these defaults when PasswordPolicy is nil.
func DefaultPasswordPolicyConfig() *PasswordPolicyConfig {
	return defaultPasswordPolicyConfig()
}

// AuthRateLimitConfig returns stricter limits for authentication endpoints.
//
// Deprecated: configure rate limiting via config.WithRateLimiting /
// WithRateLimitConfig.
func AuthRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: AuthRateLimitRequests,
		WindowDuration:    DefaultRateLimitWindow,
		KeyPrefix:         AuthRateLimitKeyPrefix,
		ByIP:              true,
		ByUser:            false,
		ExcludePaths:      []string{},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Validation helpers
// ═══════════════════════════════════════════════════════════════════════════

// ValidatePassword validates password strength against policy (nil = defaults).
//
// Deprecated: use AuthService.ValidatePassword, which uses the configured
// policy; UserService.CreateUser and AccountService.UpdatePassword enforce it
// automatically.
func ValidatePassword(password string, policy *PasswordPolicyConfig) error {
	return validatePassword(password, policy)
}

// ValidatePasswordSimple validates only password length.
//
// Deprecated: use AuthService.ValidatePassword with a configured policy.
func ValidatePasswordSimple(password string, minLength int) error {
	return validatePasswordSimple(password, minLength)
}

// BindAndValidate decodes a JSON request body into T and calls T.Validate.
//
// Deprecated: use core.ReadJSON plus your own validation.
func BindAndValidate[T interface{ Validate() error }](r *http.Request) (T, error) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := req.Validate(); err != nil {
		return req, err
	}
	return req, nil
}

// ValidateMiddleware validates a request body and passes it to the handler.
//
// Deprecated: use core.ReadJSON plus your own validation in the handler.
func ValidateMiddleware[T interface{ Validate() error }](
	handler func(w http.ResponseWriter, r *http.Request, req T),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := BindAndValidate[T](r)
		if err != nil {
			if vErrs := GetValidationErrors(err); vErrs != nil {
				WriteJSON(w, http.StatusBadRequest, &Response{
					Success: false,
					Error:   "validation failed",
					Data:    vErrs,
				})
				return
			}
			WriteJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		handler(w, r, req)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Error helpers
// ═══════════════════════════════════════════════════════════════════════════

// WrapError wraps an error with additional context.
//
// Deprecated: use fmt.Errorf("%s: %w", message, err).
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsValidationError reports whether err is a ValidationError.
//
// Deprecated: use errors.As with ValidationError.
func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

// ═══════════════════════════════════════════════════════════════════════════
// Context helpers
// ═══════════════════════════════════════════════════════════════════════════

// MustGetEnrichedUser extracts the enriched user, panicking if absent.
//
// Deprecated: use GetEnrichedUser and check for nil.
func MustGetEnrichedUser(ctx context.Context) *EnrichedUser {
	eu := GetEnrichedUser(ctx)
	if eu == nil {
		panic("MustGetEnrichedUser called without enriched user in context")
	}
	return eu
}

// MustGetUser extracts the user, panicking if absent.
//
// Deprecated: use GetUser and handle the error.
func MustGetUser(ctx context.Context) *auth.User {
	user, err := GetUser(ctx)
	if err != nil {
		panic("MustGetUser called without authenticated user in context")
	}
	return user
}

// IsContextInitialized reports whether AegisContextMiddleware has run.
//
// Deprecated: rely on the middleware chain; this is no longer used internally.
func IsContextInitialized(ctx context.Context) bool {
	initialized, ok := ctx.Value(contextInitializedKey).(bool)
	return ok && initialized
}

// AegisContext is a builder for Aegis-enriched contexts.
//
// Deprecated: use the WithUser/WithSession/... context helpers directly.
type AegisContext struct {
	ctx context.Context
}

// NewAegisContext creates a context builder from an existing context.
//
// Deprecated: use the WithUser/WithSession/... context helpers directly.
func NewAegisContext(ctx context.Context) *AegisContext {
	return &AegisContext{ctx: ctx}
}

// WithUser adds a user (and enriched user) to the context.
//
// Deprecated: use WithUser + WithEnrichedUser.
func (ac *AegisContext) WithUser(user *auth.User) *AegisContext {
	ac.ctx = WithUser(ac.ctx, user)
	ac.ctx = WithEnrichedUser(ac.ctx, NewEnrichedUser(user))
	return ac
}

// WithSession adds a session to the context.
//
// Deprecated: use WithSession.
func (ac *AegisContext) WithSession(session *auth.Session) *AegisContext {
	ac.ctx = WithSession(ac.ctx, session)
	return ac
}

// WithRequestID adds a request ID to the context.
//
// Deprecated: use WithRequestID.
func (ac *AegisContext) WithRequestID(id string) *AegisContext {
	ac.ctx = WithRequestID(ac.ctx, id)
	return ac
}

// WithRequestMeta adds request metadata to the context.
//
// Deprecated: use WithRequestMeta.
func (ac *AegisContext) WithRequestMeta(meta *RequestMeta) *AegisContext {
	ac.ctx = WithRequestMeta(ac.ctx, meta)
	return ac
}

// WithPluginData adds plugin data to the context.
//
// Deprecated: use WithPluginData + NewPluginData.
func (ac *AegisContext) WithPluginData() *AegisContext {
	ac.ctx = WithPluginData(ac.ctx, NewPluginData())
	return ac
}

// WithExtension adds a user extension to the context.
//
// Deprecated: use ExtendUser.
func (ac *AegisContext) WithExtension(key string, value any) *AegisContext {
	ExtendUser(ac.ctx, key, value)
	return ac
}

// Context returns the built context.
//
// Deprecated: part of the AegisContext builder.
func (ac *AegisContext) Context() context.Context {
	return ac.ctx
}

// ═══════════════════════════════════════════════════════════════════════════
// Sanitization / utility helpers
// ═══════════════════════════════════════════════════════════════════════════

// SanitizeFilename sanitizes filenames to prevent directory traversal.
//
// Deprecated: use filepath.Base at the call site; not all traversal is
// filename-shaped.
func SanitizeFilename(filename string) string { return sanitizeFilename(filename) }

// SanitizeHTML escapes HTML entities so content can be displayed safely.
//
// Deprecated: use html.EscapeString or a template engine's auto-escaping.
func SanitizeHTML(content string) string { return sanitizeHTML(content) }

// SanitizeSQL strips SQL comment patterns as defense-in-depth.
//
// Deprecated: use parameterized queries; this does not make input safe.
func SanitizeSQL(input string) string { return sanitizeSQL(input) }

// StripTags removes all HTML tags from a string.
//
// Deprecated: use SanitizeString with StripHTML, or a template engine.
func StripTags(input string) string { return stripTags(input) }

// NormalizeWhitespace collapses runs of whitespace into single spaces.
//
// Deprecated: use SanitizeString, which normalizes by default.
func NormalizeWhitespace(input string) string { return normalizeWhitespace(input) }

// SanitizeSQLIdentifier validates/normalizes a SQL identifier.
//
// Deprecated: internal schema helper; do not pass untrusted input as identifiers.
func SanitizeSQLIdentifier(name string) string { return sanitizeSQLIdentifier(name) }

// RedactForLog returns a masked identifier suitable for logs.
//
// Deprecated: implement redaction in your logging layer.
func RedactForLog(s string) string { return redactForLog(s) }

// HashShort returns the first 8 hex chars of SHA-256(input).
//
// Deprecated: use hashTokenHex-style full digests or your own hashing.
func HashShort(s string) string { return hashShort(s) }

// HashTokenHex returns the SHA-256 hex digest of a token.
//
// Deprecated: session/verification tokens are hashed internally now.
func HashTokenHex(token string) string { return hashTokenHex(token) }

// IsHashedToken reports whether s looks like a SHA-256 hex digest.
//
// Deprecated: internal migration helper.
func IsHashedToken(s string) bool { return isHashedToken(s) }

// BoolPtr returns a pointer to b.
//
// Deprecated: use a local helper or a pointer to a bool variable.
func BoolPtr(b bool) *bool { return &b }

// IDGeneratorFunc is a custom ID generation function.
//
// Deprecated: SetCustomIDGenerator takes a plain func() string.
type IDGeneratorFunc func() string

// ═══════════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════════

// AccountModel is the account model contract previously used by the service
// layer.
//
// Deprecated: auth.Account is the concrete model; this interface is unused.
type AccountModel interface {
	GetID() string
	SetID(string)
	GetUserID() string
	SetUserID(string)
	GetProvider() string
	SetProvider(string)
	GetPasswordHash() string
	SetPasswordHash(string)
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)
	GetAccessToken() string
	SetAccessToken(string)
	GetRefreshToken() string
	SetRefreshToken(string)
	GetProviderAccountID() string
	SetProviderAccountID(string)
}

// VerificationModel is the verification model contract previously used by the
// service layer.
//
// Deprecated: auth.Verification is the concrete model; this interface is unused.
type VerificationModel interface {
	GetID() string
	SetID(string)
	GetToken() string
	SetToken(string)
	GetIdentifier() string
	SetIdentifier(string)
	SetCreatedAt(time.Time)
	GetExpiresAt() time.Time
	SetExpiresAt(time.Time)
}

// ═══════════════════════════════════════════════════════════════════════════
// Audit logging adapter
// ═══════════════════════════════════════════════════════════════════════════

// LoggerAuditLogger implements AuditLogger using a structured logger.
//
// Deprecated: config.WithAuditLogger accepts any AuditLogger; adapt your
// logger directly.
type LoggerAuditLogger struct {
	logger interface {
		Info(msg string, keysAndValues ...any)
		Error(msg string, keysAndValues ...any)
		Debug(msg string, keysAndValues ...any)
	}
}

// NewLoggerAuditLogger creates an audit logger writing to a structured logger.
//
// Deprecated: adapt your logger to AuditLogger directly.
func NewLoggerAuditLogger(logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
}) *LoggerAuditLogger {
	return &LoggerAuditLogger{logger: logger}
}

// LogEvent implements AuditLogger.
func (l *LoggerAuditLogger) LogEvent(_ context.Context, event *AuditEvent) error {
	if l.logger == nil {
		return nil
	}
	fields := []any{
		"event_type", event.EventType,
		"user_id", event.UserID,
		"ip_address", event.IPAddress,
		"user_agent", event.UserAgent,
		"resource", event.Resource,
		"action", event.Action,
		"success", event.Success,
		"timestamp", event.Timestamp,
	}
	if event.Error != "" {
		fields = append(fields, "error", event.Error)
	}
	for k, v := range event.Details {
		fields = append(fields, k, v)
	}
	l.logger.Info("audit event", fields...)
	return nil
}

// LogAuthEvent implements AuditLogger.
func (l *LoggerAuditLogger) LogAuthEvent(ctx context.Context, eventType AuditEventType, userID string, success bool, details map[string]any) error {
	return l.LogEvent(ctx, &AuditEvent{
		ID:        GenerateID(),
		EventType: eventType,
		UserID:    userID,
		IPAddress: GetIPAddress(ctx),
		UserAgent: GetUserAgent(ctx),
		Details:   details,
		Timestamp: time.Now(),
		Success:   success,
	})
}
