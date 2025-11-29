package email

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"

	"github.com/theinventorylib/aegis/server"
)

// Plugin provides email verification functionality
type Plugin struct {
	db             *DB
	provider       Provider
	otpExpiry      time.Duration
	authService    *core.AuthService // Use core AuthService for password operations
	sessionService *core.SessionService
}

// Config for email plugin
type Config struct {
	DB             db.Provider
	Provider       Provider      // Email sending provider
	OTPExpiry      time.Duration // OTP expiry duration
	SessionService *core.SessionService
}

// New creates a new email plugin
func New(cfg *Config) *Plugin {
	if cfg.OTPExpiry == 0 {
		cfg.OTPExpiry = 10 * time.Minute
	}

	return &Plugin{
		db:        NewDB(cfg.DB),
		provider:  cfg.Provider,
		otpExpiry: cfg.OTPExpiry,
		// Password functionality now provided by core.AuthService at runtime
		sessionService: cfg.SessionService,
	}
}

// Name returns the plugin identifier
func (p *Plugin) Name() string {
	return "email"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description
func (p *Plugin) Description() string {
	return "Email verification plugin for email validation, magic links, and password reset"
}

// Init initializes the email plugin.
func (p *Plugin) Init(_ context.Context, a plugins.Aegis) error {
	// Get session service from Aegis instance
	if app, ok := a.(*aegis.Aegis); ok {
		p.sessionService = app.GetSessionService()
		p.authService = app.GetAuthService()
	}
	return nil
}

// MountRoutes registers HTTP routes for the email plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// Email+password authentication (uses core AuthService)
	if p.authService != nil {
		router.POST(prefix+"/email/login", handlers.LoginWithEmailHandler)
		router.RegisterRouteMetadata(models.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/email/login",
			Summary:     "Login with email and password",
			Description: "Authenticate using email address and password",
			Tags:        []string{"Email", "Authentication"},
			Protected:   false,
			RequestBody: &models.RequestBodyMeta{
				Description: "Email and password credentials",
				Required:    true,
				Schema:      SchemaLoginWithEmailRequest,
			},
			Responses: map[string]*models.ResponseMeta{
				"200": {Description: "Login successful, session created", Schema: models.SchemaSession},
				"400": {Description: "Invalid request", Schema: models.SchemaError},
				"401": {Description: "Invalid credentials", Schema: models.SchemaError},
			},
		})

		router.POST(prefix+"/email/register", handlers.RegisterWithEmailHandler)
		router.RegisterRouteMetadata(models.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/email/register",
			Summary:     "Register with email and password",
			Description: "Create a new account using email address and password",
			Tags:        []string{"Email", "Authentication"},
			Protected:   false,
			RequestBody: &models.RequestBodyMeta{
				Description: "Email and password credentials",
				Required:    true,
				Schema:      "RegisterWithEmailRequest",
			},
			Responses: map[string]*models.ResponseMeta{
				"201": {Description: "Registration successful, session created", Schema: models.SchemaSession},
				"400": {Description: "Invalid request or email already exists", Schema: models.SchemaError},
			},
		})
	}

	// Email verification routes would go here
	// e.g., POST /email/send-verification, POST /email/verify
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
	return []string{"email_otp", "magic_link"}
}

// SendPasswordResetEmail sends a password reset email with a magic link
// SendPasswordResetEmail sends a password reset email.
func (p *Plugin) SendPasswordResetEmail(_ context.Context, email, token, resetURL string) error {
	if p.provider == nil {
		return fmt.Errorf("email provider not configured")
	}

	subject := "Password Reset Request"
	body := fmt.Sprintf(`
		<h2>Password Reset Request</h2>
		<p>You requested to reset your password. Click the link below to reset it:</p>
		<p><a href="%s?token=%s">Reset Password</a></p>
		<p>This link will expire in 1 hour.</p>
		<p>If you didn't request this, please ignore this email.</p>
	`, resetURL, token)

	return p.provider.SendEmail(email, subject, body)
}

// SendPasswordResetOTP sends a password reset OTP via email
func (p *Plugin) SendPasswordResetOTP(ctx context.Context, email, userID string) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("database not configured")
	}

	// Generate OTP code using shared utility
	code, err := core.GenerateOTPCode(6)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP code: %w", err)
	}

	// Invalidate any existing OTPs for password reset
	if userID != "" {
		if err := p.db.InvalidateVerifications(ctx, userID, "password_reset"); err != nil {
			return "", fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification record
	id := core.GenerateID()
	expiresAt := time.Now().Add(p.otpExpiry)
	createdAt := time.Now()

	verification := &Verification{
		ID:        id,
		Email:     email,
		Code:      code,
		Purpose:   "password_reset",
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}

	if err := p.db.CreateVerification(ctx, verification); err != nil {
		return "", fmt.Errorf("failed to create verification: %w", err)
	}

	// Send OTP via email
	if p.provider != nil {
		subject := "Password Reset Code"
		body := fmt.Sprintf(`
			<h2>Password Reset Code</h2>
			<p>Your password reset code is:</p>
			<h1 style="font-size: 32px; letter-spacing: 5px;">%s</h1>
			<p>This code will expire in %d minutes.</p>
			<p>If you didn't request this, please ignore this email.</p>
		`, code, int(p.otpExpiry.Minutes()))

		if err := p.provider.SendEmail(email, subject, body); err != nil {
			return "", fmt.Errorf("failed to send email: %w", err)
		}
	}

	return code, nil
}

// SendVerificationEmail sends an email verification email
// SendVerificationEmail sends an email verification email.
func (p *Plugin) SendVerificationEmail(_ context.Context, email, token, verifyURL string) error {
	if p.provider == nil {
		return fmt.Errorf("email provider not configured")
	}

	subject := "Verify Your Email"
	body := fmt.Sprintf(`
		<h2>Email Verification</h2>
		<p>Please verify your email address by clicking the link below:</p>
		<p><a href="%s?token=%s">Verify Email</a></p>
		<p>If you didn't create an account, please ignore this email.</p>
	`, verifyURL, token)

	return p.provider.SendEmail(email, subject, body)
}

// VerifyOTP verifies an OTP code for email-based verification
func (p *Plugin) VerifyOTP(ctx context.Context, email, code, purpose string) (bool, error) {
	if p.db == nil {
		return false, fmt.Errorf("database not configured")
	}
	return p.db.VerifyOTP(ctx, email, code, purpose)
}

// VerifyToken verifies a token for link-based verification
func (p *Plugin) VerifyToken(ctx context.Context, token string) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("database not configured")
	}
	return p.db.VerifyToken(ctx, token)
}

// GetUserByEmail retrieves a user by email
func (p *Plugin) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return p.db.GetUserByEmail(ctx, email)
}

// CreateUserWithEmail creates a user with email
func (p *Plugin) CreateUserWithEmail(ctx context.Context, email string) (*models.User, error) {
	// 1. Create core user
	dbProvider := p.sessionService.GetDB()
	user, err := dbProvider.CreateUser(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Update email
	if err := p.db.UpdateUserEmail(ctx, user.ID, email, false); err != nil {
		return nil, fmt.Errorf("failed to set email: %w", err)
	}

	return user, nil
}

// CreateUserWithEmailAndPassword creates a new user and a password account, then sets the email.
func (p *Plugin) CreateUserWithEmailAndPassword(ctx context.Context, email, password string) (*models.User, error) {
	if p.authService == nil {
		return nil, fmt.Errorf("core auth service not configured")
	}

	// Create user with password atomically at core level (best-effort cleanup handled by core)
	user, err := p.authService.CreateUserWithPassword(ctx, password)
	if err != nil {
		return nil, err
	}

	// Set email for the created user
	if err := p.db.UpdateUserEmail(ctx, user.ID, email, false); err != nil {
		// If email set fails, attempt to cleanup the user
		_ = p.sessionService.GetDB().DeleteUser(ctx, user.ID)
		return nil, fmt.Errorf("failed to set email: %w", err)
	}

	return user, nil
}
