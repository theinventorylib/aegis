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

// Config holds the configuration for Aegis
type Config struct {
	// Core dependencies
	DB     db.Provider
	Router server.Router

	// Internal error tracking (not exported)
	dbError error

	// Security
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
	if c.dbError != nil {
		return c.dbError
	}
	if c.DB == nil {
		return errors.New("database provider is required")
	}
	if c.Router == nil {
		return errors.New("router is required")
	}
	// CSRF secret only required for web apps (not API-only mode)
	if !c.APIMode && len(c.CSRFSecret) == 0 {
		return errors.New("CSRF secret is required (or set APIMode=true for API-only apps)")
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

// WithIDGenerator sets a custom ID generation function
// Example: WithIDGenerator(func() string { return ulid.Make().String() })
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
