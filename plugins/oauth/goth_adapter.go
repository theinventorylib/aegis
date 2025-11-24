package oauth

import (
	"fmt"
	"time"

	"github.com/markbates/goth"
)

// GothAdapter adapts goth.Provider to our Provider interface
// This makes Goth the default/recommended implementation while keeping it optional
type GothAdapter struct {
	provider goth.Provider
}

// NewGothAdapter creates a new Goth adapter
// Example usage:
//
//	googleProvider, _ := google.New("client-id", "client-secret", "callback-url")
//	adapter := oauth.NewGothAdapter(googleProvider)
func NewGothAdapter(provider goth.Provider) *GothAdapter {
	return &GothAdapter{provider: provider}
}

// Name returns the provider identifier
func (g *GothAdapter) Name() string {
	return g.provider.Name()
}

// GetAuthURL returns the authorization URL
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
func (g *GothAdapter) Exchange(_ string) (*User, error) {
	// Note: This is a simplified version. In practice, you'd need to handle
	// the full OAuth flow with sessions. For now, we'll use the CompleteAuth approach
	// which requires gothic to be properly configured.

	// This method signature is simplified for the interface
	// In real usage, you'd call gothic.CompleteUserAuth which handles the full flow
	return nil, fmt.Errorf("use gothic.CompleteUserAuth for full Goth integration")
}

// GothUserToUser converts goth.User to User
// This is a helper function for converting Goth users to our abstraction
func GothUserToUser(gothUser goth.User) *User {
	var expiresAt int64
	if !gothUser.ExpiresAt.IsZero() {
		expiresAt = gothUser.ExpiresAt.Unix()
	}

	return &User{
		ID:           gothUser.UserID,
		Email:        gothUser.Email,
		Name:         gothUser.Name,
		AvatarURL:    gothUser.AvatarURL,
		AccessToken:  gothUser.AccessToken,
		RefreshToken: gothUser.RefreshToken,
		ExpiresAt:    expiresAt,
	}
}

// UserToGothUser converts User to goth.User
// This is a helper function for converting our abstraction to Goth
func UserToGothUser(oauthUser *User, provider string) goth.User {
	var expiresAt time.Time
	if oauthUser.ExpiresAt > 0 {
		expiresAt = time.Unix(oauthUser.ExpiresAt, 0)
	}

	return goth.User{
		UserID:       oauthUser.ID,
		Email:        oauthUser.Email,
		Name:         oauthUser.Name,
		AvatarURL:    oauthUser.AvatarURL,
		AccessToken:  oauthUser.AccessToken,
		RefreshToken: oauthUser.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     provider,
	}
}
