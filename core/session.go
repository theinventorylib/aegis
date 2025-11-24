package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// SessionService handles session management
type SessionService struct {
	db            db.Provider
	config        *SessionConfig
	redisClient   *redis.Client
	tokenProvider TokenProvider // Optional: can be nil
}

// NewSessionService creates a new session service
func NewSessionService(database db.Provider, cfg *SessionConfig) *SessionService {
	if cfg == nil {
		cfg = DefaultSessionConfig()
	}

	var redisClient *redis.Client

	// Initialize Redis if configured
	if cfg.Redis != nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}

	return &SessionService{
		db:          database,
		config:      cfg,
		redisClient: redisClient,
		// tokenProvider will be set by plugin if registered
	}
}

// SetTokenProvider allows a plugin to register as the token provider
func (s *SessionService) SetTokenProvider(provider TokenProvider) {
	s.tokenProvider = provider
}

// GetRedisClient returns the Redis client (if configured)
func (s *SessionService) GetRedisClient() *redis.Client {
	return s.redisClient
}

// CreateSession creates a new session for a user
func (s *SessionService) CreateSession(ctx context.Context, user *models.User, ipAddress, userAgent string) (*models.Session, error) {
	var token, refreshToken string
	var expiresAt time.Time

	if s.tokenProvider != nil {
		// Use plugin-provided token generation
		tokenPair, err := s.tokenProvider.GenerateTokenPair(user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token pair: %w", err)
		}
		token = tokenPair.AccessToken
		refreshToken = tokenPair.RefreshToken
		expiresAt = tokenPair.AccessExpiry
	} else {
		// Fallback to simple random token generation
		token = generateRandomToken()
		refreshToken = generateRandomToken()
		expiresAt = time.Now().Add(s.config.SessionExpiry)
	}

	session := &models.Session{
		ID:           GenerateSessionID(),
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
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
func (s *SessionService) ValidateSession(ctx context.Context, tokenString string) (*models.Session, *models.User, error) {
	var userID string

	if s.tokenProvider != nil {
		// Use plugin-provided token validation
		claims, err := s.tokenProvider.ValidateToken(tokenString)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid token: %w", err)
		}
		userID = claims.UserID
	} else {
		// Fallback: validate by database lookup only
		session, err := s.db.GetSession(ctx, tokenString)
		if err != nil {
			return nil, nil, fmt.Errorf("session not found: %w", err)
		}

		// Check if session is expired
		if time.Now().After(session.ExpiresAt) {
			return nil, nil, fmt.Errorf("session expired")
		}

		userID = session.UserID
	}

	// Get session from database (always needed for additional metadata)
	session, err := s.db.GetSession(ctx, tokenString)
	if err != nil {
		return nil, nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is expired (additional check for plugin tokens)
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

// DeleteSession deletes a session and blacklists the token
func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	// Blacklist the token if a token provider is available
	if s.tokenProvider != nil {
		_ = s.tokenProvider.BlacklistToken(token) // Ignore error, continue with database deletion
	}

	// Delete from database
	return s.db.DeleteSession(ctx, token)
}

// RefreshSession refreshes a session using a refresh token
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	// Get session by refresh token
	session, err := s.db.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if s.tokenProvider != nil {
		// Use plugin-provided token refresh
		tokenPair, err := s.tokenProvider.RefreshTokens(refreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh tokens: %w", err)
		}

		// Update session with new tokens
		session.Token = tokenPair.AccessToken
		session.RefreshToken = tokenPair.RefreshToken
		session.ExpiresAt = tokenPair.AccessExpiry
	} else {
		// Fallback: generate new simple tokens
		session.Token = generateRandomToken()
		session.RefreshToken = generateRandomToken()
		session.ExpiresAt = time.Now().Add(s.config.SessionExpiry)
	}

	if err := s.db.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// GenerateSessionID generates a random ID for sessions
func GenerateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate session ID: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// generateRandomToken generates a random token for fallback authentication
func generateRandomToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random token: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// GetConfig returns the session configuration (useful for middleware)
func (s *SessionService) GetConfig() *SessionConfig {
	return s.config
}

// GetDB returns the database provider (useful for handlers)
func (s *SessionService) GetDB() db.Provider {
	return s.db
}

// CreateSessionForPlugin creates a new session (interface implementation for plugins)
// func (s *SessionService) CreateSessionForPlugin(ctx context.Context, user models.User, ipAddress, userAgent string) (interface{}, error) {
// 	// context, ok := ctx.(context.Context)
// 	// if !ok {
// 	// 	return nil, fmt.Errorf("invalid context type")
// 	// }
// 	// u, ok := user.(*User)
// 	// if !ok {
// 	// 	return nil, fmt.Errorf("invalid user type")
// 	// }
// 	session, err := s.CreateSession(ctx, user, ipAddress, userAgent)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return session, nil
// }
