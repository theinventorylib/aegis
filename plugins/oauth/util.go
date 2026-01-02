package oauth

import (
	"fmt"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/amazon"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/bitbucket"
	"github.com/markbates/goth/providers/discord"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/line"
	"github.com/markbates/goth/providers/linkedin"
	"github.com/markbates/goth/providers/microsoftonline"
	"github.com/markbates/goth/providers/slack"
	"github.com/markbates/goth/providers/spotify"
	"github.com/markbates/goth/providers/twitch"
	"github.com/markbates/goth/providers/twitter"
)

// CreateGothProvider creates a goth.Provider from ProviderConfig.
//
// This factory function instantiates the correct Goth provider based on the
// ProviderType, applying scopes and callback URL. It handles provider-specific
// initialization quirks (e.g., Apple's JWT client secret).
//
// Supported Providers:
//   - google, github, line, microsoft, apple
//   - discord, slack, gitlab, bitbucket, twitter
//   - linkedin, spotify, twitch, amazon
//   - generic (manual endpoint configuration)
//
// Parameters:
//   - cfg: Provider configuration with type, credentials, and options
//   - callbackURL: OAuth callback URL for this provider
//
// Returns:
//   - goth.Provider: Initialized Goth provider
//   - error: Unsupported provider type or missing configuration
//
// Example:
//
//	cfg := oauth.ProviderConfig{
//	    ProviderType: "google",
//	    ClientID:     "...",
//	    ClientSecret: "...",
//	    Scopes:       []string{"email", "profile"},
//	}
//	provider, _ := oauth.CreateGothProvider(cfg, "https://example.com/auth/oauth/google/callback")
func CreateGothProvider(cfg ProviderConfig, callbackURL string) (goth.Provider, error) {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = getDefaultScopes(cfg.ProviderType)
	}

	switch cfg.ProviderType {
	case "google":
		return google.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "github":
		return github.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "line":
		return line.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "microsoft":
		return microsoftonline.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "apple":
		// Apple requires special setup - client secret is actually a JWT
		return apple.New(cfg.ClientID, cfg.ClientSecret, callbackURL, nil, scopes...), nil

	case "discord":
		return discord.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "slack":
		return slack.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "gitlab":
		return gitlab.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "bitbucket":
		return bitbucket.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "twitter":
		return twitter.New(cfg.ClientID, cfg.ClientSecret, callbackURL), nil

	case "linkedin":
		return linkedin.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "spotify":
		return spotify.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "twitch":
		return twitch.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "amazon":
		return amazon.New(cfg.ClientID, cfg.ClientSecret, callbackURL, scopes...), nil

	case "generic":
		// For generic providers, require manual endpoint configuration or discovery
		if cfg.DiscoveryURL != "" {
			return nil, fmt.Errorf("OIDC discovery not yet implemented - use explicit endpoints")
		}
		if cfg.AuthURL == "" || cfg.TokenURL == "" {
			return nil, fmt.Errorf("generic provider requires AuthURL and TokenURL")
		}
		// Generic providers would need custom implementation or use openidConnect provider
		return nil, fmt.Errorf("generic providers not yet fully implemented")

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.ProviderType)
	}
}

// getDefaultScopes returns sensible default OAuth scopes for known providers.
//
// These scopes request basic user profile information (email, name, avatar)
// suitable for authentication. You can override these with WithScopes().
//
// Default Scopes by Provider:
//   - google: ["email", "profile"]
//   - github: ["user:email"]
//   - line: ["profile", "openid", "email"]
//   - microsoft: ["openid", "email", "profile"]
//   - apple: ["name", "email"]
//   - discord: ["identify", "email"]
//   - slack: ["openid", "profile", "email"]
//
// Parameters:
//   - providerType: Provider type ("google", "github", etc.)
//
// Returns:
//   - []string: Default scopes for the provider (empty if unknown)
func getDefaultScopes(providerType string) []string {
	switch providerType {
	case "google":
		return []string{"email", "profile"}
	case "github":
		return []string{"user:email"}
	case "line":
		return []string{"profile", "openid", "email"}
	case "microsoft":
		return []string{"openid", "email", "profile"}
	case "apple":
		return []string{"name", "email"}
	case "discord":
		return []string{"identify", "email"}
	case "slack":
		return []string{"openid", "profile", "email"}
	case "gitlab":
		return []string{"read_user"}
	case "bitbucket":
		return []string{"account", "email"}
	case "linkedin":
		return []string{"r_liteprofile", "r_emailaddress"}
	case "spotify":
		return []string{"user-read-email"}
	case "twitch":
		return []string{"user:read:email"}
	default:
		return []string{}
	}
}

// ========== Provider Configuration Helpers ==========
//
// These helper functions provide pre-configured ProviderConfig instances for
// common OAuth providers. They set sensible defaults for scopes and discovery
// URLs while allowing customization via ProviderOption.
//
// Example:
//
//	// Basic Google configuration
//	googleCfg := oauth.Google(clientID, clientSecret)
//
//	// Google with custom scopes
//	googleCfg := oauth.Google(clientID, clientSecret,
//	    oauth.WithScopes("email", "profile", "calendar.readonly"),
//	    oauth.WithPrompt("consent"),
//	)
//
//	// Multiple providers
//	cfg := &oauth.Config{
//	    Providers: []oauth.ProviderConfig{
//	        oauth.Google(googleClientID, googleClientSecret),
//	        oauth.GitHub(githubClientID, githubClientSecret),
//	        oauth.LINE(lineClientID, lineClientSecret),
//	    },
//	}

// Google creates a Google OAuth provider configuration.
//
// Default Scopes: ["openid", "email", "profile"]
// Discovery: https://accounts.google.com/.well-known/openid-configuration
//
// Parameters:
//   - clientID: Google OAuth client ID (from Google Cloud Console)
//   - clientSecret: Google OAuth client secret
//   - opts: Optional customization (scopes, prompt, etc.)
//
// Returns:
//   - ProviderConfig: Google provider configuration
//
// Example:
//
//	googleCfg := oauth.Google("123456.apps.googleusercontent.com", "secret",
//	    oauth.WithScopes("email", "profile", "calendar.readonly"),
//	    oauth.WithAccessType("offline"), // Request refresh token
//	    oauth.WithPrompt("consent"),      // Force consent to get refresh token
//	)
func Google(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "google",
		ProviderType: "google",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"openid", "email", "profile"},
		DiscoveryURL: "https://accounts.google.com/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// GitHub creates a GitHub OAuth provider configuration.
//
// Default Scopes: ["user:email"]
// No discovery URL (GitHub uses fixed endpoints)
//
// Parameters:
//   - clientID: GitHub OAuth App client ID
//   - clientSecret: GitHub OAuth App client secret
//   - opts: Optional customization
//
// Returns:
//   - ProviderConfig: GitHub provider configuration
func GitHub(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "github",
		ProviderType: "github",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:email"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// LINE creates a LINE OAuth provider configuration.
//
// LINE supports multiple channels for different countries:
//   - Japan: Regular LINE Login
//   - Thailand, Taiwan, Indonesia: Country-specific configurations
//
// Default Scopes: ["profile", "openid", "email"]
// Discovery: https://access.line.me/.well-known/openid-configuration
//
// For multiple LINE channels:
//
//	lineJP := oauth.LINE(clientID_JP, clientSecret_JP,
//	    oauth.WithProviderID("line-jp"),
//	)
//	lineTW := oauth.LINE(clientID_TW, clientSecret_TW,
//	    oauth.WithProviderID("line-tw"),
//	)
//
// Parameters:
//   - clientID: LINE Channel ID
//   - clientSecret: LINE Channel Secret
//   - opts: Optional customization
//
// Returns:
//   - ProviderConfig: LINE provider configuration
func LINE(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "line",
		ProviderType: "line",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"profile", "openid", "email"},
		DiscoveryURL: "https://access.line.me/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Microsoft creates a Microsoft/Azure AD OAuth provider configuration.
//
// Default Scopes: ["openid", "email", "profile"]
// Discovery: Uses tenant-specific discovery URL
//
// Parameters:
//   - clientID: Azure AD App client ID
//   - clientSecret: Azure AD App client secret
//   - tenantID: Azure AD tenant ID (or "common" for multi-tenant)
//   - opts: Optional customization
//
// Returns:
//   - ProviderConfig: Microsoft provider configuration
func Microsoft(clientID, clientSecret, tenantID string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "microsoft",
		ProviderType: "microsoft",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"openid", "email", "profile"},
		DiscoveryURL: "https://login.microsoftonline.com/" + tenantID + "/v2.0/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Apple creates an Apple Sign In provider configuration.
//
// Apple requires special setup:
//   - Client Secret: Not a simple string, but a JWT signed with your private key
//   - Team ID: Your Apple Developer Team ID
//
// Default Scopes: ["name", "email"]
// Discovery: https://appleid.apple.com/.well-known/openid-configuration
//
// Note: Apple's "client secret" is actually a JWT that you must generate and
// sign with your private key. See Apple's documentation for details.
//
// Parameters:
//   - clientID: Apple Service ID
//   - clientSecret: JWT signed with your private key
//   - teamID: Apple Developer Team ID (currently unused)
//   - opts: Optional customization
//
// Returns:
//   - ProviderConfig: Apple provider configuration
func Apple(clientID, clientSecret, teamID string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "apple",
		ProviderType: "apple",
		ClientID:     clientID,
		ClientSecret: clientSecret, // Apple uses a JWT as client secret
		Scopes:       []string{"name", "email"},
		DiscoveryURL: "https://appleid.apple.com/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Discord creates a Discord OAuth provider configuration.
//
// Default Scopes: ["identify", "email"]
//
// Parameters:
//   - clientID: Discord Application client ID
//   - clientSecret: Discord Application client secret
//   - opts: Optional customization
func Discord(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "discord",
		ProviderType: "discord",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"identify", "email"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Slack creates a Slack OAuth provider configuration.
//
// Default Scopes: ["openid", "profile", "email"]
// Discovery: https://slack.com/.well-known/openid-configuration
func Slack(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "slack",
		ProviderType: "slack",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"openid", "profile", "email"},
		DiscoveryURL: "https://slack.com/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// GitLab creates a GitLab OAuth provider configuration.
//
// Default Scopes: ["read_user", "openid", "profile", "email"]
//
// Supports both GitLab.com and self-hosted instances.
func GitLab(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "gitlab",
		ProviderType: "gitlab",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"read_user", "openid", "profile", "email"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Bitbucket creates a Bitbucket OAuth provider configuration.
//
// Default Scopes: ["account", "email"]
func Bitbucket(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "bitbucket",
		ProviderType: "bitbucket",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"account", "email"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Twitter (X) creates a Twitter/X OAuth provider configuration.
//
// Default Scopes: ["tweet.read", "users.read"]
//
// Note: Twitter's OAuth implementation has changed over time. This uses OAuth 2.0.
func Twitter(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "twitter",
		ProviderType: "twitter",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"tweet.read", "users.read"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// LinkedIn creates a LinkedIn OAuth provider configuration.
//
// Default Scopes: ["r_liteprofile", "r_emailaddress"]
//
// Note: LinkedIn's API and scopes change frequently. Verify current scopes in LinkedIn's docs.
func LinkedIn(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "linkedin",
		ProviderType: "linkedin",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"r_liteprofile", "r_emailaddress"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Spotify creates a Spotify OAuth provider configuration.
//
// Default Scopes: ["user-read-email", "user-read-private"]
func Spotify(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "spotify",
		ProviderType: "spotify",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user-read-email", "user-read-private"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Twitch creates a Twitch OAuth provider configuration.
//
// Default Scopes: ["user:read:email"]
// Discovery: https://id.twitch.tv/oauth2/.well-known/openid-configuration
func Twitch(clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   "twitch",
		ProviderType: "twitch",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:read:email"},
		DiscoveryURL: "https://id.twitch.tv/oauth2/.well-known/openid-configuration",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Generic creates a custom OAuth provider configuration.
//
// Use this for OAuth providers not included in the pre-configured helpers.
// You must provide either:
//   - Discovery URL (OIDC providers): Automatic endpoint discovery
//   - Manual endpoints: AuthURL, TokenURL, UserInfoURL
//
// Parameters:
//   - providerID: Unique provider identifier (used in URLs)
//   - clientID: OAuth client ID from provider
//   - clientSecret: OAuth client secret from provider
//   - opts: Required configuration (scopes, endpoints, etc.)
//
// Returns:
//   - ProviderConfig: Generic provider configuration
//
// Example (OIDC Discovery):
//
//	custom := oauth.Generic("keycloak", clientID, clientSecret,
//	    oauth.WithDiscoveryURL("https://auth.example.com/realms/master/.well-known/openid-configuration"),
//	    oauth.WithScopes("openid", "email", "profile"),
//	)
//
// Example (Manual Endpoints):
//
//	custom := oauth.Generic("custom", clientID, clientSecret,
//	    oauth.WithScopes("email"),
//	)
//	custom.AuthURL = "https://provider.com/oauth/authorize"
//	custom.TokenURL = "https://provider.com/oauth/token"
//	custom.UserInfoURL = "https://provider.com/oauth/userinfo"
func Generic(providerID, clientID, clientSecret string, opts ...ProviderOption) ProviderConfig {
	cfg := ProviderConfig{
		ProviderID:   providerID,
		ProviderType: "generic",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ResponseType: "code", // Default to authorization code flow
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
