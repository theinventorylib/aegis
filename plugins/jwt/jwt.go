// Package jwt implements JWT authentication plugin for Aegis.
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
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

const (
	// TokenTypeAccess represents an access token type.
	TokenTypeAccess = "access"
	// TokenTypeRefresh represents a refresh token type.
	TokenTypeRefresh = "refresh"
)

// TokenPair represents a set of access and refresh tokens.
type TokenPair struct {
	AccessToken   string    `json:"access_token"`
	AccessExpiry  time.Time `json:"access_expiry"`
	RefreshToken  string    `json:"refresh_token"`
	RefreshExpiry time.Time `json:"refresh_expiry"`
}

// TokenClaims defines the structure of our JWT claims.
type TokenClaims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
}

// Config holds JWT-specific configuration.
type Config struct {
	DB                  db.Provider   // Database provider for key storage
	Secret              []byte        // Secret for key derivation (optional)
	Issuer              string        // Token issuer
	AccessTokenExpiry   time.Duration // Access token expiry duration
	RefreshTokenExpiry  time.Duration // Refresh token expiry duration
	KeyRotationInterval time.Duration // Key rotation interval
}

// DefaultConfig returns default JWT configuration.
func DefaultConfig() *Config {
	return &Config{
		Issuer:              "aegis",
		AccessTokenExpiry:   15 * time.Minute,
		RefreshTokenExpiry:  7 * 24 * time.Hour,
		KeyRotationInterval: 24 * time.Hour,
	}
}

// Plugin represents the JWT authentication plugin.
type Plugin struct {
	db                     *DB
	handler                *Handler
	config                 *Config
	accessTokenPrivateKey  jwk.Key
	accessTokenPublicKey   jwk.Key
	refreshTokenPrivateKey jwk.Key
	refreshTokenPublicKey  jwk.Key
	redisClient            *redis.Client
	sessionService         *core.SessionService // Added for middleware access
}

// New creates a new JWT plugin.
func New(config *Config) *Plugin {
	if config == nil {
		config = DefaultConfig()
	}

	return &Plugin{
		config: config,
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

// Init initializes the plugin and integrates with SessionService.
func (p *Plugin) Init(ctx context.Context, aegis plugins.Aegis) error {
	// Ensure DB provider is configured
	if p.config.DB == nil {
		return fmt.Errorf("JWT plugin requires a database provider in config")
	}

	// Create database handler
	p.db = NewDB(p.config.DB)

	// Get session service to get Redis client (for token blacklisting only)
	sessionService := aegis.GetSessionService()
	p.sessionService = sessionService // Store for middleware access
	p.redisClient = sessionService.GetRedisClient()

	// Initialize keys (get or create from database)
	if err := p.initializeKeys(ctx); err != nil {
		return fmt.Errorf("failed to initialize keys: %w", err)
	}

	// Create handler for JWKS endpoint
	p.handler = NewHandler(p)

	// Register as token provider with session service
	sessionService.SetTokenProvider(p)

	return nil
}

// MountRoutes registers HTTP routes for JWT endpoints with appropriate middleware.
func (p *Plugin) MountRoutes(router server.Router, basePath string) {
	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected endpoints - require active session/cookie authentication
	router.POST(basePath+"/token", requireAuth(http.HandlerFunc(p.handler.HandleGetToken)).ServeHTTP)
	router.POST(basePath+"/getAccessToken", requireAuth(http.HandlerFunc(p.handler.HandleGetAccessToken)).ServeHTTP)
	router.POST(basePath+"/logout", requireAuth(http.HandlerFunc(p.handler.HandleLogout)).ServeHTTP)

	// Public endpoints - no authentication required
	router.POST(basePath+"/refreshToken", p.handler.HandleRefreshToken) // Refresh token is the auth
	router.GET("/.well-known/jwks.json", p.handler.HandleJWKS)          // Public key discovery
	router.GET(basePath+"/jwks", p.handler.HandleJWKS)                  // Convenience endpoint
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
	return []string{"jwt.jwks"} // JWT plugin owns and manages its JWKS table
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"jwt"}
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
	key, err := p.db.GetCurrentJWK(ctx, "RS256", "sig")
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
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
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

	// Store in database with 14-day expiry
	expiresAt := time.Now().Add(14 * 24 * time.Hour)
	if err := p.db.CreateJWK(ctx, key, "RS256", "sig", &expiresAt); err != nil {
		return nil, nil, fmt.Errorf("failed to store JWK: %w", err)
	}

	return key, pubKey, nil
}

// GenerateTokenPair creates access and refresh tokens for a user
func (p *Plugin) GenerateTokenPair(userID string) (*core.TokenPair, error) {
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

	return &core.TokenPair{
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
	token, err := jwt.NewBuilder().
		JwtID(core.GenerateSessionID()). // Add unique token ID
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
func (p *Plugin) ValidateToken(tokenStr string) (*core.TokenClaims, error) {
	ctx := context.Background()

	// Check if specific token is blacklisted (if Redis is available)
	if p.redisClient != nil {
		redisKey := fmt.Sprintf("auth:blacklist:%s", tokenStr)
		exists, err := p.redisClient.Exists(ctx, redisKey).Result()
		if err == nil && exists > 0 {
			return nil, errors.New("token has been revoked")
		}
	}

	// First parse the token without validation to get the subject (userID)
	parsedToken, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithVerify(false),
		jwt.WithValidate(true),
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

	claims := &core.TokenClaims{
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
func (p *Plugin) RefreshTokens(refreshToken string) (*core.TokenPair, error) {
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
		_ = p.blacklistTokenWithContext(ctx, refreshToken) // Ignore error, continue to ensure user gets new tokens
	}

	// Generate new token pair
	return p.GenerateTokenPair(tokenSubject)
}

// blacklistTokenWithContext is an internal helper that uses a specific context
func (p *Plugin) blacklistTokenWithContext(ctx context.Context, tokenStr string) error {
	// Only works if Redis is configured
	if p.redisClient == nil {
		return nil // Silently skip if Redis not available
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

// BlacklistToken adds a token to the blacklist (for logout).
func (p *Plugin) BlacklistToken(tokenStr string) error {
	// Only works if Redis is configured
	if p.redisClient == nil {
		return nil // Silently skip if Redis not available
	}

	ctx := context.Background()

	// Handle Bearer prefix if present
	parts := strings.Split(tokenStr, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		tokenStr = parts[1]
	}

	// Parse token to get expiration time
	token, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithKey(jwa.RS256(), p.accessTokenPublicKey),
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
				// Use background context for rotation since ctx might be cancelled
				if err := p.RotateKeys(context.Background()); err != nil {
					// Log error but continue
					_ = err
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
	return p.db.DeleteExpiredJWKS(ctx)
}
