package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// SessionService handles session management
type SessionService struct {
	db                db.Provider
	config            *SessionConfig
	redisClient       *redis.Client
	bearerAuthEnabled bool         // Controls whether Bearer token auth is enabled
	mu                sync.RWMutex // Protects bearerAuthEnabled
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
	}
}

// GetRedisClient returns the Redis client (if configured)
func (s *SessionService) GetRedisClient() *redis.Client {
	return s.redisClient
}

// CreateSession creates a new session for a user
func (s *SessionService) CreateSession(ctx context.Context, user *models.User, ipAddress, userAgent string) (*models.Session, error) {
	var token, refreshToken string
	var expiresAt time.Time

	// Fallback to simple random token generation
	var err error
	token, err = generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err = generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	expiresAt = time.Now().Add(s.config.SessionExpiry)

	// Generate a session ID; avoid panicking on entropy failures by returning an error
	sid, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session id: %w", err)
	}

	session := &models.Session{
		ID:           sid,
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
	// Delete from database
	return s.db.DeleteSession(ctx, token)
}

// RefreshSession refreshes a session using a refresh token
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	// Get session by refresh token
	session, sErr := s.db.GetSessionByRefreshToken(ctx, refreshToken)
	if sErr != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", sErr)
	}

	// Fallback: generate new simple tokens
	var err error
	session.Token, err = generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	session.RefreshToken, err = generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	session.ExpiresAt = time.Now().Add(s.config.SessionExpiry)

	if err := s.db.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// GenerateSessionID generates a random ID for sessions
func GenerateSessionID() (string, error) {
	// Try to read cryptographically secure random bytes; return an error if unavailable
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateRandomToken generates a random token for fallback authentication
func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetConfig returns the session configuration (useful for middleware)
func (s *SessionService) GetConfig() *SessionConfig {
	return s.config
}

// GetDB returns the database provider (useful for handlers)
func (s *SessionService) GetDB() db.Provider {
	return s.db
}

// EnableBearerAuth enables Bearer token authentication.
// This should be called by the bearer plugin during initialization.
func (s *SessionService) EnableBearerAuth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bearerAuthEnabled = true
}

// IsBearerAuthEnabled returns whether Bearer token authentication is enabled.
// The core AuthMiddleware checks this before attempting to extract Bearer tokens.
func (s *SessionService) IsBearerAuthEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bearerAuthEnabled
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
