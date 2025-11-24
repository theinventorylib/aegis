package core

import "time"

// PasswordHasherConfig defines configuration for password hashing
type PasswordHasherConfig struct {
	Argon2Time      uint32
	Argon2Memory    uint32
	Argon2Threads   uint8
	Argon2KeyLength uint32
}

// SessionConfig defines configuration for session management
type SessionConfig struct {
	SessionExpiry  time.Duration
	RefreshExpiry  time.Duration
	CookieSettings CookieSettings
	Redis          *RedisConfig
}

// RedisConfig defines Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// CookieSettings defines cookie configuration
type CookieSettings struct {
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite string // "Lax", "Strict", or "None"
}

// DefaultPasswordHasherConfig returns default password hashing configuration.
func DefaultPasswordHasherConfig() *PasswordHasherConfig {
	return &PasswordHasherConfig{
		Argon2Time:      1,
		Argon2Memory:    64 * 1024, // 64 MB
		Argon2Threads:   4,
		Argon2KeyLength: 32,
	}
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionExpiry: 24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
		CookieSettings: CookieSettings{
			Secure:   true,
			HTTPOnly: true,
			SameSite: "Lax",
		},
	}
}
