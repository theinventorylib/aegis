// Package config provides configuration types and options for Aegis.
package config

import (
	"database/sql"
	"errors"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router"
)

// Logger is an optional interface for logging Aegis lifecycle events.
// Implementations can integrate with any logging framework (zap, logrus, slog, etc).
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
}

// Dialect is the database engine selector. It is an alias for auth.Dialect.
type Dialect = auth.Dialect

const (
	// DialectPostgres selects PostgreSQL.
	DialectPostgres = auth.DialectPostgres
	// DialectMySQL selects MySQL / MariaDB.
	DialectMySQL = auth.DialectMySQL
	// DialectSQLite selects SQLite.
	DialectSQLite = auth.DialectSQLite
)

// Config holds the configuration for Aegis authentication framework.
//
// Required fields:
//   - DB: sql.DB
//   - Router: HTTP router (use WithRouter)
//   - CSRFSecret: 32+ byte random secret (required for web apps, or use WithAPIOnlyMode)
//
// All other fields have secure defaults and are optional.
// See SECURITY.md for production security recommendations.
type Config struct {
	// ========== REQUIRED DEPENDENCIES ==========

	// DB is the database provider for storing users, sessions, and auth data.
	// Use a pointer so nil can represent "not provided" and to avoid copying
	// a large sql.DB value accidentally.
	DB *sql.DB

	// Router is the HTTP router for mounting authentication endpoints.
	// REQUIRED. Use WithRouter to set this.
	Router router.Router

	// ========== OPTIONAL OBSERVABILITY ==========

	// Logger is an optional logger for Aegis lifecycle events.
	// OPTIONAL. Use WithLogger to enable logging.
	// Default: nil (no logging)
	Logger Logger

	// AuditLogger is an optional audit logger for security events.
	// OPTIONAL. Use WithAuditLogger to enable audit logging.
	// Default: nil (no audit logging)
	AuditLogger core.AuditLogger

	// ========== SECURITY CONFIGURATION ==========

	// Secret is the master secret for Aegis.
	// REQUIRED for web applications (or use WithAPIOnlyMode for API-only apps).
	// MUST be cryptographically random (32+ bytes recommended).
	// All plugin-specific secrets (CSRF, OAuth, JWT, etc.) are derived from this.
	// Use WithSecret to set this.
	Secret []byte

	// SessionExpiry is the duration before a session token expires.
	// REQUIRED (has default).
	// Default: 24 hours
	// Production recommendation: 15 minutes to 24 hours, depending on sensitivity.
	// Use WithSessionExpiry to override.
	SessionExpiry time.Duration

	// RefreshExpiry is the duration before a refresh token expires.
	// REQUIRED (has default).
	// Default: 7 days
	// MUST be greater than SessionExpiry.
	// Production recommendation: 7-30 days.
	// Use WithRefreshExpiry to override.
	RefreshExpiry time.Duration

	// CookieDomain is the domain for session cookies.
	// OPTIONAL.
	// Default: "" (cookie applies to current domain only)
	// Set to ".example.com" to share cookies across subdomains.
	// Use WithCookieDomain to set this.
	CookieDomain string

	// CookieName is the name of the session cookie.
	// OPTIONAL.
	// Default: "aegis_session"
	// Use WithCookieName to customize.
	CookieName string

	// CookieSecure determines if cookies are sent only over HTTPS.
	// REQUIRED (has default).
	// Default: true
	// MUST be true in production for security.
	// Use WithCookieSecure to override (only for local development).
	CookieSecure bool

	// CookieHTTPOnly prevents JavaScript from accessing cookies.
	// REQUIRED (has default).
	// Default: true
	// MUST be true in production to prevent XSS attacks.
	// DO NOT override in production.
	CookieHTTPOnly bool

	// CookieSameSite controls cross-site cookie behavior.
	// REQUIRED (has default).
	// Default: "Lax"
	// Options: "Strict" (maximum protection), "Lax" (balanced), "None" (cross-site, requires Secure=true)
	// Production recommendation: "Strict" or "Lax"
	// Use WithCookieSameSite to override.
	CookieSameSite string

	// ========== API MODE ==========

	// APIMode skips CSRF secret requirement for API-only applications.
	// OPTIONAL.
	// Default: false
	// Set to true if building an API without browser-based sessions.
	// Use WithAPIOnlyMode to enable.
	APIMode bool

	// BearerAuth enables Bearer token authentication via the Authorization header.
	// When enabled, clients can authenticate with "Authorization: Bearer <token>".
	// OPTIONAL.
	// Default: false (automatically enabled when APIMode is true)
	// Use WithBearerAuth to enable explicitly.
	// Can be disabled even in APIMode with WithBearerAuth(false).
	BearerAuth *bool

	// ========== PASSWORD HASHING (ARGON2ID) ==========

	// Argon2Time is the number of iterations for Argon2id hashing.
	// REQUIRED (has default).
	// Default: 1
	// Production recommendation: 1-3 iterations (higher = more secure but slower).
	// See SECURITY.md for detailed guidance.
	Argon2Time uint32

	// Argon2Memory is the memory cost in KB for Argon2id hashing.
	// REQUIRED (has default).
	// Default: 64 MB (65536 KB)
	// Production recommendation: 64-256 MB depending on server resources.
	// See SECURITY.md for detailed guidance.
	Argon2Memory uint32

	// Argon2Threads is the degree of parallelism for Argon2id hashing.
	// REQUIRED (has default).
	// Default: 4
	// Production recommendation: 4 threads is optimal for most servers.
	Argon2Threads uint8

	// Argon2KeyLength is the output key length in bytes for Argon2id.
	// REQUIRED (has default).
	// Default: 32 bytes (256-bit security)
	// DO NOT change unless you have specific requirements.
	Argon2KeyLength uint32

	// ========== ID GENERATION ==========

	// IDGenerator is an optional custom function for generating IDs.
	// OPTIONAL.
	// Default: nil (uses built-in ULID generation - sortable, time-based, 26 characters)
	// Override with: UUID, sequential IDs, or custom format (KSUID, nanoid, etc.)
	// Example: WithIDGenerator(func() string { return uuid.New().String() })
	// Use WithIDGenerator to set this.
	IDGenerator func() string

	// IDStrategy defines the algorithm used for generating unique identifiers.
	// OPTIONAL.
	// Default: core.IDStrategyULID
	// Options: core.IDStrategyULID, core.IDStrategyUUID, core.IDStrategyCustom
	// Use WithIDStrategy to set this.
	IDStrategy core.IDStrategy

	// ========== REDIS (OPTIONAL) ==========

	// Redis is optional configuration for Redis-based session storage.
	// OPTIONAL.
	// Default: nil (uses database for session storage)
	// Use WithRedis to enable Redis sessions.
	Redis *RedisConfig

	// ========== AUTH CONFIGURATION ==========

	// Auth holds low-level authentication configuration for database operations.
	// REQUIRED. This is typically initialized automatically from the main DB connection.
	Auth auth.Config

	// CoreAuth holds high-level core authentication service configuration.
	// OPTIONAL.
	// Default: DefaultAuthConfig()
	// Use WithAuthConfig to set this.
	CoreAuth *core.AuthConfig

	// ========== RATE LIMITING ==========

	// RateLimitEnabled enables rate limiting middleware.
	// OPTIONAL.
	// Default: false
	// Use WithRateLimiting to enable.
	RateLimitEnabled bool

	// RateLimitConfig holds rate limiting configuration.
	// OPTIONAL.
	// Default: nil (uses DefaultRateLimitConfig() if enabled)
	// Use WithRateLimiting or WithRateLimitConfig to configure.
	RateLimitConfig *core.RateLimitConfig

	// LoginAttemptConfig holds login attempt tracking configuration.
	// OPTIONAL.
	// Default: nil (uses DefaultLoginAttemptConfig() if rate limiting enabled)
	// Use WithLoginAttemptConfig to configure.
	LoginAttemptConfig *core.LoginAttemptConfig
}

// RedisConfig holds configuration for Redis session storage.
type RedisConfig struct {
	// Host is the Redis server hostname or IP address.
	// REQUIRED if using Redis.
	Host string

	// Port is the Redis server port.
	// REQUIRED if using Redis.
	// Default Redis port: 6379
	Port int

	// Password is the Redis authentication password.
	// REQUIRED in production for security.
	// See SECURITY.md for Redis security recommendations.
	Password string

	// DB is the Redis database number (0-15).
	// OPTIONAL.
	// Default: 0
	DB int
}

// Default returns a Config with sensible defaults
func Default() *Config {
	return &Config{
		SessionExpiry:   core.DefaultSessionExpiry,
		RefreshExpiry:   core.DefaultRefreshExpiry,
		CookieName:      core.DefaultCookieName,
		CookieHTTPOnly:  core.DefaultCookieHTTPOnly,
		CookieSecure:    core.DefaultCookieSecure,
		CookieSameSite:  core.DefaultCookieSameSite,
		Argon2Time:      core.DefaultArgon2Time,
		Argon2Memory:    core.DefaultArgon2Memory,
		Argon2Threads:   core.DefaultArgon2Threads,
		Argon2KeyLength: core.DefaultArgon2KeyLength,
		CoreAuth:        core.DefaultAuthConfig(),
		IDStrategy:      core.IDStrategyULID,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DB == nil {
		return errors.New("database (DB) is required")
	}

	if c.Router == nil {
		return errors.New("router is required: use WithRouter")
	}

	// Secret only required for web apps (not API-only mode)
	if !c.APIMode && len(c.Secret) == 0 {
		return errors.New("secret is required (use WithSecret) or set APIMode=true for API-only apps")
	}

	// Validate security-sensitive parameters
	if c.SessionExpiry <= 0 {
		return errors.New("session expiry must be positive")
	}
	if c.RefreshExpiry <= 0 {
		return errors.New("refresh expiry must be positive")
	}
	if c.RefreshExpiry < c.SessionExpiry {
		return errors.New("refresh expiry should be greater than session expiry")
	}

	// Validate Argon2 parameters are within reasonable bounds
	if c.Argon2Time == 0 {
		return errors.New("Argon2 time parameter must be positive")
	}
	if c.Argon2Memory == 0 {
		return errors.New("Argon2 memory parameter must be positive")
	}
	if c.Argon2Threads == 0 {
		return errors.New("Argon2 threads parameter must be positive")
	}
	if c.Argon2KeyLength == 0 {
		return errors.New("Argon2 key length must be positive")
	}

	return nil
}

// IsBearerAuthEnabled returns whether Bearer token authentication should be enabled.
// Returns true if:
//   - BearerAuth is explicitly set to true via WithBearerAuth(true), OR
//   - APIMode is true and BearerAuth was not explicitly disabled
func (c *Config) IsBearerAuthEnabled() bool {
	if c.BearerAuth != nil {
		return *c.BearerAuth
	}
	// Auto-enable in API mode when not explicitly set
	return c.APIMode
}

// DeriveSecret derives a purpose-specific secret from the master secret.
// This uses HKDF-SHA256 to cryptographically separate secrets for different purposes.
// Returns nil if no master secret is configured.
//
// Plugins should define their own purpose strings (e.g., "oauth-state", "jwt-signing").
// This ensures cryptographic separation between different uses.
//
// Example:
//
//	oauthSecret := cfg.DeriveSecret("oauth-state")
//	jwtSecret := cfg.DeriveSecret("jwt-signing")
func (c *Config) DeriveSecret(purpose string) []byte {
	if len(c.Secret) == 0 {
		return nil
	}
	return core.DeriveSecret(c.Secret, purpose, core.DefaultSecretLength)
}

// WithRouter sets the HTTP router for Aegis.
// router: an implementation of server.Router (ChiRouter, DefaultRouter, etc.)
//
// Example:
//
//	mux := http.NewServeMux()
//	router := server.NewDefaultRouter(mux)
//
//	aegis.New(config.WithRouter(router), ...)
func (c *Config) WithRouter(router router.Router) *Config {
	c.Router = router
	return c
}

// WithSecret sets the master secret for Aegis.
// All plugin-specific secrets (CSRF, OAuth state, JWT, etc.) are derived from this.
// The secret should be at least 32 bytes of cryptographically random data.
//
// Example:
//
//	// Generate a secure secret (do this once and store securely)
//	secret := make([]byte, 32)
//	crypto/rand.Read(secret)
//
//	aegis.New(config.WithSecret(secret), ...)
func (c *Config) WithSecret(secret []byte) *Config {
	c.Secret = secret
	return c
}

// WithSessionExpiry sets the session expiry duration
func (c *Config) WithSessionExpiry(duration time.Duration) *Config {
	c.SessionExpiry = duration
	return c
}

// WithRefreshExpiry sets the refresh token expiry duration
func (c *Config) WithRefreshExpiry(duration time.Duration) *Config {
	c.RefreshExpiry = duration
	return c
}

// WithCookieDomain sets the cookie domain
func (c *Config) WithCookieDomain(domain string) *Config {
	c.CookieDomain = domain
	return c
}

// WithCookieName sets the session cookie name
// Default is "aegis_session"
func (c *Config) WithCookieName(name string) *Config {
	c.CookieName = name
	return c
}

// WithCookieSecure sets whether cookies should be secure
func (c *Config) WithCookieSecure(secure bool) *Config {
	c.CookieSecure = secure
	return c
}

// WithCookieSameSite sets the SameSite cookie attribute
func (c *Config) WithCookieSameSite(sameSite string) *Config {
	c.CookieSameSite = sameSite
	return c
}

// WithRedis sets the Redis configuration
func (c *Config) WithRedis(host string, port int, password string, db int) *Config {
	c.Redis = &RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}
	return c
}

// WithIDGenerator sets a custom ID generation function, overriding the default ULID strategy.
//
// By default, Aegis uses ULID (Universally Unique Lexicographically Sortable Identifier)
// which provides sortable, time-based IDs that work across restarts and distributed systems.
//
// Use this option to override with:
//   - UUID: for standard UUID v4 format
//   - Custom libraries: KSUID, nanoid, etc.
//   - Database sequences: if handling at application level
//
// Examples:
//   - UUID: WithIDGenerator(func() string { return uuid.New().String() })
//   - KSUID: WithIDGenerator(func() string { return ksuid.New().String() })
func (c *Config) WithIDGenerator(generator func() string) *Config {
	c.IDGenerator = generator
	c.IDStrategy = core.IDStrategyCustom
	return c
}

// WithIDStrategy sets the global ID generation strategy for Aegis.
//
// By default, Aegis uses ULID (Universally Unique Lexicographically Sortable Identifier)
// which provides sortable, time-based IDs.
//
// Options:
//   - core.IDStrategyULID: Sortable, 26 chars (default)
//   - core.IDStrategyUUID: Random UUID v4, 36 chars
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithIDStrategy(core.IDStrategyUUID)
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithIDStrategy(strategy core.IDStrategy) *Config {
	c.IDStrategy = strategy
	return c
}

// WithAPIOnlyMode enables API-only mode (skips CSRF secret requirement)
// Use this when building APIs without web UI.
// This automatically enables Bearer token authentication.
// To disable bearer auth in API mode, call WithBearerAuth(false) after this.
func (c *Config) WithAPIOnlyMode(enabled bool) *Config {
	c.APIMode = enabled
	return c
}

// WithBearerAuth explicitly enables or disables Bearer token authentication.
// When enabled, the AuthMiddleware will accept session tokens via the
// "Authorization: Bearer <token>" header in addition to cookies.
//
// Bearer auth is automatically enabled when APIMode is true.
// Use WithBearerAuth(false) to explicitly disable it even in API mode.
//
// Example:
//
//	// Explicitly enable bearer auth for web+API dual mode
//	cfg := config.Default().
//		WithBearerAuth(true)
//
//	// API mode auto-enables bearer; explicitly disable it
//	cfg := config.Default().
//		WithAPIOnlyMode(true).
//		WithBearerAuth(false)
func (c *Config) WithBearerAuth(enabled bool) *Config {
	c.BearerAuth = &enabled
	return c
}

// WithLogger sets an optional logger for observability of Aegis lifecycle events.
// The logger will receive events for plugin registration, initialization, and errors.
// Example: WithLogger(slog.Default()) or a custom logger implementation
func (c *Config) WithLogger(logger Logger) *Config {
	c.Logger = logger
	return c
}

// WithAuditLogger sets an optional audit logger for security events.
// The audit logger will receive events for authentication attempts, user actions, etc.
// Example: WithAuditLogger(&MyAuditLogger{})
func (c *Config) WithAuditLogger(logger core.AuditLogger) *Config {
	c.AuditLogger = logger
	return c
}

// WithAuthConfig sets the core authentication configuration
func (c *Config) WithAuthConfig(authConfig *core.AuthConfig) *Config {
	c.CoreAuth = authConfig
	return c
}

// WithRateLimiting enables rate limiting with default configuration.
// Rate limiting helps protect against brute-force attacks and DoS.
// By default, it allows 100 requests per minute per IP.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithRateLimiting()
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithRateLimiting() *Config {
	c.RateLimitEnabled = true
	if c.RateLimitConfig == nil {
		c.RateLimitConfig = core.DefaultRateLimitConfig()
	}
	if c.LoginAttemptConfig == nil {
		c.LoginAttemptConfig = core.DefaultLoginAttemptConfig()
	}
	return c
}

// WithRateLimitConfig sets custom rate limiting configuration.
// This also enables rate limiting.
//
// Example:
//
//	cfg := &core.RateLimitConfig{
//	    RequestsPerWindow: 50,
//	    WindowDuration:    time.Minute,
//	    ByIP:              true,
//	}
//	aegis.New(config.WithRateLimitConfig(cfg), ...)
func (c *Config) WithRateLimitConfig(cfg *core.RateLimitConfig) *Config {
	c.RateLimitEnabled = true
	c.RateLimitConfig = cfg
	if c.LoginAttemptConfig == nil {
		c.LoginAttemptConfig = core.DefaultLoginAttemptConfig()
	}
	return c
}

// WithPasswordPolicy sets custom password validation policies.
// This controls password strength requirements for user registration.
//
// Example:
//
//	policy := &core.PasswordPolicyConfig{
//	    MinLength:      12,
//	    RequireUpper:   true,
//	    RequireLower:   true,
//	    RequireDigit:   true,
//	    RequireSpecial: true,
//	    MaxLength:      256,
//	}
//	aegis.New(config.WithPasswordPolicy(policy), ...)
func (c *Config) WithPasswordPolicy(policy *core.PasswordPolicyConfig) *Config {
	if c.CoreAuth == nil {
		c.CoreAuth = core.DefaultAuthConfig()
	}
	c.CoreAuth.PasswordPolicy = policy
	return c
}

// WithUserFields configures which extension fields are included in user API responses.
// Use this to control what plugin data (role, permissions, organizations, etc.)
// appears in user responses.
//
// If not configured, all extension fields from plugins are included by default.
//
// Example - Include only specific fields:
//
//	aegis.New(
//	    config.WithUserFields([]string{"role", "permissions", "organizations"}),
//	    ...
//	)
//
// This produces JSON responses like:
//
//	{
//	    "id": "user_123",
//	    "email": "user@example.com",
//	    "role": "admin",
//	    "permissions": ["read", "write"],
//	    "organizations": ["org1", "org2"]
//	}
//
// Note: Session endpoints (/session/validate) always return both session and
// enriched user data. This config only filters which extension fields appear
// in the user portion of responses.
func (c *Config) WithUserFields(fields []string) *Config {
	if c.CoreAuth == nil {
		c.CoreAuth = core.DefaultAuthConfig()
	}
	if c.CoreAuth.UserFields == nil {
		c.CoreAuth.UserFields = core.DefaultUserFieldsConfig()
	}
	c.CoreAuth.UserFields.Fields = fields
	return c
}

// WithDB sets the database connection for Aegis.
// This is required for storing users, sessions, and authentication data.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithDB(db)
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithDB(db *sql.DB) *Config {
	c.DB = db
	return c
}

// WithDialect sets the database dialect for sqlc-generated query selection.
// Defaults to DialectPostgres when not set.
func (c *Config) WithDialect(d Dialect) *Config {
	c.Auth.Dialect = d
	return c
}

// WithArgon2Time sets the number of iterations for Argon2id password hashing.
// Higher values increase security but also CPU time for password operations.
// Default: 1. Recommended: 1-3 depending on latency requirements.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithArgon2Time(2)
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithArgon2Time(time uint32) *Config {
	c.Argon2Time = time
	return c
}

// WithArgon2Memory sets the memory cost in KB for Argon2id password hashing.
// Higher values increase security but also memory usage.
// Default: 65536 (64 MB). Recommended: 64-256 MB depending on resources.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithArgon2Memory(128*1024) // 128 MB
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithArgon2Memory(memory uint32) *Config {
	c.Argon2Memory = memory
	return c
}

// WithArgon2Threads sets the parallelism for Argon2id password hashing.
// Default: 4 threads.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithArgon2Threads(4)
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithArgon2Threads(threads uint8) *Config {
	c.Argon2Threads = threads
	return c
}

// WithArgon2KeyLength sets the output key length for Argon2id.
// Default: 32 bytes (256-bit security). Generally should not be changed.
//
// Example:
//
//	cfg := config.Default()
//	cfg.WithArgon2KeyLength(32)
//
//	aegis.New(ctx, cfg, ...)
func (c *Config) WithArgon2KeyLength(length uint32) *Config {
	c.Argon2KeyLength = length
	return c
}
