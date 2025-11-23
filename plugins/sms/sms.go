package sms

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

// Plugin represents the SMS plugin for Aegis
type Plugin struct {
	db             db.DBProvider
	provider       Provider
	otpExpiry      time.Duration
	otpLength      int
	passwordPlugin *password.Plugin // For phone+password authentication
	sessionService *core.SessionService
}

// Config holds SMS plugin configuration
type Config struct {
	DB             db.DBProvider
	Provider       Provider      // SMS sending provider
	OTPExpiry      time.Duration // OTP expiry duration
	OTPLength      int
	PasswordPlugin *password.Plugin // Optional: for phone+password auth
	SessionService *core.SessionService
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
		db:             cfg.DB,
		provider:       cfg.Provider,
		otpExpiry:      cfg.OTPExpiry,
		otpLength:      cfg.OTPLength,
		passwordPlugin: cfg.PasswordPlugin,
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

// Init initializes the plugin
func (p *Plugin) Init(ctx context.Context, a plugins.Aegis) error {
	// Get session service from Aegis instance
	if app, ok := a.(*aegis.Aegis); ok {
		p.sessionService = app.GetSessionService()
	}
	return nil
}

// MountRoutes registers HTTP routes for the SMS plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// SMS OTP routes
	router.POST(prefix+"/sms/send", handlers.SendOTPHandler)
	router.POST(prefix+"/sms/verify", handlers.VerifyOTPHandler)

	// Phone+password authentication (if password plugin configured)
	if p.passwordPlugin != nil {
		router.POST(prefix+"/sms/login", handlers.LoginWithPhoneHandler)
	}
}

// Dependencies returns external package dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	if p.passwordPlugin != nil {
		return []plugins.Dependency{
			{
				Package: "github.com/theinventorylib/aegis/plugins/password",
				Version: "latest",
				Purpose: "Password authentication for phone+password login",
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
		_, err := p.db.Exec(ctx, `
			DELETE FROM auth.verification
			WHERE identifier = ? AND type = ?
		`, userID, purpose)
		if err != nil {
			return fmt.Errorf("failed to invalidate existing OTPs: %w", err)
		}
	}

	// Create verification record
	id := core.GenerateID()
	expiresAt := time.Now().Add(p.otpExpiry)
	createdAt := time.Now()

	_, err = p.db.Exec(ctx, `
		INSERT INTO auth.verification (id, identifier, token, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, phoneNumber, code, purpose, expiresAt, createdAt)

	if err != nil {
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

	var id string
	var token string
	var expiresAt time.Time

	err := p.db.QueryRow(ctx, `
		SELECT id, token, expires_at
		FROM auth.verification
		WHERE identifier = ? AND type = ? AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, phoneNumber, purpose).Scan(&id, &token, &expiresAt)

	if err != nil {
		return false, fmt.Errorf("OTP not found or expired")
	}

	// Verify code
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

// GetUserByPhone retrieves a user by phone number
func (p *Plugin) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := p.db.QueryRow(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		WHERE phone_number = ?
	`, phone).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}
