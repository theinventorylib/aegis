package types

import (
	"encoding/json"
	"time"
)

// User represents the core user identity model in the authentication system.
// This is the primary entity that represents a person or service using the application.
//
// Users can have multiple Accounts (one per authentication provider) but maintain
// a single unified identity. The Email field is optional to support OAuth-only users
// who may not provide an email address.
//
// The Metadata field allows storing arbitrary JSON data for application-specific
// user attributes without requiring schema changes.
type User struct {
	// ID is the unique identifier for this user (typically a ULID or UUID)
	ID string `json:"id"`

	// Avatar is the URL to the user's profile picture
	Avatar string `json:"avatar,omitempty"`

	// Name is the user's display name
	Name string `json:"name,omitempty"`

	// Email is the user's email address (optional for OAuth-only users)
	Email string `json:"email,omitempty"`

	// CreatedAt is when the user account was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the user account was last modified
	UpdatedAt time.Time `json:"updatedAt"`

	// Disabled indicates if the user account has been deactivated
	Disabled bool `json:"disabled"`

	// Metadata stores custom JSON data for application-specific attributes
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// GetID returns the user's unique identifier.
func (u *User) GetID() string { return u.ID }

// SetID sets the user's unique identifier.
func (u *User) SetID(id string) { u.ID = id }

// GetEmail returns the user's email address.
func (u *User) GetEmail() string { return u.Email }

// SetEmail sets the user's email address.
func (u *User) SetEmail(email string) { u.Email = email }

// GetName returns the user's display name.
func (u *User) GetName() string { return u.Name }

// SetName sets the user's display name.
func (u *User) SetName(name string) { u.Name = name }

// SetCreatedAt sets the user creation timestamp.
func (u *User) SetCreatedAt(t time.Time) { u.CreatedAt = t }

// SetUpdatedAt sets the user last-modified timestamp.
func (u *User) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }

// Account represents a provider-specific authentication account linked to a User.
// A single User can have multiple Accounts (e.g., one for email/password, one for
// Google OAuth, one for GitHub OAuth, etc.).
//
// For credential-based auth (email/password), the ProviderAccountID is typically
// the email address, and PasswordHash contains the hashed password.
//
// For OAuth providers, AccessToken and RefreshToken store the provider's tokens,
// and ExpiresAt tracks when the access token expires. The PasswordHash field is
// not used for OAuth accounts.
type Account struct {
	// ID is the unique identifier for this account
	ID string `json:"id"`

	// UserID links this account to a User
	UserID string `json:"userId"`

	// Provider identifies the authentication method (e.g., "credentials", "google", "github")
	Provider string `json:"provider"`

	// ProviderAccountID is the provider-specific user identifier
	// (e.g., email for credentials, OAuth provider user ID for OAuth)
	ProviderAccountID string `json:"providerAccountId"`

	// PasswordHash stores the hashed password for credential-based accounts.
	// Never returned in JSON responses (json:"-").
	PasswordHash string `json:"-"`

	// AccessToken stores the OAuth access token (OAuth providers only)
	AccessToken string `json:"accessToken,omitempty"`

	// RefreshToken stores the OAuth refresh token (OAuth providers only)
	RefreshToken string `json:"refreshToken,omitempty"`

	// ExpiresAt indicates when the access token expires (OAuth providers only)
	ExpiresAt time.Time `json:"expiresAt,omitempty"`

	// CreatedAt is when this account was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when this account was last modified
	UpdatedAt time.Time `json:"updatedAt"`

	// Metadata stores custom JSON data for provider-specific attributes
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// GetID returns the account's unique identifier.
func (a *Account) GetID() string { return a.ID }

// SetID sets the account's unique identifier.
func (a *Account) SetID(id string) { a.ID = id }

// GetUserID returns the ID of the user this account belongs to.
func (a *Account) GetUserID() string { return a.UserID }

// SetUserID sets the ID of the user this account belongs to.
func (a *Account) SetUserID(id string) { a.UserID = id }

// GetProvider returns the authentication provider name.
func (a *Account) GetProvider() string { return a.Provider }

// SetProvider sets the authentication provider name.
func (a *Account) SetProvider(p string) { a.Provider = p }

// GetPasswordHash returns the hashed password (for credential-based accounts).
func (a *Account) GetPasswordHash() string { return a.PasswordHash }

// SetPasswordHash sets the hashed password (for credential-based accounts).
func (a *Account) SetPasswordHash(h string) { a.PasswordHash = h }

// SetCreatedAt sets the account creation timestamp.
func (a *Account) SetCreatedAt(t time.Time) { a.CreatedAt = t }

// SetUpdatedAt sets the account last-modified timestamp.
func (a *Account) SetUpdatedAt(t time.Time) { a.UpdatedAt = t }

// GetUpdatedAt returns the account last-modified timestamp.
func (a *Account) GetUpdatedAt() time.Time { return a.UpdatedAt }

// GetCreatedAt returns the account creation timestamp.
func (a *Account) GetCreatedAt() time.Time { return a.CreatedAt }

// GetExpiresAt returns when the OAuth access token expires.
func (a *Account) GetExpiresAt() time.Time { return a.ExpiresAt }

// SetExpiresAt sets when the OAuth access token expires.
func (a *Account) SetExpiresAt(t time.Time) { a.ExpiresAt = t }

// GetAccessToken returns the OAuth access token.
func (a *Account) GetAccessToken() string { return a.AccessToken }

// SetAccessToken sets the OAuth access token.
func (a *Account) SetAccessToken(s string) { a.AccessToken = s }

// GetRefreshToken returns the OAuth refresh token.
func (a *Account) GetRefreshToken() string { return a.RefreshToken }

// SetRefreshToken sets the OAuth refresh token.
func (a *Account) SetRefreshToken(s string) { a.RefreshToken = s }

// GetProviderAccountID returns the provider-specific user identifier.
func (a *Account) GetProviderAccountID() string { return a.ProviderAccountID }

// SetProviderAccountID sets the provider-specific user identifier.
func (a *Account) SetProviderAccountID(s string) { a.ProviderAccountID = s }

// Verification represents a temporary verification token used for various
// authentication flows such as email verification, password reset, OTP codes,
// magic links, and other time-limited verification mechanisms.
//
// The Type field distinguishes between different verification purposes
// (e.g., "email", "reset", "otp").
//
// Verifications are always temporary and should be deleted after use or
// after they expire.
type Verification struct {
	// ID is the unique identifier for this verification
	ID string `json:"id"`

	// Identifier is the target of verification (e.g., email address, phone number)
	Identifier string `json:"identifier"`

	// Token is the secret verification code or token
	Token string `json:"token"`

	// Type categorizes the verification purpose (e.g., "email", "reset", "otp")
	Type string `json:"type"`

	// ExpiresAt indicates when this verification token expires
	ExpiresAt time.Time `json:"expiresAt"`

	// CreatedAt is when this verification was created
	CreatedAt time.Time `json:"createdAt"`

	// Metadata stores custom JSON data for verification-specific attributes
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// GetID returns the verification's unique identifier.
func (v *Verification) GetID() string { return v.ID }

// SetID sets the verification's unique identifier.
func (v *Verification) SetID(id string) { v.ID = id }

// GetToken returns the secret verification token.
func (v *Verification) GetToken() string { return v.Token }

// SetToken sets the secret verification token.
func (v *Verification) SetToken(t string) { v.Token = t }

// GetIdentifier returns the target identifier being verified.
func (v *Verification) GetIdentifier() string { return v.Identifier }

// SetIdentifier sets the target identifier being verified.
func (v *Verification) SetIdentifier(i string) { v.Identifier = i }

// SetCreatedAt sets the verification creation timestamp.
func (v *Verification) SetCreatedAt(t time.Time) { v.CreatedAt = t }

// GetCreatedAt returns the verification creation timestamp.
func (v *Verification) GetCreatedAt() time.Time { return v.CreatedAt }

// SetExpiresAt sets the verification expiration timestamp.
func (v *Verification) SetExpiresAt(t time.Time) { v.ExpiresAt = t }

// GetExpiresAt returns the verification expiration timestamp.
func (v *Verification) GetExpiresAt() time.Time { return v.ExpiresAt }

// Session represents an active user session in the authentication system.
// Sessions track authenticated user activity and enable stateful authentication
// through session tokens and optional refresh tokens.
//
// Sessions have a limited lifetime defined by ExpiresAt. When a session expires,
// users must re-authenticate unless a refresh token flow is implemented to
// generate new sessions.
//
// IPAddress and UserAgent are captured for security auditing and can be used
// to detect suspicious activity or allow users to review active sessions.
type Session struct {
	// ID is the unique identifier for this session
	ID string `json:"id"`

	// UserID links this session to a User
	UserID string `json:"userId"`

	// Token is the session authentication token used in requests
	Token string `json:"token"`

	// RefreshToken is an optional long-lived token for obtaining new sessions
	RefreshToken string `json:"refreshToken,omitempty"`

	// ExpiresAt indicates when this session becomes invalid
	ExpiresAt time.Time `json:"expiresAt"`

	// CreatedAt is when this session was created
	CreatedAt time.Time `json:"createdAt"`

	// IPAddress of the client that created this session
	IPAddress string `json:"ipAddress,omitempty"`

	// UserAgent of the client that created this session
	UserAgent string `json:"userAgent,omitempty"`

	// Metadata stores custom JSON data for session-specific attributes
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// GetID returns the session's unique identifier.
func (s *Session) GetID() string { return s.ID }

// SetID sets the session's unique identifier.
func (s *Session) SetID(id string) { s.ID = id }

// GetUserID returns the ID of the user this session belongs to.
func (s *Session) GetUserID() string { return s.UserID }

// SetUserID sets the ID of the user this session belongs to.
func (s *Session) SetUserID(id string) { s.UserID = id }

// GetToken returns the session authentication token.
func (s *Session) GetToken() string { return s.Token }

// SetToken sets the session authentication token.
func (s *Session) SetToken(t string) { s.Token = t }

// GetRefreshToken returns the optional refresh token.
func (s *Session) GetRefreshToken() string { return s.RefreshToken }

// SetRefreshToken sets the optional refresh token.
func (s *Session) SetRefreshToken(r string) { s.RefreshToken = r }

// SetCreatedAt sets the session creation timestamp.
func (s *Session) SetCreatedAt(t time.Time) { s.CreatedAt = t }

// SetExpiresAt sets the session expiration timestamp.
func (s *Session) SetExpiresAt(t time.Time) { s.ExpiresAt = t }

// GetExpiresAt returns the session expiration timestamp.
func (s *Session) GetExpiresAt() time.Time { return s.ExpiresAt }

// SetIPAddress sets the client IP address that created this session.
func (s *Session) SetIPAddress(ip string) { s.IPAddress = ip }

// SetUserAgent sets the client user agent that created this session.
func (s *Session) SetUserAgent(ua string) { s.UserAgent = ua }
