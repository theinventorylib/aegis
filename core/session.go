package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/db"
)

// SessionService handles session management
type SessionService struct {
	db          db.DBProvider
	config      *SessionConfig
	keyManager  KeyManager
	redisClient *redis.Client
}

// NewSessionService creates a new session service
func NewSessionService(database db.DBProvider, cfg *SessionConfig) *SessionService {
	if cfg == nil {
		cfg = DefaultSessionConfig()
	}

	var keyManager KeyManager
	var redisClient *redis.Client

	// Initialize Redis if configured
	if cfg.Redis != nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		keyManager = NewRedisKeyManager(redisClient)
	} else {
		// Fallback to static keys
		km, err := NewStaticKeyManager()
		if err != nil {
			// In production this should probably panic or return error, but signature is fixed
			panic(fmt.Sprintf("failed to initialize static key manager: %v", err))
		}
		keyManager = km
	}

	return &SessionService{
		db:          database,
		config:      cfg,
		keyManager:  keyManager,
		redisClient: redisClient,
	}
}

// CreateSession creates a new session for a user
func (s *SessionService) CreateSession(ctx context.Context, user *User, ipAddress, userAgent string) (*Session, error) {
	// Generate access token (JWT)
	token, err := s.generateJWT(user.ID, s.config.SessionExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	session := &Session{
		ID:           generateSessionID(),
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.config.SessionExpiry),
		CreatedAt:    time.Now(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.db.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ValidateSession validates a session token
func (s *SessionService) ValidateSession(ctx context.Context, tokenString string) (*Session, *User, error) {
	// 1. Check Redis cache first (if enabled)
	if s.redisClient != nil {
		// Check blacklist
		if exists, _ := s.redisClient.Exists(ctx, "auth:blacklist:"+tokenString).Result(); exists > 0 {
			return nil, nil, fmt.Errorf("token revoked")
		}
	}

	// 2. Parse and verify JWT signature
	// We need to find the right key. For now, we just use the current access key or validate against all.
	// In a real rotation scenario, we'd look up key by ID (kid) header.
	// jwx/v3 Parse can handle key sets or providers.
	// For simplicity with our KeyManager, we'll fetch the current key.
	// Ideally, we should fetch the key based on the token's kid.

	// Let's try to parse insecurely first to get kid? No, jwx allows providing a key provider.
	// But our KeyManager isn't exactly a jwx KeyProvider.
	// We'll use the current access key for verification as a baseline.
	// If rotation happened, we might need to check previous keys (not implemented in simple KeyManager yet).

	key, err := s.keyManager.GetAccessKey(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get verification key: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract user ID from token
	userID, ok := token.Subject()
	if !ok {
		return nil, nil, fmt.Errorf("token missing subject")
	}

	// 3. Check session validity
	// If Redis is enabled, we can check if session is cached
	if s.redisClient != nil {
		// TODO: Implement session caching in Redis
		// For now, we still hit DB for session details to ensure consistency
	}

	// Get session from database
	session, err := s.db.GetSession(ctx, tokenString)
	if err != nil {
		return nil, nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, nil, fmt.Errorf("session expired")
	}

	// Get user
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}

	return session, user, nil
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	return s.db.DeleteSession(ctx, token)
}

// RefreshSession refreshes a session using a refresh token
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*Session, error) {
	// Get session by refresh token
	session, err := s.db.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Get user
	user, err := s.db.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new access token
	token, err := s.generateJWT(user.ID, s.config.SessionExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Update session
	session.Token = token
	session.ExpiresAt = time.Now().Add(s.config.SessionExpiry)

	if err := s.db.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// generateJWT generates a JWT token
func (s *SessionService) generateJWT(userID string, expiry time.Duration) (string, error) {
	now := time.Now()

	// Get signing key
	key, err := s.keyManager.GetAccessKey(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get signing key: %w", err)
	}

	token, err := jwt.NewBuilder().
		Subject(userID).
		JwtID(generateSessionID()). // Add unique ID
		IssuedAt(now).
		Expiration(now.Add(expiry)).
		Build()
	if err != nil {
		return "", err
	}

	// Sign with RS256
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

// generateRefreshToken generates a random refresh token
func (s *SessionService) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateSessionID generates a random ID for sessions
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// GetConfig returns the session configuration (useful for middleware)
func (s *SessionService) GetConfig() *SessionConfig {
	return s.config
}

// GetDB returns the database provider (useful for handlers)
func (s *SessionService) GetDB() db.DBProvider {
	return s.db
}
