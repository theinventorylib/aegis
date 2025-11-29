package sms

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"

	"github.com/theinventorylib/aegis/server"
)

// Plugin represents the SMS plugin for Aegis
type Plugin struct {
	db             *DB
	provider       Provider
	otpExpiry      time.Duration
	otpLength      int
	authService    *core.AuthService // For phone+password authentication
	sessionService *core.SessionService
}

// Config holds SMS plugin configuration
type Config struct {
	DB             db.Provider
	Provider       Provider      // SMS sending provider
	OTPExpiry      time.Duration // OTP expiry duration
	OTPLength      int
	SessionService *core.SessionService
	// Password auth is provided by core AuthService at runtime
}

// New creates a new SMS plugin instance
func New(cfg *Config) *Plugin {
	if cfg == nil {
		cfg = &Config{} // Initialize cfg to avoid nil pointer dereference
	}

	if cfg.OTPExpiry == 0 {
		cfg.OTPExpiry = 10 * time.Minute
	}
	if cfg.OTPLength == 0 {
		cfg.OTPLength = 6
	}

	return &Plugin{
		db:             NewDB(cfg.DB),
		provider:       cfg.Provider,
		otpExpiry:      cfg.OTPExpiry,
		otpLength:      cfg.OTPLength,
		sessionService: cfg.SessionService,
	}
}

// Name returns the plugin identifier
func (p *Plugin) Name() string {
	return "sms"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description
func (p *Plugin) Description() string {
	return "SMS verification plugin for phone number validation and MFA"
}

// Init initializes the plugin.
func (p *Plugin) Init(_ context.Context, a plugins.Aegis) error {
	// Get session service from Aegis instance
	if app, ok := a.(*aegis.Aegis); ok {
		p.sessionService = app.GetSessionService()
		p.authService = app.GetAuthService()
	}
	return nil
}

// MountRoutes registers HTTP routes for the SMS plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected route - sending OTP requires authentication to prevent spam/abuse
	router.POST(prefix+"/sms/send", requireAuth(http.HandlerFunc(handlers.SendOTPHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/send",
		Summary:     "Send SMS OTP",
		Description: "Send a one-time password via SMS to the authenticated user's phone number",
		Tags:        []string{"SMS"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Phone number to send OTP to",
			Required:    true,
			Schema:      "SendOTPRequest",
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "OTP sent successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"500": {Description: "Failed to send SMS", Schema: models.SchemaError},
		},
	})

	// Public routes
	router.POST(prefix+"/sms/verify", handlers.VerifyOTPHandler) // User proving phone ownership
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/sms/verify",
		Summary:     "Verify SMS OTP",
		Description: "Verify a one-time password sent via SMS",
		Tags:        []string{"SMS"},
		Protected:   false,
		RequestBody: &models.RequestBodyMeta{
			Description: "Phone number and OTP code",
			Required:    true,
			Schema:      "VerifyOTPRequest",
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "OTP verified successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or incorrect OTP", Schema: models.SchemaError},
			"401": {Description: "OTP expired or not found", Schema: models.SchemaError},
		},
	})

	// Phone+password authentication (if core AuthService configured)
	if p.authService != nil {
		router.POST(prefix+"/sms/login", handlers.LoginWithPhoneHandler) // Login endpoint
		router.RegisterRouteMetadata(models.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/sms/login",
			Summary:     "Login with phone and password",
			Description: "Authenticate using phone number and password",
			Tags:        []string{"SMS", "Authentication"},
			Protected:   false,
			RequestBody: &models.RequestBodyMeta{
				Description: "Phone number and password credentials",
				Required:    true,
				Schema:      "PhoneLoginRequest",
			},
			Responses: map[string]*models.ResponseMeta{
				"200": {Description: "Login successful, session created", Schema: models.SchemaSession},
				"400": {Description: "Invalid request", Schema: models.SchemaError},
				"401": {Description: "Invalid credentials", Schema: models.SchemaError},
			},
		})

		router.POST(prefix+"/sms/register", handlers.RegisterWithPhoneHandler)
		router.RegisterRouteMetadata(models.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/sms/register",
			Summary:     "Register with phone and password",
			Description: "Create a new account using phone number and password",
			Tags:        []string{"SMS", "Authentication"},
			Protected:   false,
			RequestBody: &models.RequestBodyMeta{
				Description: "Phone number and password credentials",
				Required:    true,
				Schema:      "RegisterWithPhoneRequest",
			},
			Responses: map[string]*models.ResponseMeta{
				"201": {Description: "Registration successful, session created", Schema: models.SchemaSession},
				"400": {Description: "Invalid request or phone number already exists", Schema: models.SchemaError},
			},
		})
	}
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

// SendOTP generates and sends an OTP via SMS
func (p *Plugin) SendOTP(ctx context.Context, phoneNumber, userID, purpose string) error {
	if p.db == nil {
		return fmt.Errorf("database not configured")
	}

	// Generate OTP code using shared utility
	code, err := core.GenerateOTPCode(p.otpLength)
	if err != nil {
		return fmt.Errorf("failed to generate OTP code: %w", err)
	}

	// Invalidate any existing OTPs for this user and purpose
	if userID != "" {
		if err := p.db.InvalidateVerifications(ctx, userID, purpose); err != nil {
			return fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification record
	id := core.GenerateID()
	expiresAt := time.Now().Add(p.otpExpiry)
	createdAt := time.Now()

	verification := &Verification{
		ID:          id,
		PhoneNumber: phoneNumber,
		Code:        code,
		Purpose:     purpose,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	}

	if err := p.db.CreateVerification(ctx, verification); err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}

	// Send OTP via SMS provider
	if p.provider != nil {
		if err := p.provider.SendSMS(phoneNumber, fmt.Sprintf("Your verification code is: %s", code)); err != nil {
			return fmt.Errorf("failed to send SMS: %w", err)
		}
	}

	return nil
}

// VerifyOTP verifies an OTP code
func (p *Plugin) VerifyOTP(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	if p.db == nil {
		return false, fmt.Errorf("database not configured")
	}
	return p.db.VerifyOTP(ctx, phoneNumber, code, purpose)
}

// GetUserByPhone retrieves a user by phone number
func (p *Plugin) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	return p.db.GetUserByPhone(ctx, phone)
}

// CreateUserWithPhoneAndPassword creates a new user and password account, then sets the phone number.
func (p *Plugin) CreateUserWithPhoneAndPassword(ctx context.Context, phone, password string) (*models.User, error) {
	if p.authService == nil {
		return nil, fmt.Errorf("core auth service not configured")
	}

	user, err := p.authService.CreateUserWithPassword(ctx, password)
	if err != nil {
		return nil, err
	}

	if err := p.db.UpdateUserPhone(ctx, user.ID, phone, false); err != nil {
		_ = p.sessionService.GetDB().DeleteUser(ctx, user.ID)
		return nil, fmt.Errorf("failed to set phone number: %w", err)
	}

	return user, nil
}
