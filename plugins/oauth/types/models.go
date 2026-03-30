// Package types defines the domain models and types used by the oauth plugin.
package types

import (
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// ProviderOption is a functional option for customizing provider configuration.
type ProviderOption func(*ProviderConfig)

// User represents an OAuth-authenticated user returned from a provider during
// the login or registration flow. It embeds the core auth.User identity fields
// and adds the provider token credentials that were returned at the time of
// authentication.
type User struct {
	auth.User
	// AccessToken is the OAuth access token issued by the provider.
	AccessToken string
	// RefreshToken is the OAuth refresh token, if provided by the provider.
	// Not all providers issue refresh tokens; this may be empty.
	RefreshToken string
	// ExpiresAt is when the access token expires.
	ExpiresAt time.Time
	// ProviderData contains any additional raw data returned by the provider's
	// user-info endpoint, keyed by field name.
	ProviderData map[string]any
}

// Connection represents a persisted OAuth provider link stored in the database.
// Each Connection ties one Aegis user (UserID) to one provider account
// (Provider + ProviderUserID). A single user may have multiple Connections,
// one per distinct provider.
type Connection struct {
	// ID is the unique identifier for this connection record.
	ID string
	// UserID is the Aegis user this connection belongs to.
	UserID string
	// CreatedAt is when the connection was first established.
	CreatedAt time.Time
	// UpdatedAt is when the connection was last refreshed or modified.
	UpdatedAt time.Time
	// Provider is the OAuth provider name (e.g., "google", "github", "discord").
	Provider string
	// ProviderUserID is the user's unique ID within the provider's system.
	ProviderUserID string
	// Email is the email address returned by the provider, if available.
	Email string
	// Name is the display name returned by the provider, if available.
	Name string
	// AvatarURL is the profile picture URL returned by the provider, if available.
	AvatarURL string
	// AccessToken is the most recently issued access token for this connection.
	AccessToken string
	// RefreshToken is the most recently issued refresh token. May be empty if
	// the provider does not support token refresh.
	RefreshToken string
	// ExpiresAt is when the current access token expires.
	ExpiresAt time.Time
	// ProviderData stores arbitrary additional data returned by the provider.
	ProviderData map[string]any
}

// ProviderConfig defines the full configuration for a single OAuth provider.
// It is used when registering a provider with the OAuth plugin.
//
// Mandatory fields for all providers: ProviderID, ProviderType, ClientID,
// ClientSecret, and Scopes. The remaining fields apply depending on the
// provider's protocol and desired behavior.
type ProviderConfig struct {
	// ProviderID is the unique name used to identify this provider within
	// Aegis (e.g., "google", "github"). Used in URLs like /oauth/{providerID}.
	ProviderID string
	// ProviderType indicates the protocol variant: "oidc" for OpenID Connect
	// providers or "oauth2" for plain OAuth 2.0 providers.
	ProviderType string
	// ClientID is the application ID issued by the OAuth provider.
	ClientID string
	// ClientSecret is the application secret issued by the OAuth provider.
	// Keep this confidential and never expose it to clients.
	ClientSecret string
	// AuthURL is the provider's authorization endpoint URL.
	// Required for providers not configured via DiscoveryURL.
	AuthURL string
	// TokenURL is the provider's token endpoint URL.
	// Required for providers not configured via DiscoveryURL.
	TokenURL string
	// UserInfoURL is the provider's user-info endpoint URL.
	// Required for providers not configured via DiscoveryURL.
	UserInfoURL string
	// DiscoveryURL is the OpenID Connect discovery document URL
	// (e.g., "https://accounts.google.com/.well-known/openid-configuration").
	// When set, AuthURL, TokenURL, and UserInfoURL are resolved automatically.
	DiscoveryURL string
	// Scopes is the list of OAuth scopes to request from the provider.
	// Typically includes "openid", "profile", and "email" for OIDC providers.
	Scopes []string
	// RedirectURI is the callback URL registered with the provider.
	// Must exactly match a URI configured in the provider's application settings.
	RedirectURI string
	// PKCE enables Proof Key for Code Exchange (RFC 7636) for the authorization
	// code flow. Recommended for public clients; required by some providers.
	PKCE bool
	// ResponseType is the OAuth response_type parameter (default: "code").
	ResponseType string
	// ResponseMode controls how the provider sends the authorization response
	// to the redirect URI (e.g., "query", "fragment", "form_post").
	ResponseMode string
	// Prompt specifies the authentication prompt behavior (e.g., "consent",
	// "select_account", "none"). Provider-specific and may be ignored.
	Prompt string
	// AccessType controls whether the provider issues a refresh token.
	// Set to "offline" to request offline access and receive refresh tokens.
	AccessType string
	// DisableImplicitSignUp prevents a new Aegis user from being created
	// automatically when an unrecognized provider user completes the OAuth
	// flow. When true, unlinked users receive an error instead of having an
	// account created on their behalf.
	DisableImplicitSignUp bool
	// DisableSignUp prevents any new user from being created via this provider.
	// Only users who already have a linked Connection can authenticate through it.
	DisableSignUp bool
	// OverrideUserInfo forces the user's profile fields (name, avatar, email)
	// to be updated from the provider on every login, overwriting stored values.
	OverrideUserInfo bool
	// GetUserInfo is an optional custom function that fetches the provider user
	// profile from the given token set. Implement this when the provider's
	// user-info endpoint deviates from standard OIDC conventions.
	GetUserInfo func(*Tokens) (*User, error)
	// MapProfile is an optional function to transform the raw provider profile
	// map into an Aegis User. Use this to adapt non-standard provider response
	// shapes without implementing a full GetUserInfo override.
	MapProfile func(map[string]any) (*User, error)
}

// Tokens represents the full OAuth token response obtained after a successful
// authorization code exchange.
type Tokens struct {
	// AccessToken is the bearer token used to authenticate API requests
	// to the provider on behalf of the user.
	AccessToken string
	// RefreshToken can be exchanged for a new AccessToken when the current one
	// expires. Not all providers return a refresh token.
	RefreshToken string
	// ExpiresAt is when the AccessToken expires.
	ExpiresAt time.Time
	// Scopes is the list of scopes actually granted by the provider, which may
	// be a subset of the scopes that were requested.
	Scopes []string
	// Raw contains the raw token response payload from the provider, keyed by
	// field name. Useful for accessing non-standard provider-specific fields.
	Raw map[string]any
}

// LinkAccountRequest is the request body for the link-account endpoint.
// It specifies which provider the authenticated user wants to connect to their
// existing Aegis account.
type LinkAccountRequest struct {
	// Provider is the ID of the OAuth provider to link (e.g., "google").
	Provider string `json:"provider"`
}

// CallbackResponse is the data payload returned by OAuth callback endpoints
// after a successful authentication or account-link flow.
type CallbackResponse struct {
	// User is the authenticated or newly-created Aegis user.
	User *User `json:"user"`
	// Session is the new Aegis session created for the authenticated user.
	Session *auth.Session `json:"session"`
}

// TokenRefreshResponse is the data payload returned after a successful
// provider token refresh, containing updated token metadata.
type TokenRefreshResponse struct {
	// Provider is the OAuth provider whose token was refreshed (e.g., "google").
	Provider string `json:"provider"`
	// ExpiresAt is when the newly issued access token expires.
	ExpiresAt time.Time `json:"expires_at"`
}
