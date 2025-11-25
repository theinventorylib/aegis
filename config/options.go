// Package config provides configuration types and options for Aegis.
package config

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/server"
)

// Logger is an optional interface for logging Aegis lifecycle events.
// Implementations can integrate with any logging framework (zap, logrus, slog, etc).
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// Config holds the configuration for Aegis authentication framework.
//
// Required fields:
//   - DB: Database provider (use WithDB, WithPostgres, or WithMySQL)
//   - Router: HTTP router (use WithRouter)
//   - CSRFSecret: 32+ byte random secret (required for web apps, or use WithAPIOnlyMode)
//
// All other fields have secure defaults and are optional.
// See SECURITY.md for production security recommendations.
type Config struct {
	// ========== REQUIRED DEPENDENCIES ==========

	// DB is the database provider for storing users, sessions, and auth data.
	// REQUIRED. Use WithDB, WithPostgres, or WithMySQL to set this.
	DB db.Provider

	// Router is the HTTP router for mounting authentication endpoints.
	// REQUIRED. Use WithRouter to set this.
	Router server.Router

	// ========== OPTIONAL OBSERVABILITY ==========

	// Logger is an optional logger for Aegis lifecycle events.
	// OPTIONAL. Use WithLogger to enable logging.
	// Default: nil (no logging)
	Logger Logger

	// ========== INTERNAL (DO NOT SET DIRECTLY) ==========

	// dbError tracks database connection errors from WithPostgres/WithMySQL.
	// Internal use only. DO NOT set directly.
	dbError error

	// ========== SECURITY CONFIGURATION ==========

	// CSRFSecret is the secret key for CSRF token generation.
	// REQUIRED for web applications with browser-based sessions.
	// MUST be cryptographically random (32+ bytes recommended).
	// OPTIONAL if using WithAPIOnlyMode(true).
	// Use WithCSRFSecret to set this.
	CSRFSecret []byte

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

	// ========== REDIS (OPTIONAL) ==========

	// Redis is optional configuration for Redis-based session storage.
	// OPTIONAL.
	// Default: nil (uses database for session storage)
	// Use WithRedis to enable Redis sessions.
	Redis *RedisConfig
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
		SessionExpiry:   24 * time.Hour,
		RefreshExpiry:   7 * 24 * time.Hour,
		CookieHTTPOnly:  true,
		CookieSecure:    true,
		CookieSameSite:  "Lax",
		Argon2Time:      1,
		Argon2Memory:    64 * 1024, // 64 MB
		Argon2Threads:   4,
		Argon2KeyLength: 32,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check for database connection errors from WithPostgres/WithMySQL
	if c.dbError != nil {
		return fmt.Errorf("database configuration error: %w", c.dbError)
	}

	// Verify required dependencies with explicit errors
	if c.DB == nil {
		return errors.New("database provider is required: use WithDB, WithPostgres, or WithMySQL")
	}
	if c.Router == nil {
		return errors.New("router is required: use WithRouter")
	}

	// CSRF secret only required for web apps (not API-only mode)
	if !c.APIMode && len(c.CSRFSecret) == 0 {
		return errors.New("CSRF secret is required (or set APIMode=true for API-only apps)")
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

// Option is a functional option for configuring Aegis
type Option func(*Config)

// WithDB sets the database provider from a standard *sql.DB connection
// db: a *sql.DB connection from any driver (pgx, lib/pq, mysql, sqlite, etc.)
// dialect: the SQL dialect for query syntax (db.PostgreSQL, db.MySQL, db.SQLite)
//
// Example:
//
//	import "database/sql"
//	import _ "github.com/lib/pq"
//
//	sqlDB, _ := sql.Open("postgres", connString)
//	aegis.New(config.WithDB(sqlDB, db.PostgreSQL), ...)
func WithDB(sqlDB interface{}, dialect db.Dialect) Option {
	return func(c *Config) {
		// Support both *sql.DB and db.Provider for flexibility
		switch v := sqlDB.(type) {
		case *sql.DB:
			c.DB = db.NewSQLProvider(v, dialect)
		case db.Provider:
			// Allow passing DBProvider directly for advanced use cases
			c.DB = v
		}
	}
}

// WithPostgres creates a PostgreSQL database provider from a connection string
// This is a convenience helper that uses the lib/pq driver
//
// Example:
//
//	aegis.New(
//	    config.WithPostgres("postgres://user:pass@localhost:5432/db?sslmode=disable"),
//	    ...
//	)
//
// For more control over the connection, use WithDB with your own *sql.DB
func WithPostgres(connString string) Option {
	return func(c *Config) {
		sqlDB, err := sql.Open("postgres", connString)
		if err != nil {
			// Store error to be caught during Validate()
			c.dbError = fmt.Errorf("failed to open postgres connection: %w", err)
			return
		}
		// Test the connection
		if err := sqlDB.Ping(); err != nil {
			c.dbError = fmt.Errorf("failed to ping postgres database: %w", err)
			_ = sqlDB.Close() // Ignore close error, ping already failed
			return
		}
		c.DB = db.NewSQLProvider(sqlDB, db.PostgreSQL)
	}
}

// WithMySQL creates a MySQL database provider from a connection string
// This is a convenience helper that uses the go-sql-driver/mysql driver
//
// Example:
//
//	aegis.New(
//	    config.WithMySQL("user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true"),
//	    ...
//	)
//
// For more control over the connection, use WithDB with your own *sql.DB
func WithMySQL(connString string) Option {
	return func(c *Config) {
		sqlDB, err := sql.Open("mysql", connString)
		if err != nil {
			c.dbError = fmt.Errorf("failed to open mysql connection: %w", err)
			return
		}
		// Test the connection
		if err := sqlDB.Ping(); err != nil {
			c.dbError = fmt.Errorf("failed to ping mysql database: %w", err)
			_ = sqlDB.Close() // Ignore close error, ping already failed
			return
		}
		c.DB = db.NewSQLProvider(sqlDB, db.MySQL)
	}
}

// WithRouter sets the router
func WithRouter(router server.Router) Option {
	return func(c *Config) {
		c.Router = router
	}
}

// WithCSRFSecret sets the CSRF protection secret
func WithCSRFSecret(secret []byte) Option {
	return func(c *Config) {
		c.CSRFSecret = secret
	}
}

// WithSessionExpiry sets the session expiry duration
func WithSessionExpiry(duration time.Duration) Option {
	return func(c *Config) {
		c.SessionExpiry = duration
	}
}

// WithRefreshExpiry sets the refresh token expiry duration
func WithRefreshExpiry(duration time.Duration) Option {
	return func(c *Config) {
		c.RefreshExpiry = duration
	}
}

// WithCookieDomain sets the cookie domain
func WithCookieDomain(domain string) Option {
	return func(c *Config) {
		c.CookieDomain = domain
	}
}

// WithCookieSecure sets whether cookies should be secure
func WithCookieSecure(secure bool) Option {
	return func(c *Config) {
		c.CookieSecure = secure
	}
}

// WithCookieSameSite sets the SameSite cookie attribute
func WithCookieSameSite(sameSite string) Option {
	return func(c *Config) {
		c.CookieSameSite = sameSite
	}
}

// WithRedis sets the Redis configuration
func WithRedis(host string, port int, password string, db int) Option {
	return func(c *Config) {
		c.Redis = &RedisConfig{
			Host:     host,
			Port:     port,
			Password: password,
			DB:       db,
		}
	}
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
func WithIDGenerator(generator func() string) Option {
	return func(c *Config) {
		c.IDGenerator = generator
	}
}

// WithAPIOnlyMode enables API-only mode (skips CSRF secret requirement)
// Use this when building APIs without web UI
func WithAPIOnlyMode(enabled bool) Option {
	return func(c *Config) {
		c.APIMode = enabled
	}
}

// WithLogger sets an optional logger for observability of Aegis lifecycle events.
// The logger will receive events for plugin registration, initialization, and errors.
// Example: WithLogger(slog.Default()) or a custom logger implementation
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}
