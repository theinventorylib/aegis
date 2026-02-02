package core

import "time"

// AuthConfig defines core authentication system configuration.
// This is the primary configuration struct passed to NewAuthService.
type AuthConfig struct {
	// EnableEmailPassword controls whether email/password authentication is available.
	// When false, users cannot signup or login with credentials (OAuth/SSO only).
	EnableEmailPassword bool

	// PasswordPolicy defines password strength requirements for signup/change.
	// If nil, uses DefaultPasswordPolicyConfig (8+ chars, mixed case, digit required).
	PasswordPolicy *PasswordPolicyConfig

	// InvalidateSessionsOnPasswordChange, when true, logs users out from all
	// devices when their password changes. This is a security best practice that
	// prevents attackers from maintaining access after a password is reset.
	// Recommended: true
	InvalidateSessionsOnPasswordChange bool

	// UserFields controls which plugin extension fields are included in user
	// API responses. If nil, all extension fields are included.
	// Use this to limit what data is exposed in user objects.
	UserFields *UserFieldsConfig
}

// UserFieldsConfig defines which extension fields plugins should add to EnrichedUser.
// This allows users to configure what additional data appears in user API responses.
//
// Example configuration:
//
//	UserFields: &core.UserFieldsConfig{
//	    Fields: []string{"role", "permissions", "organizations", "verified"},
//	}
//
// This produces JSON responses like:
//
//	{
//	    "id": "user_123",
//	    "email": "user@example.com",
//	    "role": "admin",
//	    "permissions": ["read", "write"],
//	    "organizations": ["org1", "org2"],
//	    "verified": true
//	}
type UserFieldsConfig struct {
	// Fields is the list of extension field names to include in user responses.
	// Plugins will only add fields that are in this list (if configured).
	// If nil or empty, all plugin fields are included (default behavior).
	Fields []string
}

// PasswordHasherConfig defines Argon2id parameters for password hashing.
//
// Argon2id is memory-hard and resistant to GPU/ASIC attacks. Higher values
// increase security at the cost of CPU/memory usage during login.
//
// OWASP 2024 recommendations:
//   - Time: 1-3 iterations
//   - Memory: 64MB-256MB (64*1024 - 256*1024 KiB)
//   - Threads: Match CPU cores (typically 4)
//   - KeyLength: 32 bytes (256 bits)
//
// For high-security applications, increase Memory to 256MB+ and Time to 3+.
// For resource-constrained environments, keep defaults but monitor load.
type PasswordHasherConfig struct {
	// Argon2Time is the number of iterations (time cost)
	Argon2Time uint32

	// Argon2Memory is the memory cost in KiB (e.g., 65536 = 64MB)
	Argon2Memory uint32

	// Argon2Threads is the degree of parallelism (typically 4)
	Argon2Threads uint8

	// Argon2KeyLength is the derived key length in bytes (typically 32)
	Argon2KeyLength uint32
}

// PasswordPolicyConfig defines password validation rules.
//
// Modern password policy recommendations (NIST/OWASP 2024):
//   - Require minimum length (8+ characters)
//   - Optionally require character diversity (mixed case, digits, symbols)
//   - Don't require forced expiration or rotation
//   - Check against breached password databases
//
// Note: Overly strict policies can lead to weaker passwords (users write them down,
// use predictable patterns, etc.). Balance security with usability.
type PasswordPolicyConfig struct {
	// MinLength is minimum password length (default: 8, NIST minimum)
	MinLength int

	// RequireUpper requires at least one uppercase letter (default: true)
	RequireUpper bool

	// RequireLower requires at least one lowercase letter (default: true)
	RequireLower bool

	// RequireDigit requires at least one numeric digit (default: true)
	RequireDigit bool

	// RequireSpecial requires at least one special character (default: false)
	// Special chars: !@#$%^&*()_+-=[]{}|;:,.<>?
	RequireSpecial bool

	// MaxLength caps password length to prevent DoS (default: 128, 0 = unlimited)
	// Very long passwords can cause excessive CPU usage during hashing.
	MaxLength int
}

// SessionConfig defines session and cookie management settings.
type SessionConfig struct {
	// SessionExpiry is how long session tokens remain valid.
	// After this, users must re-login or use a refresh token.
	// Typical: 24 hours for session, 7 days for refresh
	SessionExpiry time.Duration

	// RefreshExpiry is how long refresh tokens remain valid.
	// Enables "remember me" functionality by allowing new sessions
	// to be created without re-entering credentials.
	RefreshExpiry time.Duration

	// CookieSettings configures HTTP session cookies
	CookieSettings CookieSettings

	// Redis connection for session caching (optional).
	// If nil, sessions are always loaded from database (slower but simpler).
	Redis *RedisConfig
}

// RedisConfig defines Redis connection parameters.
// Redis is used for session caching and distributed rate limiting.
type RedisConfig struct {
	// Host is the Redis server hostname or IP (e.g., "localhost", "redis.example.com")
	Host string

	// Port is the Redis server port (default: 6379)
	Port int

	// Password for Redis authentication (empty if no auth required)
	Password string

	// DB is the Redis database number (0-15, default: 0)
	DB int
}

// CookieSettings defines HTTP cookie configuration for session tokens.
//
// Security best practices:
//   - Always set HTTPOnly=true (prevents JavaScript access)
//   - Set Secure=true in production (HTTPS only)
//   - Use SameSite="Lax" or "Strict" (CSRF protection)
//   - Set Domain to your domain for subdomain sharing
type CookieSettings struct {
	// Name is the cookie name (default: "aegis_session")
	Name string

	// Domain controls which domains can access the cookie.
	// Empty: Current domain only
	// ".example.com": All subdomains of example.com
	Domain string

	// Secure requires HTTPS for cookie transmission.
	// Always true in production. Can be false for local development.
	Secure bool

	// HTTPOnly prevents JavaScript from accessing the cookie.
	// Always true for session cookies (XSS protection).
	HTTPOnly bool

	// SameSite controls cross-site cookie behavior (CSRF protection).
	// Options:
	//   - "Strict": Cookie never sent in cross-site requests
	//   - "Lax": Cookie sent on top-level navigation (default, recommended)
	//   - "None": Cookie always sent (requires Secure=true)
	SameSite string
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		EnableEmailPassword:                true, // Enabled by default
		PasswordPolicy:                     DefaultPasswordPolicyConfig(),
		InvalidateSessionsOnPasswordChange: true, // Security best practice
		UserFields:                         nil,  // Include all extension fields by default
	}
}

// DefaultUserFieldsConfig returns default user fields configuration.
// By default, all extension fields are included in responses.
func DefaultUserFieldsConfig() *UserFieldsConfig {
	return &UserFieldsConfig{
		Fields: nil, // All fields included
	}
}

// DefaultPasswordHasherConfig returns default password hashing configuration.
func DefaultPasswordHasherConfig() *PasswordHasherConfig {
	return &PasswordHasherConfig{
		Argon2Time:      DefaultArgon2Time,
		Argon2Memory:    DefaultArgon2Memory,
		Argon2Threads:   DefaultArgon2Threads,
		Argon2KeyLength: DefaultArgon2KeyLength,
	}
}

// DefaultPasswordPolicyConfig returns default password policy configuration
func DefaultPasswordPolicyConfig() *PasswordPolicyConfig {
	return &PasswordPolicyConfig{
		MinLength:      DefaultPasswordMinLength,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: false,
		MaxLength:      DefaultPasswordMaxLength,
	}
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionExpiry: DefaultSessionExpiry,
		RefreshExpiry: DefaultRefreshExpiry,
		CookieSettings: CookieSettings{
			Name:     DefaultCookieName,
			Secure:   true,
			HTTPOnly: true,
			SameSite: DefaultCookieSameSite,
		},
	}
}
