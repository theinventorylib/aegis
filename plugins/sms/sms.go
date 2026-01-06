// Package sms provides phone-based OTP (One-Time Password) verification and authentication.
//
// This plugin enables:
//   - Phone number verification via SMS OTP codes
//   - Phone+password registration and login
//   - Multi-factor authentication (MFA) via SMS
//   - Password reset via phone verification
//
// OTP Flow:
//  1. User requests OTP via SendOTP endpoint (requires authentication)
//  2. Plugin generates 6-digit code (configurable length)
//  3. Code sent via configured SMS provider (Twilio, AWS SNS, etc.)
//  4. Code stored in database with expiry (default: 10 minutes)
//  5. User submits code via VerifyOTP endpoint
//  6. Plugin validates code and marks phone as verified
//
// Phone+Password Flow:
//  1. User registers with phone+password via /register endpoint
//  2. Password hashed with bcrypt and stored
//  3. User can login with phone+password via /login endpoint
//  4. Session created on successful authentication
//
// Route Structure:
//   - POST /sms/send     - Send SMS OTP code (protected)
//   - POST /sms/verify   - Verify SMS OTP code (public)
//   - POST /sms/login    - Login with phone+password (public)
//   - POST /sms/register - Register with phone+password (public)
//
// Provider Integration:
// Implement the Provider interface to use your SMS service:
//
//	type MyTwilioProvider struct { accountSID, authToken, from string }
//	func (p *MyTwilioProvider) SendOTP(to, code string) error {
//	  // Send SMS with OTP code via Twilio API
//	}
package sms

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin provides phone-based OTP verification and authentication.
//
// This plugin integrates with SMS service providers to send OTP codes and
// supports phone+password authentication as an alternative auth method.
//
// Features:
//   - Configurable OTP length and expiry
//   - Pluggable SMS providers (Twilio, AWS SNS, Vonage, etc.)
//   - International phone number validation (E.164 format)
//   - Password hashing with bcrypt
//   - Session management integration
type Plugin struct {
	// provider sends OTP codes via SMS (can be nil for testing)
	provider Provider
	// otpExpiry defines how long OTP codes remain valid (default: 10 minutes)
	otpExpiry time.Duration
	// otpLength specifies OTP code length (default: 6 digits)
	otpLength int
	// store handles phone-specific database operations
	store Store
	// logger for SMS sending events (nil-safe)
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

// Config holds SMS plugin configuration.
//
// Example:
//
//	cfg := &sms.Config{
//	  Provider:  myTwilioProvider,
//	  OTPExpiry: 15 * time.Minute,
//	  OTPLength: 8,
//	}
type Config struct {
	Provider  Provider      // SMS sending provider (required for production)
	OTPExpiry time.Duration // OTP expiry duration (default: 10 minutes)
	OTPLength int           // OTP code length (default: 6)
}

// New creates a new SMS plugin instance.
//
// Parameters:
//   - cfg: Plugin configuration (can be nil for defaults)
//   - store: Custom SMSStore implementation (can be nil, will use DefaultSMSStore)
//   - dialect: Database dialect (optional, defaults to PostgreSQL)
//
// Returns:
//   - *Plugin: Configured plugin ready for initialization
//
// Example:
//
//	plugin := sms.New(&sms.Config{
//	  Provider: myTwilioProvider,
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
	return "sms"
}

// Version returns the plugin version for compatibility tracking.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description for logging.
func (p *Plugin) Description() string {
	return "SMS verification plugin for phone number validation and MFA"
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
		p.store = NewDefaultSMSStore(a.DB())
	}

	return nil
}

// MountRoutes registers HTTP routes for the SMS plugin
func (p *Plugin) MountRoutes(router router.Router, prefix string) {
	handlers := NewHandlers(p)

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected route - sending OTP requires authentication to prevent spam/abuse
	router.POST(prefix+"/sms/send", requireAuth(http.HandlerFunc(handlers.SendOTPHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/send",
		Summary:     "Send SMS OTP",
		Description: "Send a one-time password via SMS to the authenticated user's phone number",
		Tags:        []string{"SMS"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Phone number to send OTP to",
			Required:    true,
			Schema:      "SendOTPRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "OTP sent successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"500": {Description: "Failed to send SMS", Schema: core.SchemaError},
		},
	})

	// Public routes
	router.POST(prefix+"/sms/verify", handlers.VerifyOTPHandler) // User proving phone ownership
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/verify",
		Summary:     "Verify SMS OTP",
		Description: "Verify a one-time password sent via SMS",
		Tags:        []string{"SMS"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Phone number and OTP code",
			Required:    true,
			Schema:      "VerifyOTPRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "OTP verified successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or incorrect OTP", Schema: core.SchemaError},
			"401": {Description: "OTP expired or not found", Schema: core.SchemaError},
		},
	})

	// Phone+password authentication (if core AuthService configured)
	router.POST(prefix+"/sms/login", handlers.LoginWithPhoneHandler) // Login endpoint
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/login",
		Summary:     "Login with phone and password",
		Description: "Authenticate using phone number and password",
		Tags:        []string{"SMS", "Authentication"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Phone number and password credentials",
			Required:    true,
			Schema:      "PhoneLoginRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Login successful, session created", Schema: core.SchemaSession},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Invalid credentials", Schema: core.SchemaError},
		},
	})

	router.POST(prefix+"/sms/register", handlers.RegisterWithPhoneHandler)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/register",
		Summary:     "Register with phone and password",
		Description: "Create a new account using phone number and password",
		Tags:        []string{"SMS", "Authentication"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Phone number and password credentials",
			Required:    true,
			Schema:      "RegisterWithPhoneRequest",
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Registration successful, session created", Schema: core.SchemaSession},
			"400": {Description: "Invalid request or phone number already exists", Schema: core.SchemaError},
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
	return []string{"sms_otp"}
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

// SendOTP generates and sends an OTP via SMS
func (p *Plugin) SendOTP(ctx context.Context, phoneNumber, purpose string) error {

	// Generate OTP code using shared utility
	code, err := core.GenerateOTPCode(p.otpLength)
	if err != nil {
		return fmt.Errorf("failed to generate OTP code: %w", err)
	}

	// Check if provider supports OTP operations
	if p.provider != nil {
		// Use provider's OTP sending
		if err := p.provider.SendOTP(phoneNumber, code); err != nil {
			return fmt.Errorf("failed to send OTP via provider: %w", err)
		}
	} else {
		// Fallback: log the OTP code (for testing purposes)
		if p.logger != nil {
			p.logger.Info("OTP code generated (no provider configured)",
				"phone", phoneNumber,
				"purpose", purpose,
				"code", code)
		}
	}

	// Fall back to core verification service
	if p.verificationService == nil {
		return fmt.Errorf("verification service not configured")
	}

	// Invalidate any existing OTPs for this phone number and purpose
	if phoneNumber != "" {
		if err := p.verificationService.InvalidateVerification(ctx, phoneNumber, purpose); err != nil {
			return fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification using core service with our custom OTP code as the token
	_, err = p.verificationService.CreateVerification(ctx, phoneNumber, purpose, p.otpExpiry, &code)
	if err != nil {
		return fmt.Errorf("failed to generate OTP code: %w", err)
	}

	return nil
}

// VerifyOTP verifies an OTP code
func (p *Plugin) VerifyOTP(ctx context.Context, phoneNumber, code string) (bool, error) {
	// Check if provider supports OTP operations
	if p.provider != nil {
		// Use provider's OTP verification
		return p.provider.VerifyOTP(phoneNumber, code)
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

// GetUserByPhone retrieves a user by phone number
func (p *Plugin) GetUserByPhone(ctx context.Context, phone string) (*auth.User, error) {
	if p.store == nil {
		return nil, fmt.Errorf("store not configured")
	}
	user, err := p.store.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	// Convert from our User type to auth.User
	return &auth.User{
		ID:        user.ID,
		Avatar:    user.Avatar,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Disabled:  user.Disabled,
	}, nil
}

// CreateUserWithPhoneAndPassword creates a new user with name, password account, and phone number.
func (p *Plugin) CreateUserWithPhoneAndPassword(ctx context.Context, name, phone, password string) (*User, error) {
	if p.store == nil {
		return nil, fmt.Errorf("core auth service not configured")
	}

	user := User{
		User: auth.User{
			ID:   core.GenerateID(),
			Name: name,
		},
		Phone:         &phone,
		PhoneVerified: false,
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
