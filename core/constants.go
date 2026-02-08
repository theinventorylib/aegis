package core

import "time"

// This file defines default configuration values used throughout the Aegis framework.
// All constants are exported so applications can reference them when creating custom
// configurations. Values are chosen to balance security and usability based on OWASP
// and industry best practices.

// Default rate limiting constants control API request throttling.
// Separate limits are defined for general endpoints vs. authentication endpoints.
const (
	// DefaultRateLimitRequests is the default number of requests allowed per window
	// for general API endpoints (100 requests/minute is suitable for most applications)
	DefaultRateLimitRequests = 100

	// DefaultRateLimitWindow is the default time window for rate limiting (1 minute)
	DefaultRateLimitWindow = time.Minute

	// DefaultRateLimitKeyPrefix is the default Redis key prefix for rate limit counters
	DefaultRateLimitKeyPrefix = "aegis:ratelimit:"

	// AuthRateLimitRequests is a stricter limit for authentication endpoints
	// (10 requests/minute prevents brute force while allowing legitimate retries)
	AuthRateLimitRequests = 10

	// AuthRateLimitKeyPrefix is the Redis key prefix for auth-specific rate limits
	AuthRateLimitKeyPrefix = "aegis:ratelimit:auth:"
)

// Default login attempt tracking constants prevent brute force attacks.
// After exceeding max attempts, accounts are temporarily locked.
const (
	// DefaultMaxLoginAttempts is the maximum number of failed login attempts allowed
	// before triggering account lockout (5 attempts is OWASP-recommended)
	DefaultMaxLoginAttempts = 5

	// DefaultLoginLockoutDuration is how long to lock out after max attempts
	// (15 minutes balances security with user convenience)
	DefaultLoginLockoutDuration = 15 * time.Minute

	// DefaultLoginAttemptWindow is the time window for counting attempts
	// (attempts older than this are not counted toward the limit)
	DefaultLoginAttemptWindow = 15 * time.Minute
)

// Default password policy constants enforce minimum security requirements.
const (
	// DefaultPasswordMinLength is the minimum password length (8 characters)
	// NIST recommends 8+ characters for user-chosen passwords
	DefaultPasswordMinLength = 8

	// DefaultPasswordMaxLength is the maximum password length (128 characters)
	// Prevents DoS attacks via extremely long password hashing.
	// 0 would mean no limit.
	DefaultPasswordMaxLength = 128
)

// Default session configuration constants control session lifetimes.
const (
	// DefaultSessionExpiry is the default session expiration time (24 hours)
	// After this, users must re-authenticate or use refresh token
	DefaultSessionExpiry = 24 * time.Hour

	// DefaultRefreshExpiry is the default refresh token expiration time (7 days)
	// Allows "remember me" functionality while limiting token lifetime
	DefaultRefreshExpiry = 7 * 24 * time.Hour
)

// Default password hashing constants use Argon2id parameters.
// These values are based on OWASP recommendations for 2024 and balance
// security (resistance to attacks) with performance (server load).
const (
	// DefaultArgon2Time is the number of iterations (time cost)
	// Higher = slower hashing = more brute force resistant
	DefaultArgon2Time = 1

	// DefaultArgon2Memory is the memory cost in KiB (64 MB)
	// Higher memory makes GPU attacks less effective
	DefaultArgon2Memory = 64 * 1024

	// DefaultArgon2Threads is the degree of parallelism
	// Should match typical server CPU core count
	DefaultArgon2Threads = 4

	// DefaultArgon2KeyLength is the derived key length in bytes (256 bits)
	DefaultArgon2KeyLength = 32
)

// Token generation constants define entropy for cryptographic tokens.
const (
	// TokenLength is the length of generated tokens in bytes (32 bytes = 256 bits)
	// Provides sufficient entropy to prevent guessing attacks
	TokenLength = 32

	// SaltLength is the length of password salt in bytes (16 bytes = 128 bits)
	// Ensures unique hash outputs even for identical passwords
	SaltLength = 16
)

// Redis key prefixes prevent key collisions when using shared Redis instances.
// All Aegis keys are prefixed with "aegis:" for easy identification.
const (
	// RedisSessionPrefix is the Redis key prefix for session storage
	RedisSessionPrefix = "aegis:session:"

	// RedisRefreshTokenPrefix is the Redis key prefix for refresh tokens
	RedisRefreshTokenPrefix = "aegis:refresh:"

	// RedisUserSessionsPrefix is the Redis key prefix for user session sets
	// (used to track all sessions for a user for "logout all devices")
	RedisUserSessionsPrefix = "aegis:user_sessions:"

	// RedisLoginAttemptsPrefix is the Redis key prefix for login attempt counters
	RedisLoginAttemptsPrefix = "aegis:login_attempts:"
)

// Default cookie settings for session management.
const (
	// DefaultCookieName is the default session cookie name
	DefaultCookieName = "aegis_session"
	// DefaultCookiePath is the default cookie path
	DefaultCookiePath = "/"
	// DefaultCookieSameSite is the default SameSite attribute
	DefaultCookieSameSite = "Lax"
	// DefaultCookieHTTPOnly is the default HttpOnly attribute
	DefaultCookieHTTPOnly = true
	// DefaultCookieSecure is the default Secure attribute
	// Ensures cookies are only sent over HTTPS in production
	DefaultCookieSecure = true
)

// Provider constants
const (
	// PasswordProvider is the provider name for password authentication
	PasswordProvider = "password"
)

// Character range constants for password validation
const (
	// UppercaseStart is the start of uppercase ASCII range
	UppercaseStart = 'A'
	// UppercaseEnd is the end of uppercase ASCII range
	UppercaseEnd = 'Z'
	// LowercaseStart is the start of lowercase ASCII range
	LowercaseStart = 'a'
	// LowercaseEnd is the end of lowercase ASCII range
	LowercaseEnd = 'z'
	// DigitStart is the start of digit ASCII range
	DigitStart = '0'
	// DigitEnd is the end of digit ASCII range
	DigitEnd = '9'
	// SpecialRange1Start is the start of first special character range
	SpecialRange1Start = '!'
	// SpecialRange1End is the end of first special character range
	SpecialRange1End = '/'
	// SpecialRange2Start is the start of second special character range
	SpecialRange2Start = ':'
	// SpecialRange2End is the end of second special character range
	SpecialRange2End = '@'
	// SpecialRange3Start is the start of third special character range
	SpecialRange3Start = '['
	// SpecialRange3End is the end of third special character range
	SpecialRange3End = '`'
	// SpecialRange4Start is the start of fourth special character range
	SpecialRange4Start = '{'
	// SpecialRange4End is the end of fourth special character range
	SpecialRange4End = '~'
)

// EmailRegexPattern is the regex pattern for email validation (RFC 5322 simplified)
const EmailRegexPattern = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
