// Package oauth provides OAuth 2.0 / OpenID Connect authentication for Aegis.
//
// This plugin enables "Login with Google", "Login with GitHub", and other OAuth-based
// authentication flows. It uses the Goth library as the provider implementation while
// maintaining abstraction for potential alternatives.
//
// OAuth Flow:
//  1. User clicks "Login with Google" → GET /auth/oauth/google
//  2. Plugin generates CSRF state token and stores it in signed cookie
//  3. Plugin redirects to Google's authorization page
//  4. User approves → Google redirects to /auth/oauth/google/callback?code=xxx&state=xxx
//  5. Plugin validates state token (CSRF protection)
//  6. Plugin exchanges authorization code for access token
//  7. Plugin fetches user profile from Google
//  8. Plugin creates/links Aegis user account and session
//  9. User is authenticated with session cookie
//
// Supported Providers (via Goth):
//   - google: Google OAuth 2.0
//   - github: GitHub OAuth 2.0
//   - line: LINE OAuth 2.0 (Japan, Taiwan, Thailand)
//   - microsoft: Microsoft Azure AD / Office 365
//   - apple: Apple Sign In
//   - discord, slack, gitlab, bitbucket, twitter, linkedin, spotify, twitch, amazon
//   - generic: Custom OAuth 2.0 / OIDC providers (requires manual configuration)
//
// Multi-Provider Setup:
// Configure multiple providers to offer users a choice:
//
//	cfg := &oauth.Config{
//	    CallbackURL: "https://example.com/auth",
//	    Providers: []oauth.ProviderConfig{
//	        {ProviderID: "google", ProviderType: "google", ClientID: "...", ClientSecret: "..."},
//	        {ProviderID: "github", ProviderType: "github", ClientID: "...", ClientSecret: "..."},
//	    },
//	}
//	oauthPlugin := oauth.New(cfg, nil)
//
// Security Features:
//   - CSRF Protection: State tokens with HMAC signing prevent cross-site request forgery
//   - Secure Cookies: HTTPOnly, Secure, SameSite settings via CookieManager
//   - State Expiration: OAuth states expire after 15 minutes
//   - Token Storage: Access/refresh tokens stored in database (not cookies)
//
// Account Linking:
// Users can link multiple OAuth providers to a single Aegis account:
//   - Same email: Automatically linked on first sign-in
//   - Different emails: Manual linking via LinkAccount API
//   - Multiple providers: One user can have Google + GitHub + Apple linked
//
// Database Schema:
//   - oauth_connections table: Stores provider links (user_id, provider, provider_user_id, tokens)
//   - Foreign key to auth.users: Ensures referential integrity
package oauth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/markbates/goth"
	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin provides OAuth 2.0 authentication integration for Aegis.
//
// This plugin manages multiple OAuth providers simultaneously, allowing users to
// authenticate with Google, GitHub, Apple, and other services. It integrates with
// Aegis's user and session management to create unified accounts.
//
// Components:
//   - providerConfigs: Plugin configuration for each provider (client ID, secret, scopes)
//   - gothProviders: Goth provider instances for OAuth protocol handling
//   - stateStore: CSRF protection via signed cookies (state parameter)
//   - store: Database persistence for OAuth connections
//   - sessionService: Creates Aegis sessions after successful OAuth
//
// Thread Safety:
// Plugin is safe for concurrent use after initialization (Init called).
type Plugin struct {
	providerConfigs map[string]ProviderConfig // providerID -> config
	gothProviders   map[string]goth.Provider  // providerID -> goth.Provider
	baseCallbackURL string                    // Base URL for OAuth callbacks
	stateStore      *StateStore               // OAuth state management
	store           Store
	logger          config.Logger
	accountService  *core.AccountService
	userService     *core.UserService
	sessionService  *core.SessionService
	dialect         plugins.Dialect
}

// Config holds OAuth plugin configuration.
//
// This structure defines all OAuth providers to enable and their settings.
// Multiple providers can be configured to offer users authentication choices.
//
// Example:
//
//	cfg := &oauth.Config{
//	    CallbackURL: "https://example.com/auth",
//	    Providers: []oauth.ProviderConfig{
//	        {
//	            ProviderID:   "google",
//	            ProviderType: "google",
//	            ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
//	            ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
//	            Scopes:       []string{"email", "profile"},
//	        },
//	    },
//	}
type Config struct {
	// Providers configures which OAuth providers to enable.
	// Each provider needs a client ID, client secret from the provider's developer console.
	Providers []ProviderConfig

	// CallbackURL is the base URL for OAuth callbacks (e.g., "https://example.com/auth").
	// The plugin appends "/oauth/:provider/callback" to this base.
	// Example: CallbackURL="https://example.com/auth" → callback at "https://example.com/auth/oauth/google/callback"
	CallbackURL string

	// StateSecret is deprecated - Aegis now derives state secrets from master secret.
	// This field is kept for backward compatibility but is ignored.
	StateSecret []byte
}

// New creates a new OAuth plugin with configured providers.
//
// This function initializes the plugin with provider configurations and creates
// Goth provider instances for each configured provider. Providers that fail to
// initialize are logged but don't prevent other providers from working.
//
// Provider Configuration:
// Each provider needs:
//   - ProviderID: Unique identifier (used in URLs like /oauth/google)
//   - ProviderType: Provider implementation type ("google", "github", etc.)
//   - ClientID: OAuth client ID from provider's developer console
//   - ClientSecret: OAuth client secret from provider's developer console
//
// Parameters:
//   - cfg: Plugin configuration with providers and callback URL
//   - store: OAuth connection storage (nil = use DefaultOAuthStore)
//   - dialect: Database dialect (optional, defaults to PostgreSQL)
//
// Returns:
//   - *Plugin: Initialized plugin ready for Init() call
//
// Example:
//
//	cfg := &oauth.Config{
//	    CallbackURL: "https://example.com/auth",
//	    Providers: []oauth.ProviderConfig{
//	        {ProviderID: "google", ProviderType: "google", ClientID: "...", ClientSecret: "..."},
//	        {ProviderID: "github", ProviderType: "github", ClientID: "...", ClientSecret: "..."},
//	    },
//	}
//	plugin := oauth.New(cfg, nil, plugins.DialectPostgres)
func New(cfg *Config, store *Store, dialect ...plugins.Dialect) *Plugin {
	if cfg == nil {
		cfg = &Config{}
	}

	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}

	plugin := &Plugin{
		store:           *store,
		providerConfigs: make(map[string]ProviderConfig),
		gothProviders:   make(map[string]goth.Provider),
		baseCallbackURL: cfg.CallbackURL,
		stateStore:      nil, // Will be initialized in Init
		dialect:         d,
	}

	// Initialize providers from config
	gothProviders := make([]goth.Provider, 0, len(cfg.Providers))
	for _, providerCfg := range cfg.Providers {
		// Build callback URL for this provider
		callbackURL := plugin.buildCallbackURL(providerCfg.ProviderID)
		if providerCfg.RedirectURI != "" {
			callbackURL = providerCfg.RedirectURI
		}

		// Create Goth provider from config
		gothProvider, err := CreateGothProvider(providerCfg, callbackURL)
		if err != nil {
			// Log error but continue with other providers
			fmt.Printf("Warning: failed to create provider %s: %v\n", providerCfg.ProviderID, err)
			continue
		}

		plugin.providerConfigs[providerCfg.ProviderID] = providerCfg
		plugin.gothProviders[providerCfg.ProviderID] = gothProvider
		gothProviders = append(gothProviders, gothProvider)
	}

	// Register all providers with Goth
	if len(gothProviders) > 0 {
		goth.UseProviders(gothProviders...)
	}

	return plugin
}

// buildCallbackURL constructs the callback URL for a provider
func (p *Plugin) buildCallbackURL(providerID string) string {
	if p.baseCallbackURL != "" {
		return p.baseCallbackURL + "/oauth/" + providerID + "/callback"
	}
	// Default fallback (will be overridden by actual request URL)
	return "/auth/oauth/" + providerID + "/callback"
}

// Name returns the plugin identifier
func (p *Plugin) Name() string {
	return "oauth"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "2.0.0" // Version 2.0 with generic provider support
}

// Description returns a human-readable description
func (p *Plugin) Description() string {
	numProviders := len(p.providerConfigs)
	if numProviders == 0 {
		return "OAuth authentication plugin (no providers configured)"
	}
	return fmt.Sprintf("OAuth authentication plugin with %d configured provider(s)", numProviders)
}

// SecretPurposeOAuthState is the purpose string for deriving OAuth state secrets.
// Aegis's secret derivation system uses this to generate a unique secret for
// signing OAuth state cookies,	// SecretPurposeOAuthState is the purpose for OAuth state tokens
// #nosec G101
const SecretPurposeOAuthState = "aegis:oauth-state"

// Init initializes the OAuth plugin with Aegis services.
//
// This method is called during Aegis startup to inject dependencies and set up
// the state store. It retrieves services from the Aegis interface and derives
// the OAuth state secret from the master secret.
//
// Initialization Steps:
//  1. Get user, session, and account services from Aegis
//  2. Initialize OAuth store if not provided
//  3. Derive state secret from master secret ("aegis:oauth-state" purpose)
//  4. Create StateStore with derived secret for CSRF protection
//
// Parameters:
//   - ctx: Initialization context (currently unused)
//   - a: Aegis interface providing services and configuration
//
// Returns:
//   - error: Initialization error (currently always nil)
func (p *Plugin) Init(_ context.Context, a plugins.Aegis) error {
	// Get services from Aegis interface
	authService := a.GetAuthService()
	p.accountService = authService.Account
	p.userService = authService.User
	p.sessionService = authService.Session
	p.logger = a.GetLogger()

	// Initialize store if not provided
	if p.store == nil {
		p.store = NewDefaultOAuthStore(a.DB())
	}

	// Initialize OAuth state store with Aegis settings if not already done
	if p.stateStore == nil && a != nil {
		secret := a.DeriveSecret(SecretPurposeOAuthState)
		if len(secret) > 0 && p.sessionService != nil {
			cfg := p.sessionService.GetConfig()
			p.stateStore = NewStateStore(&StateStoreConfig{
				SessionConfig: cfg,
				Secret:        secret,
				MaxAge:        15 * 60, // 15 minutes for OAuth state
			})
		}
	}

	return nil
}

// MountRoutes registers HTTP routes for the OAuth plugin
func (p *Plugin) MountRoutes(router router.Router, prefix string) {
	handlers := NewHandlers(p)

	// OAuth authentication routes
	router.GET(prefix+"/oauth/:provider", handlers.BeginAuthHandler)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/oauth/{provider}",
		Summary:     "Begin OAuth flow",
		Description: "Initiate OAuth authentication with the specified provider (e.g., google, github)",
		Tags:        []string{"OAuth"},
		Protected:   false,
		Responses: map[string]*core.ResponseMeta{
			"302": {Description: "Redirect to OAuth provider", Schema: "Redirect"},
			"400": {Description: "Invalid or unsupported provider", Schema: core.SchemaError},
		},
	})

	router.GET(prefix+"/oauth/:provider/callback", handlers.CallbackHandler)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/oauth/{provider}/callback",
		Summary:     "OAuth callback",
		Description: "Handle OAuth provider callback and create session",
		Tags:        []string{"OAuth"},
		Protected:   false,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Authentication successful, session created", Schema: core.SchemaSession},
			"302": {Description: "Redirect after successful authentication", Schema: "Redirect"},
			"400": {Description: "Invalid callback or authorization failed", Schema: core.SchemaError},
		},
	})
}

// Dependencies returns external package dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{
		{
			Package: "github.com/markbates/goth",
			Version: "latest",
			Purpose: "OAuth provider integration (recommended, but optional via abstraction)",
		},
	}
}

// RequiresTables returns core tables this plugin depends on
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.session"}
}

// ProvidesAuthMethods returns authentication methods provided
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"oauth_google", "oauth_github", "oauth_apple", "oauth_custom"}
}

// GetMigrations returns the plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	migs, err := GetMigrations(p.dialect)
	if err != nil {
		return []plugins.Migration{}
	}
	return migs
}

// GetSchemas returns all schemas for all supported dialects
func (p *Plugin) GetSchemas() []plugins.Schema {
	dialects := []plugins.Dialect{plugins.DialectPostgres, plugins.DialectMySQL, plugins.DialectSQLite}
	schemas := make([]plugins.Schema, 0, len(dialects))

	for _, dialect := range dialects {
		schema, err := GetSchema(dialect)
		if err != nil {
			continue
		}
		schemas = append(schemas, *schema)
	}

	return schemas
}

// BeginAuth starts the OAuth authentication flow with CSRF protection.
//
// This method initiates the OAuth flow by:
//  1. Generating a cryptographically secure CSRF state token
//  2. Starting the OAuth session with the provider
//  3. Obtaining the provider's authorization URL
//  4. Storing state and session data in a signed cookie
//  5. Redirecting the user to the provider's authorization page
//
// OAuth Flow (Step 1-3):
//  1. User → GET /auth/oauth/google
//  2. Plugin → Generate state="abc123", store in cookie
//  3. Plugin → Redirect to https://accounts.google.com/authorize?state=abc123&...
//
// State Cookie:
// The state cookie contains:
//   - CSRF state token (random 32 bytes, base64-encoded)
//   - Provider name ("google", "github", etc.)
//   - Marshaled OAuth session data (for completing the flow)
//   - HMAC signature (prevents tampering)
//   - Expiration: 15 minutes (short-lived for security)
//
// Parameters:
//   - w: HTTP response writer for setting cookies and redirecting
//   - r: HTTP request (currently unused)
//   - providerName: Provider identifier ("google", "github", etc.)
//
// Returns:
//   - error: Provider not found, state generation failed, or redirect failed
//
// Security:
//   - State token is cryptographically random (32 bytes from crypto/rand)
//   - State cookie is HMAC-signed to prevent tampering
//   - Cookie uses Secure, HTTPOnly, SameSite settings from SessionConfig
func (p *Plugin) BeginAuth(w http.ResponseWriter, r *http.Request, providerName string) error {
	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return fmt.Errorf("provider %s not found: %w", providerName, err)
	}

	// Generate CSRF state
	state, err := p.stateStore.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Begin authentication with the provider
	sess, err := provider.BeginAuth(state)
	if err != nil {
		return fmt.Errorf("failed to begin auth: %w", err)
	}

	// Get the auth URL
	authURL, err := sess.GetAuthURL()
	if err != nil {
		return fmt.Errorf("failed to get auth URL: %w", err)
	}

	// Store state and session data in cookie
	stateData := &StateData{
		State:       state,
		Provider:    providerName,
		SessionData: sess.Marshal(),
	}
	if err := p.stateStore.StoreState(w, stateData); err != nil {
		return fmt.Errorf("failed to store state: %w", err)
	}

	// Redirect to provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	return nil
}

// CompleteAuth completes the OAuth authentication flow after provider callback.
//
// This method handles the OAuth callback by:
//  1. Validating the CSRF state token from the callback
//  2. Exchanging the authorization code for an access token
//  3. Fetching the user's profile from the provider
//  4. Creating or linking an Aegis user account
//  5. Creating an Aegis session for the authenticated user
//
// OAuth Flow (Step 4-9):
//  4. Provider → Redirect to /auth/oauth/google/callback?code=xyz&state=abc123
//  5. Plugin → Validate state=abc123 matches cookie
//  6. Plugin → Exchange code=xyz for access token
//  7. Plugin → Fetch user profile from provider
//  8. Plugin → Create/link user account in Aegis
//  9. Plugin → Create session and set cookie
//
// User Account Matching:
//   - Existing OAuth connection: Retrieve linked user
//   - New OAuth connection with known email: Link to existing user
//   - New OAuth connection with unknown email: Create new user
//   - OAuth without email: Create user without email (uses provider name)
//
// Parameters:
//   - ctx: Request context for database operations
//   - w: HTTP response writer for clearing state cookie
//   - r: HTTP request with OAuth callback parameters (code, state)
//
// Returns:
//   - *User: Authenticated user with OAuth data
//   - *auth.Session: New Aegis session for the user
//   - error: State validation, token exchange, or user creation error
//
// Security:
//   - State validation prevents CSRF attacks
//   - State cookie is cleared after validation (one-time use)
//   - Access tokens stored in database (not cookies)
func (p *Plugin) CompleteAuth(ctx context.Context, w http.ResponseWriter, r *http.Request) (*User, *auth.Session, error) {
	// Get provider name from query or path
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		// Try to get from stored state
		stateData, err := p.stateStore.GetState(r)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get provider: %w", err)
		}
		providerName = stateData.Provider
	}

	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return nil, nil, fmt.Errorf("provider %s not found: %w", providerName, err)
	}

	// Validate state from callback
	callbackState := r.URL.Query().Get("state")
	stateData, err := p.stateStore.ValidateState(r, callbackState)
	if err != nil {
		return nil, nil, fmt.Errorf("state validation failed: %w", err)
	}

	// Clear the state cookie
	p.stateStore.ClearState(w)

	// Unmarshal the stored session
	sess, err := provider.UnmarshalSession(stateData.SessionData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Get authorization params from callback
	params := r.URL.Query()
	if params.Encode() == "" && r.Method == "POST" {
		if err := r.ParseForm(); err == nil {
			params = r.Form
		}
	}

	// Authorize with the provider (exchange code for token)
	_, err = sess.Authorize(provider, params)
	if err != nil {
		return nil, nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Fetch user info from provider
	gothUser, err := provider.FetchUser(sess)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Convert Goth user to our abstraction
	oauthUser := GothUserToUser(gothUser)

	// Get or create Aegis user
	user, err := p.getOrCreateUser(ctx, gothUser.Provider, oauthUser)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	// Create session
	ipAddress := r.RemoteAddr
	userAgent := r.Header.Get("User-Agent")
	session, err := p.sessionService.CreateSession(ctx, &user.User, ipAddress, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return user, session, nil
}

// GetStateStore returns the OAuth state store
func (p *Plugin) GetStateStore() *StateStore {
	return p.stateStore
}

// getOrCreateUser retrieves an existing user or creates a new one from OAuth data.
//
// This method implements the account matching logic:
//  1. Check if OAuth connection exists → return linked user
//  2. Check if user with same email exists → link OAuth to existing user
//  3. No existing user → create new user and link OAuth
//
// Account Linking Strategy:
//   - Provider User ID: Primary key for OAuth connections (e.g., Google user ID)
//   - Email Matching: Users with same email are automatically linked
//   - No Email: Create user without email (provider name used as identifier)
//
// Database Operations:
//   - Query oauth_connections by (provider, provider_user_id)
//   - Query auth.users by email (if provided)
//   - Insert new user if not found
//   - Insert oauth_connection linking user to provider
//
// Parameters:
//   - ctx: Request context
//   - provider: Provider name ("google", "github", etc.)
//   - oauthUser: User data from OAuth provider
//
// Returns:
//   - *User: Aegis user (existing or newly created)
//   - error: Database query or insert error
func (p *Plugin) getOrCreateUser(ctx context.Context, provider string, oauthUser *User) (*User, error) {
	// Check if OAuth connection already exists
	connection, err := p.store.GetConnectionByProviderUserID(ctx, provider, oauthUser.ID)
	if err == nil && connection != nil {
		// Connection exists, retrieve the actual user from database
		user, err := p.userService.GetUserByID(ctx, connection.UserID)
		if err == nil {
			return &User{User: user}, nil
		}
		// If user not found, fall through to create new user
	}

	// No existing connection, check if user with this email exists
	var user auth.User
	if oauthUser.Email != "" {
		// Try to get existing user by email
		user, err = p.userService.GetUserByEmail(ctx, oauthUser.Email)
		if err != nil {
			// User doesn't exist, create new one with OAuth name and email
			user, err = p.userService.CreateUserWithoutPassword(ctx, auth.User{Name: oauthUser.Name, Email: oauthUser.Email})
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		}
		// User exists with this email, will link OAuth to existing account
	} else {
		// No email provided by OAuth, create user without email but with name
		user, err = p.userService.CreateUserWithoutPassword(ctx, auth.User{Name: oauthUser.Name})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Save OAuth connection linked to user
	conn := Connection{
		ID:             core.GenerateID(),
		UserID:         user.GetID(),
		Provider:       provider,
		ProviderUserID: oauthUser.ID,
		Email:          oauthUser.Email,
		Name:           oauthUser.Name,
		AvatarURL:      oauthUser.Avatar,
		AccessToken:    oauthUser.AccessToken,
		RefreshToken:   oauthUser.RefreshToken,
		ExpiresAt:      oauthUser.ExpiresAt,
		ProviderData:   oauthUser.ProviderData,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = p.store.CreateConnection(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to save OAuth connection: %w", err)
	}

	return &User{User: user}, nil
}

// LinkAccount links an OAuth provider to an existing authenticated user account.
//
// This method allows users to add additional OAuth providers to their account.
// For example, a user who signed up with email/password can later link their
// Google account for easier login.
//
// Use Cases:
//   - Link Google to existing email/password account
//   - Link multiple providers to one account (Google + GitHub + Apple)
//   - Re-link provider after unlinking
//
// Parameters:
//   - ctx: Request context
//   - userID: Aegis user ID to link provider to
//   - oauthUser: OAuth user data from provider
//   - provider: Provider name ("google", "github", etc.)
//
// Returns:
//   - error: Database error or duplicate connection error
//
// Example:
//
//	// User already authenticated via session
//	user, _ := core.GetUser(r.Context())
//	err := plugin.LinkAccount(ctx, user.ID, oauthUser, "google")
func (p *Plugin) LinkAccount(ctx context.Context, userID string, oauthUser *User, provider string) error {
	conn := Connection{
		ID:             core.GenerateID(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: oauthUser.ID,
		Email:          oauthUser.Email,
		Name:           oauthUser.Name,
		AvatarURL:      oauthUser.Avatar,
		AccessToken:    oauthUser.AccessToken,
		RefreshToken:   oauthUser.RefreshToken,
		ExpiresAt:      oauthUser.ExpiresAt,
		ProviderData:   oauthUser.ProviderData,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := p.store.CreateConnection(ctx, conn)
	return err
}

// GetUserConnections retrieves all OAuth provider connections for a user.
//
// This method returns all linked OAuth providers, including access tokens,
// refresh tokens, and provider-specific data. Useful for displaying linked
// accounts in user settings or managing provider connections.
//
// Parameters:
//   - ctx: Request context
//   - userID: Aegis user ID
//
// Returns:
//   - []*Connection: List of OAuth connections (may be empty)
//   - error: Database query error
//
// Example:
//
//	connections, _ := plugin.GetUserConnections(ctx, user.ID)
//	for _, conn := range connections {
//	    fmt.Printf("Linked: %s (%s)\n", conn.Provider, conn.Email)
//	}
func (p *Plugin) GetUserConnections(ctx context.Context, userID string) ([]*Connection, error) {
	conns, err := p.store.GetConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*Connection, len(conns))
	for i, conn := range conns {
		result[i] = &conn
	}
	return result, nil
}

// UnlinkAccount removes an OAuth provider link from a user account.
//
// This method allows users to disconnect OAuth providers from their account.
// The user's Aegis account remains active, but they can no longer sign in
// using the unlinked provider.
//
// Safety:
// This method does NOT check if the user has other authentication methods.
// You should verify the user has email/password or other OAuth providers
// before allowing unlinking to prevent account lockout.
//
// Parameters:
//   - ctx: Request context
//   - userID: Aegis user ID
//   - provider: Provider name to unlink ("google", "github", etc.)
//
// Returns:
//   - error: Database error or connection not found
//
// Example:
//
//	// Unlink Google from user's account
//	err := plugin.UnlinkAccount(ctx, user.ID, "google")
func (p *Plugin) UnlinkAccount(ctx context.Context, userID, provider string) error {
	return p.store.DeleteConnection(ctx, provider, userID)
}
