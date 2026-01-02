package oauth

// ProviderOption is a functional option for customizing provider configuration.
//
// This pattern allows flexible provider configuration with sensible defaults.
// Options can be chained to customize scopes, PKCE, sign-up behavior, etc.
//
// Example:
//
//	cfg := oauth.Google(clientID, clientSecret,
//	    oauth.WithScopes("email", "profile", "calendar"),
//	    oauth.WithPKCE(),
//	    oauth.WithDisableImplicitSignUp(),
//	)
type ProviderOption func(*ProviderConfig)

// WithScopes sets custom OAuth scopes for the provider.
//
// Scopes determine what user data the provider will share. Each provider
// has different scope names and defaults.
//
// Example:
//
//	oauth.Google(clientID, clientSecret,
//	    oauth.WithScopes("email", "profile", "calendar.readonly"),
//	)
func WithScopes(scopes ...string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.Scopes = scopes
	}
}

// WithProviderID sets a custom provider ID.
//
// Useful for having multiple instances of the same provider with different
// configurations (e.g., LINE for Japan vs Taiwan, Google for different tenants).
//
// Example:
//
//	// LINE for Japan
//	lineJP := oauth.LINE(clientID_JP, clientSecret_JP,
//	    oauth.WithProviderID("line-jp"),
//	)
//
//	// LINE for Taiwan
//	lineTW := oauth.LINE(clientID_TW, clientSecret_TW,
//	    oauth.WithProviderID("line-tw"),
//	)
func WithProviderID(id string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.ProviderID = id
	}
}

// WithPKCE enables PKCE (Proof Key for Code Exchange) for enhanced security.
//
// PKCE protects against authorization code interception attacks, especially
// important for mobile apps and public clients that can't securely store secrets.
//
// Recommended for:
//   - Mobile apps (iOS, Android)
//   - Single-page applications (SPAs)
//   - Any public OAuth client
func WithPKCE() ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.PKCE = true
	}
}

// WithDiscoveryURL sets the OIDC discovery URL for automatic endpoint configuration.
//
// For OpenID Connect providers, the discovery URL provides all OAuth endpoints
// (authorization, token, userinfo, JWKS) automatically via a JSON document.
//
// Example:
//
//	oauth.Generic("custom", clientID, clientSecret,
//	    oauth.WithDiscoveryURL("https://auth.example.com/.well-known/openid-configuration"),
//	)
func WithDiscoveryURL(url string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.DiscoveryURL = url
	}
}

// WithPrompt sets the OAuth prompt parameter.
//
// The prompt parameter controls how the provider asks for user consent:
//   - "none": No UI shown (silent auth, may fail if interaction required)
//   - "login": Always show login screen (even if logged in)
//   - "consent": Always show consent screen
//   - "select_account": Show account picker
//
// Example:
//
//	oauth.Google(clientID, clientSecret,
//	    oauth.WithPrompt("select_account"), // Always show account picker
//	)
func WithPrompt(prompt string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.Prompt = prompt
	}
}

// WithAccessType sets the access type (e.g., "offline" for refresh tokens).
//
// Setting access_type="offline" requests a refresh token from the provider,
// allowing token renewal without re-authentication. This is provider-specific:
//   - Google: access_type=offline
//   - Microsoft: access_type=offline (or prompt=consent)
//
// Example:
//
//	oauth.Google(clientID, clientSecret,
//	    oauth.WithAccessType("offline"), // Request refresh token
//	    oauth.WithPrompt("consent"),     // Force consent to get refresh token
//	)
func WithAccessType(accessType string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.AccessType = accessType
	}
}

// WithDisableImplicitSignUp disables automatic sign-up for new OAuth users.
//
// When enabled, users must be explicitly invited or pre-created before they
// can sign in via OAuth. Useful for enterprise applications with controlled
// user provisioning.
//
// Behavior:
//   - Existing user + OAuth: Link OAuth to existing account (allowed)
//   - New OAuth user: Return error "Sign-up not allowed" (blocked)
//
// Example:
//
//	oauth.Google(clientID, clientSecret,
//	    oauth.WithDisableImplicitSignUp(), // Require pre-existing accounts
//	)
func WithDisableImplicitSignUp() ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.DisableImplicitSignUp = true
	}
}

// WithDisableSignUp disables sign-up entirely (only existing users can sign in).
//
// This is stricter than WithDisableImplicitSignUp - it prevents both new user
// creation and OAuth linking to existing accounts.
//
// Behavior:
//   - Existing user with OAuth already linked: Sign-in allowed
//   - Existing user without OAuth: Linking blocked
//   - New user: Sign-up blocked
//
// Use Case: Read-only OAuth for authentication of pre-provisioned users only.
func WithDisableSignUp() ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.DisableSignUp = true
	}
}

// WithOverrideUserInfo enables updating user info on each sign-in.
//
// By default, user profile data (name, email, avatar) is only saved on first
// sign-up. Enabling this option updates the user profile every time they sign
// in via OAuth, keeping data synchronized with the provider.
//
// Use Cases:
//   - Keep user names/avatars up-to-date from provider
//   - Sync email changes from provider
//   - Corporate directory synchronization
//
// Caution:
//   - May overwrite user-edited profile data
//   - Consider allowing users to opt out of sync
func WithOverrideUserInfo() ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.OverrideUserInfo = true
	}
}

// WithUserInfoFetcher sets a custom user info fetcher function.
//
// Use this to customize how user data is retrieved from the provider,
// for example to call additional API endpoints or parse non-standard responses.
//
// Example:
//
//	fetchUser := func(tokens *oauth.OAuthTokens) (*oauth.User, error) {
//	    // Call custom API with access token
//	    resp, _ := http.Get("https://api.example.com/user?access_token=" + tokens.AccessToken)
//	    // Parse custom response format
//	    var data map[string]interface{}
//	    json.NewDecoder(resp.Body).Decode(&data)
//	    return &oauth.User{...}, nil
//	}
//
//	oauth.Generic("custom", clientID, clientSecret,
//	    oauth.WithUserInfoFetcher(fetchUser),
//	)
func WithUserInfoFetcher(fn func(*OAuthTokens) (*User, error)) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.GetUserInfo = fn
	}
}

// WithProfileMapper sets a custom profile mapper function.
//
// Use this to transform provider-specific profile data into Aegis User format.
// Useful for extracting custom fields or handling non-standard profile structures.
//
// Example:
//
//	mapProfile := func(profile map[string]interface{}) (*oauth.User, error) {
//	    return &oauth.User{
//	        User: auth.User{
//	            ID:     profile["sub"].(string),
//	            Email:  profile["email"].(string),
//	            Name:   profile["display_name"].(string),
//	            Avatar: profile["picture_url"].(string),
//	        },
//	        ProviderData: profile,
//	    }, nil
//	}
//
//	oauth.Generic("custom", clientID, clientSecret,
//	    oauth.WithProfileMapper(mapProfile),
//	)
func WithProfileMapper(fn func(map[string]interface{}) (*User, error)) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.MapProfile = fn
	}
}

// WithRedirectURI sets a custom redirect URI for the provider.
//
// By default, the plugin constructs redirect URIs as:
//
//	{CallbackURL}/oauth/{provider}/callback
//
// Use this to override with a provider-specific redirect URI, for example
// if you've registered a different callback URL in the provider's console.
//
// Example:
//
//	oauth.Google(clientID, clientSecret,
//	    oauth.WithRedirectURI("https://example.com/custom/google/callback"),
//	)
func WithRedirectURI(uri string) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.RedirectURI = uri
	}
}
