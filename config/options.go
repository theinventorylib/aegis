package config

import (
	"errors"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/server"
)

// Config holds the configuration for Aegis
type Config struct {
	// Core dependencies
	DB     db.DBProvider
	Router server.Router

	// Security
	JWTSecret      []byte
	CSRFSecret     []byte
	SessionExpiry  time.Duration
	RefreshExpiry  time.Duration
	CookieDomain   string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string // "Lax", "Strict", or "None"

	// API Mode (skips CSRF requirement for API-only apps)
	APIMode bool

	// Password Hashing (Argon2id)
	Argon2Time      uint32
	Argon2Memory    uint32
	Argon2Threads   uint8
	Argon2KeyLength uint32

	// ID Generation (Optional custom function)
	IDGenerator func() string

	// Redis (Optional)
	Redis *RedisConfig
}

// RedisConfig holds configuration for Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
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
	if c.DB == nil {
		return errors.New("database provider is required")
	}
	if c.Router == nil {
		return errors.New("router is required")
	}
	if len(c.JWTSecret) == 0 {
		return errors.New("JWT secret is required")
	}
	// CSRF secret only required for web apps (not API-only mode)
	if !c.APIMode && len(c.CSRFSecret) == 0 {
		return errors.New("CSRF secret is required (or set APIMode=true for API-only apps)")
	}
	return nil
}

// Option is a functional option for configuring Aegis
type Option func(*Config)

// WithDB sets the database provider
func WithDB(db db.DBProvider) Option {
	return func(c *Config) {
		c.DB = db
	}
}

// WithPostgres is an alias for WithDB to match the user's desired API
func WithPostgres(db db.DBProvider) Option {
	return WithDB(db)
}

// WithRouter sets the router
func WithRouter(router server.Router) Option {
	return func(c *Config) {
		c.Router = router
	}
}

// WithJWTSecret sets the JWT signing secret
func WithJWTSecret(secret []byte) Option {
	return func(c *Config) {
		c.JWTSecret = secret
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

// WithIDGenerator sets a custom ID generation function
// Example: WithIDGenerator(func() string { return ulid.Make().String() })
func WithIDGenerator(generator func() string) Option {
	return func(c *Config) {
		c.IDGenerator = generator
	}
}

// WithAPIMode enables API-only mode (skips CSRF secret requirement)
// Use this when building APIs without web UI
func WithAPIMode(enabled bool) Option {
	return func(c *Config) {
		c.APIMode = enabled
	}
}
