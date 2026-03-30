package oauth

import (
	"fmt"

	"github.com/markbates/goth"
	"github.com/theinventorylib/aegis/auth"
	core "github.com/theinventorylib/aegis/core"
	oauthtypes "github.com/theinventorylib/aegis/plugins/oauth/types"
)

// GothAdapter adapts goth.Provider to Aegis's Provider interface.
//
// This adapter makes Goth the default/recommended OAuth provider implementation
// while keeping it technically optional through abstraction. If you want to use
// a different OAuth library, you can implement the Provider interface without Goth.
//
// Goth Benefits:
//   - 50+ pre-configured OAuth providers (Google, GitHub, Apple, etc.)
//   - Battle-tested OAuth 2.0 / OIDC implementation
//   - Active maintenance and security updates
//   - Provider-specific quirks handled (Apple's JWT client secret, etc.)
//
// Abstraction Benefits:
//   - Aegis core doesn't depend on Goth directly
//   - Easier testing with mock providers
//   - Future flexibility if Goth is discontinued
type GothAdapter struct {
	provider goth.Provider
}

// NewGothAdapter creates a new Goth adapter wrapping a Goth provider.
//
// This function wraps any Goth provider to work with Aegis's OAuth plugin.
// Most users won't call this directly - the plugin creates adapters automatically
// from ProviderConfig using CreateGothProvider.
//
// Parameters:
//   - provider: Goth provider instance (google.New, github.New, etc.)
//
// Returns:
//   - *GothAdapter: Adapter implementing Provider interface
//
// Example:
//
//	googleProvider := google.New(clientID, clientSecret, callbackURL)
//	adapter := oauth.NewGothAdapter(googleProvider)
func NewGothAdapter(provider goth.Provider) *GothAdapter {
	return &GothAdapter{provider: provider}
}

// Name returns the provider identifier (e.g., "google", "github").
func (g *GothAdapter) Name() string {
	return g.provider.Name()
}

// GetAuthURL returns the provider's authorization URL with CSRF state.
//
// This method starts the OAuth session and returns the URL to redirect the
// user to for authorization (e.g., https://accounts.google.com/o/oauth2/auth).
//
// Parameters:
//   - state: CSRF state token (random string for security)
//
// Returns:
//   - string: Authorization URL to redirect user to
//   - error: OAuth session creation error
func (g *GothAdapter) GetAuthURL(state string) (string, error) {
	session, err := g.provider.BeginAuth(state)
	if err != nil {
		return "", fmt.Errorf("failed to begin auth: %w", err)
	}

	authURL, err := session.GetAuthURL()
	if err != nil {
		return "", fmt.Errorf("failed to get auth URL: %w", err)
	}

	return authURL, nil
}

// Exchange exchanges authorization code for user information.
//
// Note: This is a simplified interface. In practice, Aegis uses the full
// gothic.CompleteUserAuth flow which handles the complete OAuth callback
// processing (code exchange, token retrieval, user info fetch).
//
// This method is currently not used - instead, the plugin uses Goth's
// session-based flow directly for more flexibility.
func (g *GothAdapter) Exchange(_ string) (*oauthtypes.User, error) {
	// Note: This is a simplified version. In practice, you'd need to handle
	// the full OAuth flow with sessions. For now, we'll use the CompleteAuth approach
	// which requires gothic to be properly configured.

	// This method signature is simplified for the interface
	// In real usage, you'd call gothic.CompleteUserAuth which handles the full flow
	return nil, fmt.Errorf("use gothic.CompleteUserAuth for full Goth integration")
}

// GothUserToUser converts goth.User to Aegis's User model.
//
// This helper function transforms Goth's OAuth user representation into
// Aegis's User struct, preserving OAuth tokens and provider-specific data.
//
// Field Mapping:
//   - goth.UserID → User.ID (provider's user ID)
//   - goth.Email → User.Email
//   - goth.Name → User.Name
//   - goth.AvatarURL → User.Avatar
//   - goth.AccessToken, RefreshToken, ExpiresAt preserved
//
// Parameters:
//   - gothUser: User data from Goth provider
//
// Returns:
//   - *User: Aegis user model
func GothUserToUser(gothUser goth.User) *oauthtypes.User {
	return &oauthtypes.User{
		User: auth.User{
			ID:     core.SanitizeString(gothUser.UserID, nil),
			Email:  core.SanitizeEmail(gothUser.Email),
			Name:   core.SanitizeString(gothUser.Name, nil),
			Avatar: core.SanitizeURL(gothUser.AvatarURL),
		},
		AccessToken:  gothUser.AccessToken,
		RefreshToken: gothUser.RefreshToken,
		ExpiresAt:    gothUser.ExpiresAt,
		ProviderData: make(map[string]any),
	}
}

// UserToGothUser converts Aegis User to goth.User.
//
// This helper function transforms Aegis's User struct back into Goth's
// representation, useful for interacting with Goth's provider APIs.
//
// Parameters:
//   - oauthUser: Aegis OAuth user
//   - provider: Provider name ("google", "github", etc.)
//
// Returns:
//   - goth.User: Goth user model
func UserToGothUser(oauthUser *oauthtypes.User, provider string) goth.User {
	expiresAt := oauthUser.ExpiresAt

	return goth.User{
		UserID:       oauthUser.ID,
		Email:        oauthUser.Email,
		Name:         oauthUser.Name,
		AvatarURL:    oauthUser.Avatar,
		AccessToken:  oauthUser.AccessToken,
		RefreshToken: oauthUser.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     provider,
	}
}
