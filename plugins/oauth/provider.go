package oauth

// Provider defines the interface for OAuth provider implementations
// This abstraction allows users to use Goth (default/recommended) or provide custom implementations
type Provider interface {
	// Name returns the provider identifier (e.g., "google", "github", "apple")
	Name() string

	// GetAuthURL returns the authorization URL for the OAuth flow
	GetAuthURL(state string) (string, error)

	// Exchange exchanges an authorization code for user information
	Exchange(code string) (*User, error)
}

// User represents a user from an OAuth provider
type User struct {
	ID           string // Provider-specific user ID
	Email        string
	Name         string
	AvatarURL    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64 // Unix timestamp
}
