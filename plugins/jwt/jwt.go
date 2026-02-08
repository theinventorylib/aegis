// Package jwt implements JWT (JSON Web Token) authentication for Aegis.
//
// This plugin provides stateless token-based authentication using industry-standard
// JWT tokens with RSA signing. Features include:
//
//   - Token Generation: Access tokens (15min) and refresh tokens (7d)
//   - RSA Signing: RS256 algorithm with database-backed key storage
//   - Key Rotation: Automatic key rotation with configurable intervals
//   - JWKS Endpoint: Public key discovery for token verification
//   - Token Blacklist: Redis-backed token revocation
//   - Refresh Flow: Secure token refresh without re-authentication
//
// JWT vs Session Tokens:
//
//	Session Tokens (core):
//	  - Stateful: Requires database lookup on every request
//	  - Revocable: Can be deleted from database immediately
//	  - Simple: Just random strings, no cryptography
//
//	JWT Tokens (this plugin):
//	  - Stateless: Can be verified without database (using public key)
//	  - Self-contained: Includes user ID and expiry in token
//	  - Distributed: Multiple services can verify without shared database
//	  - Revocation: Requires blacklist (Redis) for immediate revocation
//
// Architecture:
//
//	Token Pair:
//	  - Access Token: Short-lived (15min), used for API requests
//	  - Refresh Token: Long-lived (7d), used to get new access tokens
//
//	Key Storage:
//	  - Private keys: Stored in database, used for signing
//	  - Public keys: Exposed via /jwt/.well-known/jwks.json (JWKS endpoint)
//	  - Key rotation: Old keys retained for token verification
//
//	Token Flow:
//	  1. User authenticates (email/password, OAuth, etc.)
//	  2. POST /jwt/token → Get access + refresh token
//	  3. Use access token in Authorization header
//	  4. When access token expires, POST /jwt/refreshToken with refresh token
//	  5. Get new access + refresh token pair
//
// Use Cases:
//   - Microservices: Stateless authentication across services
//   - Mobile apps: Long-lived refresh tokens
//   - SPAs: JavaScript apps with token storage
//   - Third-party integrations: Standard JWT format
//
// Security Considerations:
//   - Access tokens are short-lived (minimize exposure window)
//   - Refresh tokens are long-lived (balance UX vs security)
//   - Token blacklist requires Redis (for logout/revocation)
//   - HTTPS is REQUIRED (tokens are bearer credentials)
//   - Store tokens securely (httpOnly cookies, secure storage, not localStorage)
//
// Example:
//
//	package main
//
//	import (
//		"context"
//		"github.com/theinventorylib/aegis"
//		"github.com/theinventorylib/aegis/plugins/jwt"
//	)
//
//	func main() {
//		a, _ := aegis.New(context.Background(), ...)
//
//		// Configure JWT plugin
//		jwtConfig := &jwt.Config{
//			Issuer:              "myapp",
//			AccessTokenExpiry:   15 * time.Minute,
//			RefreshTokenExpiry:  7 * 24 * time.Hour,
//			KeyRotationInterval: 24 * time.Hour,
//		}
//
//		// Register JWT plugin
//		a.Use(context.Background(), jwt.New(jwtConfig, nil, plugins.DialectPostgres))
//
//		a.MountRoutes("/auth")
//		// Routes available:
//		// POST /auth/jwt/token
//		// POST /auth/jwt/getAccessToken
//		// POST /auth/jwt/refreshToken
//		// POST /auth/jwt/logout
//		// GET  /auth/jwt/.well-known/jwks.json
//	}
package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

const (
	// TokenTypeAccess identifies an access token in JWT claims.
	// Access tokens are short-lived and used for API requests.
	TokenTypeAccess = "access"

	// TokenTypeRefresh identifies a refresh token in JWT claims.
	// Refresh tokens are long-lived and used to obtain new access tokens.
	TokenTypeRefresh = "refresh"
)

// Config holds JWT plugin configuration.
//
// All durations should be balanced for security vs user experience:
//   - Shorter access tokens: More secure (tokens expire quickly)
//   - Longer refresh tokens: Better UX (less frequent re-authentication)
//   - Frequent key rotation: More secure (limits key exposure)
//   - Longer key retention: Required for old token validation
type Config struct {
	// Issuer is the JWT "iss" claim identifying the token issuer.
	// Should be your application name or domain (e.g., "myapp.com").
	Issuer string

	// AccessTokenExpiry is how long access tokens remain valid.
	// Recommended: 15 minutes to 1 hour
	// Shorter = more secure, longer = fewer refresh requests
	AccessTokenExpiry time.Duration

	// RefreshTokenExpiry is how long refresh tokens remain valid.
	// Recommended: 7 days to 30 days
	// Shorter = more secure, longer = better UX (less frequent logins)
	RefreshTokenExpiry time.Duration

	// KeyRotationInterval is how often to rotate signing keys.
	// Recommended: 24 hours to 7 days
	// More frequent rotation limits the impact of key compromise.
	// Set to 0 to disable automatic rotation (manual rotation only).
	KeyRotationInterval time.Duration

	// KeySize is the RSA key size in bits (2048, 3072, or 4096).
	// Only used for RSA algorithm.
	// Recommended: 2048 (good balance of security and performance)
	// 4096 provides higher security but slower signing/verification.
	KeySize int

	// KeyAlgorithm indicates the signing algorithm.
	// Currently supported: "RSA" (uses RS256)
	// Future: ECDSA (ES256), HMAC (HS256) for symmetric keys
	KeyAlgorithm string

	// KeyRetention is how long to keep rotated keys in storage.
	// MUST be greater than the maximum token lifetime (RefreshTokenExpiry)
	// to ensure old tokens can still be verified.
	// Recommended: 30 days (covers refresh token lifetime + buffer)
	KeyRetention time.Duration
}

// DefaultConfig returns production-ready JWT configuration with security best practices.
//
// Default values:
//   - Issuer: "aegis"
//   - Access token: 15 minutes (short-lived for security)
//   - Refresh token: 7 days (balance between UX and security)
//   - Key rotation: 24 hours (daily rotation limits key exposure)
//   - Key size: 2048 bits (industry standard RSA key size)
//   - Algorithm: RSA/RS256 (asymmetric signing)
//   - Key retention: 30 days (covers refresh token lifetime)
//
// Customize for your use case:
//
//	config := jwt.DefaultConfig()
//	config.Issuer = "myapp.com"
//	config.AccessTokenExpiry = 1 * time.Hour  // Longer for internal APIs
//	config.RefreshTokenExpiry = 30 * 24 * time.Hour  // 30 days for mobile apps
func DefaultConfig() *Config {
	return &Config{
		Issuer:              "aegis",
		AccessTokenExpiry:   15 * time.Minute,
		RefreshTokenExpiry:  7 * 24 * time.Hour,
		KeyRotationInterval: 24 * time.Hour,
		KeySize:             2048,
		KeyAlgorithm:        "RSA",
		// Keep keys for 30 days by default which should be > refresh token lifetime
		KeyRetention: 30 * 24 * time.Hour,
	}
}

// Plugin represents the JWT authentication plugin.
//
// This plugin manages the complete JWT token lifecycle:
//   - Token generation (access + refresh pairs)
//   - Token validation (signature + expiry)
//   - Token refresh (new tokens from refresh token)
//   - Key rotation (periodic key generation)
//   - Key storage (database-backed JWKS)
//   - Token blacklist (Redis-backed revocation)
//
// Architecture:
//
//	Components:
//	  - store: Database storage for JWK keys
//	  - handler: HTTP handlers for token endpoints
//	  - config: Token expiry and key rotation settings
//	  - redisClient: Token blacklist storage
//	  - sessionService: Integration with core authentication
//
//	Keys:
//	  - accessTokenPrivateKey: Signs access tokens
//	  - accessTokenPublicKey: Verifies access tokens (exposed in JWKS)
//	  - refreshTokenPrivateKey: Signs refresh tokens
//	  - refreshTokenPublicKey: Verifies refresh tokens (not exposed)
//
//	Token Lifecycle:
//	  1. Generate: CreateTokenPair() → JWT signed with private key
//	  2. Use: Client includes token in Authorization header
//	  3. Verify: ValidateToken() → Check signature with public key
//	  4. Refresh: RefreshTokens() → Generate new pair from refresh token
//	  5. Revoke: BlacklistToken() → Add to Redis blacklist
type Plugin struct {
	// store provides database operations for JWK key storage
	store Store

	// handler manages HTTP endpoints for token operations
	handler *Handler

	// config holds token expiry and key rotation settings
	config *Config

	// dialect specifies the database dialect (postgres, mysql, sqlite)
	dialect plugins.Dialect

	// accessTokenPrivateKey signs access tokens (kept secret)
	accessTokenPrivateKey jwk.Key

	// accessTokenPublicKey verifies access tokens (exposed in JWKS endpoint)
	accessTokenPublicKey jwk.Key

	// refreshTokenPrivateKey signs refresh tokens (kept secret)
	refreshTokenPrivateKey jwk.Key

	// refreshTokenPublicKey verifies refresh tokens (not exposed publicly)
	refreshTokenPublicKey jwk.Key

	// redisClient provides token blacklist storage for revocation
	redisClient *redis.Client

	// sessionService integrates with core authentication middleware
	sessionService *core.SessionService

	// logger provides structured logging (may be nil)
	logger config.Logger
	// aegis is the main framework instance
	aegis plugins.Aegis
}

// New creates a new JWT authentication plugin.
//
// Parameters:
//   - config: JWT configuration (token expiry, key rotation, etc.)
//     Pass nil to use DefaultConfig()
//   - store: Custom JWK storage implementation
//     Pass nil to use default SQL storage (recommended)
//   - dialect: Database dialect (postgres, mysql, sqlite)
//     Optional, defaults to postgres if not provided
//
// Example:
//
//	// Use default configuration
//	jwtPlugin := jwt.New(nil, nil, plugins.DialectPostgres)
//
//	// Custom configuration
//	config := &jwt.Config{
//		Issuer:              "myapp.com",
//		AccessTokenExpiry:   1 * time.Hour,
//		RefreshTokenExpiry:  30 * 24 * time.Hour,
//		KeyRotationInterval: 7 * 24 * time.Hour,
//	}
//	jwtPlugin := jwt.New(config, nil, plugins.DialectPostgres)
func New(config *Config, store Store, dialect ...plugins.Dialect) *Plugin {
	if config == nil {
		config = DefaultConfig()
	}

	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}

	return &Plugin{
		store:   store,
		config:  config,
		dialect: d,
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "jwt"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "JWT authentication plugin for token-based authentication with database-backed key storage and JWKS endpoint"
}

// Init initializes the JWT plugin and integrates with core authentication.
//
// Initialization steps:
//  1. Initialize JWK storage (default SQL store if not provided)
//  2. Get Redis client from SessionService (for token blacklist)
//  3. Validate database schema (ensure jwk_keys table exists)
//  4. Initialize signing keys (load from DB or generate new)
//  5. Create HTTP handler for token endpoints
//  6. Start automatic key rotation (if configured)
//
// Database Schema Validation:
//
// The plugin validates that the jwk_keys table exists with required columns:
//   - kid (key ID, primary key)
//   - key_data (JWK JSON)
//   - algorithm (RS256, etc.)
//   - use (sig for signing, enc for encryption)
//   - created_at, expires_at
//
// Key Initialization:
//
// On first run, generates RSA key pairs for:
//   - Access tokens (public key exposed in JWKS)
//   - Refresh tokens (public key NOT exposed)
//
// On subsequent runs, loads existing keys from database.
//
// Parameters:
//   - ctx: Context for initialization (can be canceled)
//   - aegis: Framework instance providing database, services, etc.
//
// Returns:
//   - error: Schema validation errors, key generation errors
func (p *Plugin) Init(ctx context.Context, aegis plugins.Aegis) error {
	// Get logger from aegis instance
	p.logger = aegis.GetLogger()

	// Initialize store if not provided
	if p.store == nil {
		p.store = NewDefaultJWTStore(aegis.DB())
	}

	// Get session service to get Redis client (for token blacklisting only)
	sessionService := aegis.GetAuthService().Session
	p.sessionService = sessionService // Store for middleware access
	p.redisClient = sessionService.GetRedisClient()
	p.aegis = aegis

	// Build schema requirements: basic table existence from RequiresTables + detailed checks
	tables := p.RequiresTables()
	requirements := make([]plugins.SchemaRequirement, 0, len(tables))
	for _, table := range tables {
		requirements = append(requirements, plugins.ValidateTableExists(table))
	}
	requirements = append(requirements, GetSchemaRequirements(p.dialect)...)

	// Validate JWT plugin schema requirements
	if err := aegis.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return err
	}

	// Initialize keys (get or create from database)
	if err := p.initializeKeys(ctx); err != nil {
		return fmt.Errorf("failed to initialize keys: %w", err)
	}

	// Create handler for JWKS endpoint
	p.handler = NewHandler(p)

	// Schema registration moved to MountRoutes to ensure all plugins are initialized
	// and OpenAPI plugin is ready to receive schemas.

	// Auto-start key rotation if interval is configured
	if p.config.KeyRotationInterval > 0 {
		p.StartKeyRotation(ctx)
	}

	return nil
}

// MountRoutes registers HTTP routes for JWT endpoints with appropriate middleware.
func (p *Plugin) MountRoutes(router router.Router, basePath string) {
	// Register schemas with OpenAPI if available
	if plugin, ok := p.aegis.GetPlugin("openapi"); ok {
		if oapi, ok := plugin.(interface {
			RegisterSchemaFromType(name string, example any)
		}); ok {
			// Register request schemas
			oapi.RegisterSchemaFromType(SchemaRefreshTokenRequest, map[string]string{"refresh_token": ""})

			// Register response schemas
			oapi.RegisterSchemaFromType(SchemaTokenPair, TokenPair{})
			oapi.RegisterSchemaFromType(SchemaAccessToken, AccessToken{})
			oapi.RegisterSchemaFromType(SchemaJWKS, JWKS{})
		}
	}

	// Create route group for JWT plugin
	jwtGroup := router.Group(basePath, "JWT")

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected endpoints - require active session/cookie authentication
	jwtGroup.POST("/token", requireAuth(http.HandlerFunc(p.handler.handleGetToken)).ServeHTTP)
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        basePath + "/token",
		Summary:     "Generate JWT token pair",
		Description: "Generate access and refresh JWT tokens for the authenticated user",
		Tags:        []string{"JWT"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Token pair generated successfully", Schema: SchemaTokenPair},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"500": {Description: "Failed to generate tokens", Schema: core.SchemaError},
		},
	})

	jwtGroup.POST("/getAccessToken", requireAuth(http.HandlerFunc(p.handler.handleGetAccessToken)).ServeHTTP)
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        basePath + "/getAccessToken",
		Summary:     "Get access token",
		Description: "Generate a new access token for the authenticated user",
		Tags:        []string{"JWT"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Access token generated successfully", Schema: SchemaAccessToken},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"500": {Description: "Failed to generate token", Schema: core.SchemaError},
		},
	})

	jwtGroup.POST("/logout", requireAuth(http.HandlerFunc(p.handler.handleLogout)).ServeHTTP)
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        basePath + "/logout",
		Summary:     "Logout and blacklist tokens",
		Description: "Logout the user and blacklist their JWT tokens (requires Redis)",
		Tags:        []string{"JWT"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Successfully logged out and tokens blacklisted", Schema: core.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	// Public endpoints - no authentication required
	jwtGroup.POST("/refreshToken", p.handler.handleRefreshToken) // Refresh token is its own auth
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        basePath + "/refreshToken",
		Summary:     "Refresh JWT tokens",
		Description: "Use a refresh token to obtain new access and refresh tokens",
		Tags:        []string{"JWT"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Refresh token",
			Required:    true,
			Schema:      core.SchemaRefreshTokenRequest,
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Tokens refreshed successfully", Schema: SchemaTokenPair},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Invalid or expired refresh token", Schema: core.SchemaError},
		},
	})

	// JWKS endpoint is slightly special (public discovery)
	jwtGroup.GET("/.well-known/jwks.json", p.handler.handleJWKS)
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        "/.well-known/jwks.json",
		Summary:     "Get JWKS",
		Description: "Retrieve the JSON Web Key Set for JWT verification",
		Tags:        []string{"JWT"},
		Protected:   false,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "JWKS retrieved successfully", Schema: SchemaJWKS},
			"500": {Description: "Failed to retrieve JWKS", Schema: core.SchemaError},
		},
	})

	jwtGroup.GET("/jwks", p.handler.handleJWKS) // Convenience endpoint
	jwtGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        basePath + "/jwks",
		Summary:     "Get JWKS (convenience endpoint)",
		Description: "Retrieve the JSON Web Key Set for JWT verification",
		Tags:        []string{"JWT"},
		Protected:   false,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "JWKS retrieved successfully", Schema: SchemaJWKS},
			"500": {Description: "Failed to retrieve JWKS", Schema: core.SchemaError},
		},
	})
}

// Dependencies returns external package dependencies.
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{
		{
			Package: "github.com/lestrrat-go/jwx/v3",
			Version: "v3.x",
			Purpose: "JWT token generation and validation",
		},
	}
}

// RequiresTables returns tables this plugin manages.
func (p *Plugin) RequiresTables() []string {
	return []string{"jwks"} // JWT plugin owns and manages its JWKS table
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"jwt"}
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
	dialects := []plugins.Dialect{plugins.DialectPostgres, plugins.DialectMySQL}
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

// initializeKeys loads or creates JWT signing keys from the database
func (p *Plugin) initializeKeys(ctx context.Context) error {
	// Get or create access keys
	accessPrivKey, accessPubKey, err := p.getOrCreateKeyPair(ctx, "access")
	if err != nil {
		return fmt.Errorf("failed to get access keys: %w", err)
	}

	// Get or create refresh keys
	refreshPrivKey, refreshPubKey, err := p.getOrCreateKeyPair(ctx, "refresh")
	if err != nil {
		return fmt.Errorf("failed to get refresh keys: %w", err)
	}

	p.accessTokenPrivateKey = accessPrivKey
	p.accessTokenPublicKey = accessPubKey
	p.refreshTokenPrivateKey = refreshPrivKey
	p.refreshTokenPublicKey = refreshPubKey

	return nil
}

// getOrCreateKeyPair retrieves or creates a key pair for the given type
func (p *Plugin) getOrCreateKeyPair(ctx context.Context, keyType string) (jwk.Key, jwk.Key, error) {
	// Try to get existing key from database
	key, err := p.store.GetCurrentJWK(ctx, "RS256", "sig")
	if err == nil {
		pubKey, err := key.PublicKey()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get public key: %w", err)
		}
		return key, pubKey, nil
	}

	// Generate new key if none exists
	return p.rotateKeyPair(ctx, keyType)
}

// rotateKeyPair generates a new key pair and stores it in the database
func (p *Plugin) rotateKeyPair(ctx context.Context, keyType string) (jwk.Key, jwk.Key, error) {
	// Generate new RSA key
	keySize := 2048
	if p.config != nil && p.config.KeySize > 0 {
		keySize = p.config.KeySize
	}
	privKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Import to JWK
	key, err := jwk.Import(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create JWK: %w", err)
	}

	// Set key metadata
	keyID := fmt.Sprintf("%s-%d", keyType, time.Now().Unix())
	if err := key.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, nil, fmt.Errorf("failed to set key ID: %w", err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		return nil, nil, fmt.Errorf("failed to set algorithm: %w", err)
	}

	// Get public key
	pubKey, err := key.PublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Store in database with configured retention
	retention := 14 * 24 * time.Hour
	if p.config != nil && p.config.KeyRetention > 0 {
		retention = p.config.KeyRetention
	}
	expiresAt := time.Now().Add(retention)
	if err := p.store.StoreJWK(ctx, key, "RS256", "sig", &expiresAt); err != nil {
		return nil, nil, fmt.Errorf("failed to store JWK: %w", err)
	}

	return key, pubKey, nil
}

// GenerateTokenPair creates access and refresh tokens for a user
func (p *Plugin) GenerateTokenPair(userID string) (*TokenPair, error) {
	// Access Token
	accessToken, accessExpiry, err := p.generateToken(userID, TokenTypeAccess, p.config.AccessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Refresh Token
	refreshToken, refreshExpiry, err := p.generateToken(userID, TokenTypeRefresh, p.config.RefreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:   accessToken,
		AccessExpiry:  accessExpiry,
		RefreshToken:  refreshToken,
		RefreshExpiry: refreshExpiry,
	}, nil
}

// generateToken creates a signed JWT token
func (p *Plugin) generateToken(userID, tokenType string, duration time.Duration) (string, time.Time, error) {
	// Determine which key to use based on token type
	var privateKey jwk.Key
	switch tokenType {
	case TokenTypeAccess:
		privateKey = p.accessTokenPrivateKey
	case TokenTypeRefresh:
		privateKey = p.refreshTokenPrivateKey
	default:
		return "", time.Time{}, fmt.Errorf("invalid token type: %s", tokenType)
	}

	expiration := time.Now().Add(duration)

	// Create token with claims (including unique JTI)
	// Generate a unique token ID (JTI)
	jti := core.GenerateID()

	token, err := jwt.NewBuilder().
		JwtID(jti). // Add unique token ID
		Claim("user_id", userID).
		Claim("token_type", tokenType).
		Issuer(p.config.Issuer).
		Subject(userID).
		Expiration(expiration).
		NotBefore(time.Now()).
		IssuedAt(time.Now()).
		Build()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to build token: %w", err)
	}

	// Sign the token
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateKey))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), expiration, nil
}

// ValidateToken checks the validity of a token.
func (p *Plugin) ValidateToken(tokenStr string) (*Claims, error) {
	ctx := context.Background()

	// Check if specific token is blacklisted (if Redis is available)
	if p.redisClient != nil {
		redisKey := fmt.Sprintf("auth:blacklist:%s", tokenStr)
		exists, err := p.redisClient.Exists(ctx, redisKey).Result()
		if err == nil && exists > 0 {
			return nil, errors.New("token has been revoked")
		}
	}

	// First parse the token WITHOUT verification. IMPORTANT: do NOT enable
	// claim validation here because the token signature has not been verified
	// yet. Validating claims on an unverified token can be manipulated by an
	// attacker. We only extract the (untrusted) subject for optional checks
	// such as looking up a user blacklist; any security decisions must be
	// performed after the verified parse below.
	parsedToken, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithVerify(false),
		jwt.WithValidate(false),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract the user id to check for user blacklist
	tokenSubject, ok := parsedToken.Subject()
	if ok && p.redisClient != nil {
		// Check if user is blacklisted
		redisKey := fmt.Sprintf("auth:user_blacklist:%s", tokenSubject)
		exists, err := p.redisClient.Exists(ctx, redisKey).Result()
		if err == nil && exists > 0 {
			return nil, errors.New("user has been logged out from all sessions")
		}
	}

	// Parse and verify the token with public key
	token, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithKey(jwa.RS256(), p.accessTokenPublicKey),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Extract claims
	tokenSubject, ok = token.Subject()
	if !ok {
		return nil, errors.New("missing subject")
	}

	var claimTokenType string
	err = token.Get("token_type", &claimTokenType)
	if err != nil {
		return nil, errors.New("missing token type")
	}

	claims := &Claims{
		UserID:    tokenSubject,
		TokenType: claimTokenType,
	}

	// Additional validation
	if claims.TokenType != TokenTypeAccess {
		return nil, errors.New("token type mismatch")
	}

	return claims, nil
}

// RefreshTokens handles token refresh mechanism.
func (p *Plugin) RefreshTokens(refreshToken string) (*TokenPair, error) {
	ctx := context.Background()

	// Validate the refresh token first
	token, err := jwt.Parse(
		[]byte(refreshToken),
		jwt.WithKey(jwa.RS256(), p.refreshTokenPublicKey),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Extract user ID
	tokenSubject, ok := token.Subject()
	if !ok {
		return nil, errors.New("missing subject in refresh token")
	}

	// Immediately blacklist the used refresh token to prevent reuse (if Redis available)
	if p.redisClient != nil {
		err := p.blacklistTokenWithContext(ctx, refreshToken)
		_ = err // Ignore error, continue to ensure user gets new tokens
	}

	// Generate new token pair
	return p.GenerateTokenPair(tokenSubject)
}

// blacklistTokenWithContext is an internal helper that uses a specific context
func (p *Plugin) blacklistTokenWithContext(ctx context.Context, tokenStr string) error {
	// Only works if Redis is configured
	if p.redisClient == nil {
		return fmt.Errorf("redis not configured")
	}

	// Handle Bearer prefix if present
	parts := strings.Split(tokenStr, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		tokenStr = parts[1]
	}

	// Parse token to get expiration time
	token, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithKey(jwa.RS256(), p.refreshTokenPublicKey),
		jwt.WithValidate(true),
	)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// Get token expiration time
	exp, ok := token.Expiration()
	if !ok {
		// If no expiration, blacklist for a default period
		exp = time.Now().Add(24 * time.Hour)
	}

	// Calculate TTL for the blacklist entry
	ttl := time.Until(exp)
	if ttl <= 0 {
		// Token is already expired, no need to blacklist
		return nil
	}

	// Add to blacklist with expiration matching token's expiry
	redisKey := fmt.Sprintf("auth:blacklist:%s", tokenStr)
	return p.redisClient.Set(ctx, redisKey, "1", ttl).Err()
}

// Logout is a programmatic alias for BlacklistToken.
func (p *Plugin) Logout(tokenStr string) error {
	return p.BlacklistToken(tokenStr)
}

// BlacklistToken adds a token to the revocation list.
func (p *Plugin) BlacklistToken(tokenStr string) error {
	return p.blacklistTokenWithContext(context.Background(), tokenStr)
}

// LogoutAllSessions invalidates all tokens for a user.
func (p *Plugin) LogoutAllSessions(userID string) error {
	// Only works if Redis is configured
	if p.redisClient == nil {
		return errors.New("redis is required for user-wide logout")
	}

	ctx := context.Background()

	// Add user ID to a blacklist set with a long expiration
	redisKey := fmt.Sprintf("auth:user_blacklist:%s", userID)

	// Set for 30 days or longer depending on security needs
	return p.redisClient.Set(ctx, redisKey, "1", 30*24*time.Hour).Err()
}

// StartKeyRotation begins periodic key rotation (only if Redis is configured).
func (p *Plugin) StartKeyRotation(ctx context.Context) {
	if p.config.KeyRotationInterval == 0 {
		return
	}

	ticker := time.NewTicker(p.config.KeyRotationInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Use background context for rotation since ctx might be canceled
				if err := p.RotateKeys(context.Background()); err != nil {
					// Log error but continue - key rotation failure doesn't break existing tokens
					if p.logger != nil {
						p.logger.Error("JWT key rotation failed", "error", err)
					}
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// RotateKeys rotates both access and refresh keys
func (p *Plugin) RotateKeys(ctx context.Context) error {
	// Rotate access keys
	newAccessPriv, newAccessPub, err := p.rotateKeyPair(ctx, "access")
	if err != nil {
		return fmt.Errorf("failed to rotate access keys: %w", err)
	}

	// Rotate refresh keys
	newRefreshPriv, newRefreshPub, err := p.rotateKeyPair(ctx, "refresh")
	if err != nil {
		return fmt.Errorf("failed to rotate refresh keys: %w", err)
	}

	// Update active keys
	p.accessTokenPrivateKey = newAccessPriv
	p.accessTokenPublicKey = newAccessPub
	p.refreshTokenPrivateKey = newRefreshPriv
	p.refreshTokenPublicKey = newRefreshPub

	return nil
}

// CleanupExpiredKeys removes expired keys from the database.
func (p *Plugin) CleanupExpiredKeys(ctx context.Context) error {
	return p.store.DeleteExpiredJWKS(ctx)
}

// Ensure Plugin implements Plugin
var _ plugins.Plugin = (*Plugin)(nil)
