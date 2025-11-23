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
	"github.com/theinventorylib/aegis/plugins/password"
	"github.com/theinventorylib/aegis/server"
)

// Plugin provides email verification functionality
type Plugin struct {
	db             db.DBProvider
	provider       Provider
	otpExpiry      time.Duration
	passwordPlugin *password.Plugin // For email+password authentication
	sessionService *core.SessionService
}

// Config for email plugin
type Config struct {
	DB             db.DBProvider
	Provider       Provider         // Email sending provider
	OTPExpiry      time.Duration    // OTP expiry duration
	PasswordPlugin *password.Plugin // Optional: for email+password auth
	SessionService *core.SessionService
}

// New creates a new email plugin
func New(cfg *Config) *Plugin {
	if cfg.OTPExpiry == 0 {
		cfg.OTPExpiry = 10 * time.Minute
	}

	return &Plugin{
		db:             cfg.DB,
		provider:       cfg.Provider,
		otpExpiry:      cfg.OTPExpiry,
		passwordPlugin: cfg.PasswordPlugin,
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

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, a plugins.Aegis) error {
	// Get session service from Aegis instance
	if app, ok := a.(*aegis.Aegis); ok {
		p.sessionService = app.GetSessionService()
	}
	return nil
}

// MountRoutes registers HTTP routes for the email plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// Email+password authentication (if password plugin configured)
	if p.passwordPlugin != nil {
		router.POST(prefix+"/email/login", handlers.LoginWithEmailHandler)
	}

	// Email verification routes would go here
	// e.g., POST /email/send-verification, POST /email/verify
}

// Dependencies returns external package dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	if p.passwordPlugin != nil {
		return []plugins.Dependency{
			{
				Package: "github.com/theinventorylib/aegis/plugins/password",
				Version: "latest",
				Purpose: "Password authentication for email+password login",
			},
		}
	}
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
func (p *Plugin) SendPasswordResetEmail(ctx context.Context, email, token, resetURL string) error {
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
		_, err := p.db.Exec(ctx, `
			DELETE FROM auth.verification
			WHERE identifier = ? AND type = ?
		`, userID, "password_reset")
		if err != nil {
			return "", fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification record
	id := core.GenerateID()
	expiresAt := time.Now().Add(p.otpExpiry)
	createdAt := time.Now()

	_, err = p.db.Exec(ctx, `
		INSERT INTO auth.verification (id, identifier, token, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, email, code, "password_reset", expiresAt, createdAt)

	if err != nil {
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
func (p *Plugin) SendVerificationEmail(ctx context.Context, email, token, verifyURL string) error {
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

	var id string
	var token string
	var expiresAt time.Time

	err := p.db.QueryRow(ctx, `
		SELECT id, token, expires_at
		FROM auth.verification
		WHERE identifier = ? AND type = ? AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, email, purpose).Scan(&id, &token, &expiresAt)

	if err != nil {
		return false, fmt.Errorf("OTP not found or expired")
	}

	// Verify the code matches
	if token != code {
		return false, nil
	}

	// Delete the used verification token
	_, err = p.db.Exec(ctx, `
		DELETE FROM auth.verification
		WHERE id = ?
	`, id)

	if err != nil {
		return false, fmt.Errorf("failed to delete used OTP: %w", err)
	}

	return true, nil
}

// VerifyToken verifies a token for link-based verification
func (p *Plugin) VerifyToken(ctx context.Context, token string) (string, error) {
	if p.db == nil {
		return "", fmt.Errorf("database not configured")
	}

	var id string
	var identifier string
	var expiresAt time.Time

	err := p.db.QueryRow(ctx, `
		SELECT id, identifier, expires_at
		FROM auth.verification
		WHERE token = ? AND expires_at > NOW()
		LIMIT 1
	`, token).Scan(&id, &identifier, &expiresAt)

	if err != nil {
		return "", fmt.Errorf("token not found or expired")
	}

	// Delete the used verification token
	_, err = p.db.Exec(ctx, `
		DELETE FROM auth.verification
		WHERE id = ?
	`, id)

	if err != nil {
		return "", fmt.Errorf("failed to delete used token: %w", err)
	}

	return identifier, nil
}

// GetUserByEmail retrieves a user by email
func (p *Plugin) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := p.db.QueryRow(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		WHERE email = ?
	`, email).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

// CreateUserWithEmail creates a user with email
func (p *Plugin) CreateUserWithEmail(ctx context.Context, email string) (*models.User, error) {
	// 1. Create core user
	user, err := p.sessionService.GetDB().CreateUser(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Update email (assuming column exists)
	_, err = p.db.Exec(ctx, `
		UPDATE auth.user
		SET email = ?, email_verified = ?
		WHERE id = ?
	`, email, false, user.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to set email: %w", err)
	}

	return user, nil
}
