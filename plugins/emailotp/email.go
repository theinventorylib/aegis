// Package emailotp provides email-based OTP (One-Time Password) verification and authentication.
//
// This plugin enables:
//   - Email address verification via OTP codes
//   - Email+password registration and login
//   - Multi-factor authentication (MFA) via email
//   - Password reset via email verification
//
// OTP Flow:
//  1. User requests OTP via SendOTP endpoint (requires authentication)
//  2. Plugin generates 6-digit code (configurable length)
//  3. Code sent via configured email provider (SMTP, SendGrid, etc.)
//  4. Code stored in database with expiry (default: 10 minutes)
//  5. User submits code via VerifyOTP endpoint
//  6. Plugin validates code and marks email as verified
//
// Email+Password Flow:
//  1. User registers with email+password via /register endpoint
//  2. Password hashed with bcrypt and stored
//  3. User can login with email+password via /login endpoint
//  4. Session created on successful authentication
//
// Route Structure:
//   - POST /emailotp/send     - Send OTP code (protected)
//   - POST /emailotp/verify   - Verify OTP code (public)
//   - POST /emailotp/login    - Login with email+password (public)
//   - POST /emailotp/register - Register with email+password (public)
//
// Provider Integration:
// Implement the Provider interface to use your email service:
//
//	type MySMTPProvider struct { host, user, pass string }
//	func (p *MySMTPProvider) SendOTP(to, code string) error {
//	  // Send email with OTP code
//	}
package emailotp

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// ValidateEmail validates an email address format using RFC 5322 regex.
//
// Parameters:
//   - email: Email address to validate
//
// Returns:
//   - error: If email is empty or has invalid format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	// Basic regex validation for obvious invalid formats
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// Plugin provides email-based OTP verification and authentication.
//
// This plugin integrates with email service providers to send OTP codes and
// supports email+password authentication as an alternative auth method.
//
// Features:
//   - Configurable OTP length and expiry
//   - Pluggable email providers (SMTP, SendGrid, SES, etc.)
//   - Email address validation
//   - Password hashing with bcrypt
//   - Session management integration
type Plugin struct {
	// provider sends OTP codes via email (can be nil for testing)
	provider Provider
	// otpExpiry defines how long OTP codes remain valid (default: 10 minutes)
	otpExpiry time.Duration
	// otpLength specifies OTP code length (default: 6 digits)
	otpLength int
	// store handles email-specific database operations
	store Store // Storage driver for user datasending events (nil-safe)
	// logger for OTP sending events (nil-safe)
	logger config.Logger
	// accountService manages password authentication
	accountService *core.AccountService
	// verificationService manages OTP code storage and validation
	verificationService *core.VerificationService
	// sessionService creates sessions after successful authentication
	sessionService *core.SessionService
	// dialect specifies database dialect (postgres, mysql)
	dialect plugins.Dialect
}

// Config holds Email OTP plugin configuration.
//
// Example:
//
//	cfg := &emailotp.Config{
//	  Provider:  myEmailProvider,
//	  OTPExpiry: 15 * time.Minute,
//	  OTPLength: 8,
//	}
type Config struct {
	Provider  Provider      // Email sending provider (required for production)
	OTPExpiry time.Duration // OTP expiry duration (default: 10 minutes)
	OTPLength int           // OTP code length (default: 6)
}

// New creates a new Email OTP plugin instance.
//
// Parameters:
//   - cfg: Plugin configuration (can be nil for defaults)
//   - store: Custom Store implementation (can be nil, will use DefaultEmailOTPStore)
//   - dialect: Database dialect (optional, defaults to PostgreSQL)
//
// Returns:
//   - *Plugin: Configured plugin ready for initialization
//
// Example:
//
//	plugin := emailotp.New(&emailotp.Config{
//	  Provider: mySMTPProvider,
//	  OTPExpiry: 15 * time.Minute,
//	  OTPLength: 8,
//	}, nil, plugins.DialectPostgres)
func New(cfg *Config, store Store, dialect ...plugins.Dialect) *Plugin {
	if cfg == nil {
		cfg = &Config{} // Initialize cfg to avoid nil pointer dereference
	}

	if cfg.OTPExpiry == 0 {
		cfg.OTPExpiry = 10 * time.Minute
	}
	if cfg.OTPLength == 0 {
		cfg.OTPLength = 6
	}

	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}

	return &Plugin{
		store:     store,
		provider:  cfg.Provider,
		otpExpiry: cfg.OTPExpiry,
		otpLength: cfg.OTPLength,
		dialect:   d,
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "emailotp"
}

// Version returns the plugin version for compatibility tracking.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description for logging.
func (p *Plugin) Description() string {
	return "Email OTP verification plugin for email validation and MFA"
}

// Init initializes the plugin.
func (p *Plugin) Init(_ context.Context, a plugins.Aegis) error {
	// Get services from Aegis interface - no type assertion needed
	authService := a.GetAuthService()
	p.accountService = authService.Account
	p.sessionService = authService.Session
	p.verificationService = authService.Verification
	p.logger = a.GetLogger()

	// Initialize store if not provided
	if p.store == nil {
		p.store = NewDefaultEmailOTPStore(a.DB())
	}

	return nil
}

// MountRoutes registers HTTP routes for the Email OTP plugin
func (p *Plugin) MountRoutes(router router.Router, prefix string) {
	handlers := NewHandlers(p)

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected route - sending OTP requires authentication to prevent spam/abuse
	router.POST(prefix+"/emailotp/send", requireAuth(http.HandlerFunc(handlers.SendOTPHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/emailotp/send",
		Summary:     "Send Email OTP",
		Description: "Send a one-time password via email to the authenticated user's email address",
		Tags:        []string{"EmailOTP"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Email address to send OTP to",
			Required:    true,
			Schema:      "SendOTPRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "OTP sent successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"500": {Description: "Failed to send email", Schema: core.SchemaError},
		},
	})

	// Public routes
	router.POST(prefix+"/emailotp/verify", handlers.VerifyOTPHandler) // User proving email ownership
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/emailotp/verify",
		Summary:     "Verify Email OTP",
		Description: "Verify a one-time password sent via email",
		Tags:        []string{"EmailOTP"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Email address and OTP code",
			Required:    true,
			Schema:      "VerifyOTPRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "OTP verified successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or incorrect OTP", Schema: core.SchemaError},
			"401": {Description: "OTP expired or not found", Schema: core.SchemaError},
		},
	})

	// Email+password authentication (if core AuthService configured)
	router.POST(prefix+"/emailotp/login", handlers.LoginWithEmailHandler) // Login endpoint
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/emailotp/login",
		Summary:     "Login with email and password",
		Description: "Authenticate using email address and password",
		Tags:        []string{"EmailOTP", "Authentication"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Email address and password credentials",
			Required:    true,
			Schema:      "EmailLoginRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Login successful, session created", Schema: core.SchemaSession},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Invalid credentials", Schema: core.SchemaError},
		},
	})

	router.POST(prefix+"/emailotp/register", handlers.RegisterWithEmailHandler)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/emailotp/register",
		Summary:     "Register with email and password",
		Description: "Create a new account using email address and password",
		Tags:        []string{"EmailOTP", "Authentication"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Email address and password credentials",
			Required:    true,
			Schema:      "RegisterWithEmailRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Registration successful, session created", Schema: core.SchemaSession},
			"400": {Description: "Invalid request or email already exists", Schema: core.SchemaError},
		},
	})
}

// Dependencies returns external package dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns core tables this plugin depends on
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user"}
}

// ProvidesAuthMethods returns authentication methods provided
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"email_otp"}
}

// GetMigrations returns the plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	migs, err := GetMigrations(p.dialect)
	if err != nil {
		return nil
	}
	return migs
}

// GetSchemas returns all schemas for all supported dialects
func (p *Plugin) GetSchemas() []plugins.Schema {
	dialects := []plugins.Dialect{plugins.DialectPostgres, plugins.DialectMySQL}
	schemas := make([]plugins.Schema, 0, len(dialects))

	for _, dialect := range dialects {
		schema, err := GetSchema(dialect)
		if err != nil {
			continue
		}
		schemas = append(schemas, *schema)
	}

	return schemas
}

// SendOTP generates and sends an OTP code via email.
//
// OTP Generation and Delivery:
//  1. Generate random N-digit code (configurable length)
//  2. Send code via email provider
//  3. Store code in verification service with expiry
//  4. Invalidate any previous OTPs for same email+purpose
//
// Parameters:
//   - ctx: Request context
//   - emailAddress: Recipient email address
//   - userID: User ID requesting OTP
//   - purpose: OTP purpose ("email_verification", "password_reset", "login_mfa")
//
// Returns:
//   - error: If OTP generation or sending fails
//
// Example:
//
//	err := plugin.SendOTP(ctx, "user@example.com", "email_verification")
func (p *Plugin) SendOTP(ctx context.Context, emailAddress, purpose string) error {

	// Generate OTP code using shared utility
	code, err := core.GenerateOTPCode(p.otpLength)
	if err != nil {
		return fmt.Errorf("failed to generate OTP code: %w", err)
	}

	// Check if provider supports OTP operations
	if p.provider != nil {
		// Use provider's OTP sending
		if err := p.provider.SendOTP(emailAddress, code); err != nil {
			return fmt.Errorf("failed to send OTP via provider: %w", err)
		}
	} else {
		// Fallback: log the OTP code (for testing purposes)
		if p.logger != nil {
			p.logger.Info("OTP code generated (no provider configured)",
				"email", emailAddress,
				"purpose", purpose,
				"code", code)
		}
	}

	// Fall back to core verification service
	if p.verificationService == nil {
		return fmt.Errorf("verification service not configured")
	}

	// Invalidate any existing OTPs for this email address and purpose
	if emailAddress != "" {
		if err := p.verificationService.InvalidateVerification(ctx, emailAddress, purpose); err != nil {
			return fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification using core service with our custom OTP code as the token
	_, err = p.verificationService.CreateVerification(ctx, emailAddress, purpose, p.otpExpiry, &code)
	if err != nil {
		return fmt.Errorf("failed to generate OTP code: %w", err)
	}

	return nil
}

// VerifyOTP verifies an OTP code for an email address.
//
// Verification Process:
//  1. Check if provider supports OTP verification (rare)
//  2. Fall back to core verification service (standard)
//  3. Validate code and check expiry
//  4. Return success/failure
//
// Parameters:
//   - ctx: Request context
//   - emailAddress: Email address to verify
//   - code: OTP code to verify (e.g., "123456")
//   - purpose: OTP purpose (must match SendOTP purpose)
//
// Returns:
//   - bool: true if OTP is valid and not expired
//   - error: If verification fails
//
// VerifyOTP verifies an OTP code for the given email address.
//
// Example:
//
//	valid, err := plugin.VerifyOTP(ctx, "user@example.com", "123456")
//	if valid {
//	  // Mark email as verified
//	}
func (p *Plugin) VerifyOTP(ctx context.Context, emailAddress, code string) (bool, error) {
	// Check if provider supports OTP operations
	if p.provider != nil {
		// Use provider's OTP verification
		return p.provider.VerifyOTP(emailAddress, code)
	}

	// Fall back to core verification service
	if p.verificationService == nil {
		return false, fmt.Errorf("verification service not configured")
	}

	// Validate the verification using the core service
	_, err := p.verificationService.ValidateVerification(ctx, code)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetUserByEmail retrieves a user by email address
func (p *Plugin) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	if p.store == nil {
		return nil, fmt.Errorf("store not configured")
	}
	user, err := p.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	// Convert from our User type to auth.User
	userEmail := ""
	if user.Email != nil {
		userEmail = *user.Email
	}
	return &auth.User{
		ID:        user.ID,
		Avatar:    user.Avatar,
		Name:      user.Name,
		Email:     userEmail,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Disabled:  user.Disabled,
	}, nil
}

// CreateUserWithEmailAndPassword creates a new user with email+password authentication.
//
// Registration Process:
//  1. Create user record with email (unverified)
//  2. Hash password with bcrypt
//  3. Create password account in auth.accounts table
//  4. Return user for session creation
//
// Parameters:
//   - ctx: Request context
//   - name: User display name
//   - email: User email address (becomes primary identifier)
//   - password: Plain text password (will be hashed)
//
// Returns:
//   - *User: Created user with email field
//   - error: If user creation or password hashing fails
//
// Example:
//
//	user, err := plugin.CreateUserWithEmailAndPassword(ctx, "John Doe", "john@example.com", "SecurePass123!")
func (p *Plugin) CreateUserWithEmailAndPassword(ctx context.Context, name, email, password string) (*User, error) {
	if p.store == nil {
		return nil, fmt.Errorf("core auth service not configured")
	}

	user := User{
		User: auth.User{
			ID:   core.GenerateID(),
			Name: name,
		},
		Email:         &email,
		EmailVerified: false,
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	u, err := p.store.CreateUser(ctx, user)
	if err != nil {
		return u, err
	}

	uid := u.GetID()

	hashedPassword, err := core.HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		return u, err
	}

	account := auth.Account{
		ID:           core.GenerateID(),
		UserID:       uid,
		Provider:     core.PasswordProvider,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := p.accountService.CreateAccount(ctx, account); err != nil {
		return u, err
	}

	return u, nil
}
