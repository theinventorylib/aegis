package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/auth"
)

// SessionService manages user session lifecycle including creation, validation,
// refresh, and invalidation. It provides optional Redis-based caching for
// high-performance session lookups in high-traffic applications.
//
// Key features:
//   - Token-based authentication with access and refresh tokens
//   - Optional Redis caching layer for fast session validation
//   - Session expiry and refresh token rotation
//   - IP address and user agent tracking for security auditing
//   - Bulk session invalidation (logout all devices)
//
// The service is safe for concurrent use and should be shared across handlers.
type SessionService struct {
	// userStore retrieves user data during session validation
	userStore auth.UserStore

	// sessionStore persists sessions to the database
	sessionStore auth.SessionStore

	// config holds session duration and Redis settings
	config *SessionConfig

	// cookieManager handles secure cookie operations
	cookieManager *CookieManager

	// redisClient enables session caching (nil if caching disabled)
	redisClient *redis.Client

	// bearerAuthEnabled indicates if bearer token auth is supported
	bearerAuthEnabled bool

	// mu protects concurrent access to bearerAuthEnabled
	mu sync.RWMutex

	// auditLogger records session creation and validation events
	auditLogger AuditLogger
}

// NewSessionService creates a new session service with optional Redis caching.
//
// Parameters:
//   - userStore: Storage for user lookups during session validation
//   - sessionStore: Storage for session persistence
//   - cfg: Session configuration (expiry, Redis settings). Uses defaults if nil.
//   - auditLogger: Logger for security events. Uses no-op if nil.
//
// If cfg.Redis is provided, a Redis client is created for session caching.
// This significantly improves performance by avoiding database queries for
// every authenticated request.
func NewSessionService(userStore auth.UserStore, sessionStore auth.SessionStore, cfg *SessionConfig, auditLogger AuditLogger) *SessionService {
	if cfg == nil {
		cfg = DefaultSessionConfig()
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}

	var redisClient *redis.Client
	if cfg.Redis != nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}

	return &SessionService{
		userStore:     userStore,
		sessionStore:  sessionStore,
		config:        cfg,
		cookieManager: NewCookieManager(cfg),
		redisClient:   redisClient,
		auditLogger:   auditLogger,
	}
}

// CreateSession creates a new authenticated session for a user.
//
// Generates cryptographically secure random tokens for both session access
// and refresh tokens. The session is persisted to the database and optionally
// cached in Redis for fast subsequent lookups.
//
// Parameters:
//   - ctx: Request context for cancellation
//   - user: The authenticated user to create a session for
//   - ipAddress: Client IP address for security auditing
//   - userAgent: Client user agent for security auditing
//
// Returns the created session with populated Token and RefreshToken fields.
// These tokens should be sent to the client (typically via HTTP-only cookies
// or Authorization header).
//
// Logs a successful login audit event upon session creation.
func (s *SessionService) CreateSession(ctx context.Context, user *auth.User, ipAddress, userAgent string) (*auth.Session, error) {
	uid := user.GetID()

	token, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate access token", err)
	}
	refreshToken, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate refresh token", err)
	}
	expiresAt := time.Now().Add(s.config.SessionExpiry)

	session := auth.Session{
		ID:           GenerateID(),
		UserID:       uid,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to create session", err)
	}

	if s.redisClient != nil {
		s.cacheSession(ctx, &session)
	}

	_ = s.auditLogger.LogAuthEvent(ctx, AuditEventLoginSuccess, uid, ipAddress, userAgent, true, nil)
	return &session, nil
}

// ValidateSession validates a session token and returns the session and user.
//
// The validation flow:
//  1. Check Redis cache if available (fast path)
//  2. Fall back to database lookup if not cached
//  3. Verify session hasn't expired
//  4. Load associated user data
//  5. Cache the session in Redis for future requests
//
// This method is called on every authenticated request, so caching is critical
// for performance in production deployments.
//
// Parameters:
//   - ctx: Request context for cancellation
//   - tokenString: The session token to validate
//
// Returns:
//   - *auth.Session: The valid session
//   - *auth.User: The user associated with this session
//   - error: AuthErrorCodeTokenExpired if expired, AuthErrorCodeSessionInvalid
//     if not found, AuthErrorCodeUserNotFound if user was deleted
func (s *SessionService) ValidateSession(ctx context.Context, tokenString string) (*auth.Session, *auth.User, error) {
	var session *auth.Session
	var err error

	if s.redisClient != nil {
		session, err = s.getSessionFromCache(ctx, tokenString)
		if err == nil && session != nil {
			if time.Now().After(session.ExpiresAt) {
				s.invalidateSessionCache(ctx, session)
				return nil, nil, NewAuthError(AuthErrorCodeTokenExpired, "session expired")
			}
			user, err := s.userStore.GetByID(ctx, session.UserID)
			if err != nil {
				return nil, nil, NewAuthErrorWithCause(AuthErrorCodeUserNotFound, "user not found", err)
			}
			return session, &user, nil
		}
	}

	dbSession, err := s.sessionStore.GetByToken(ctx, tokenString)
	if err != nil {
		return nil, nil, NewAuthErrorWithCause(AuthErrorCodeSessionInvalid, "session not found", err)
	}

	session = &dbSession
	if time.Now().After(session.ExpiresAt) {
		return nil, nil, NewAuthError(AuthErrorCodeTokenExpired, "session expired")
	}

	if s.redisClient != nil {
		s.cacheSession(ctx, session)
	}

	user, err := s.userStore.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, NewAuthErrorWithCause(AuthErrorCodeUserNotFound, "user not found", err)
	}

	return session, &user, nil
}

// cacheSession stores a session in Redis for fast validation lookups.
//
// Caching strategy:
//   - Session token -> session data (TTL = session expiry)
//   - Refresh token -> session data (TTL = refresh expiry)
//   - User ID -> set of session IDs (for bulk invalidation)
//
// This allows:
//   - Fast session validation without database queries
//   - Refresh token lookup
//   - Efficient "logout all devices" by invalidating all user sessions
//
// This method is a no-op if Redis is not configured.
func (s *SessionService) cacheSession(ctx context.Context, session *auth.Session) {
	if s.redisClient == nil || session == nil {
		return
	}
	sessionJSON, _ := json.Marshal(session)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return
	}
	_ = s.redisClient.Set(ctx, RedisSessionPrefix+session.Token, sessionJSON, ttl).Err()
	if session.RefreshToken != "" {
		_ = s.redisClient.Set(ctx, RedisRefreshTokenPrefix+session.RefreshToken, sessionJSON, s.config.RefreshExpiry).Err()
	}
	_ = s.redisClient.SAdd(ctx, RedisUserSessionsPrefix+session.UserID, session.ID).Err()
}

// invalidateSessionCache removes session from Redis cache
func (s *SessionService) invalidateSessionCache(ctx context.Context, session *auth.Session) {
	if s.redisClient == nil || session == nil {
		return
	}
	_ = s.redisClient.Del(ctx, RedisSessionPrefix+session.Token).Err()
	if session.RefreshToken != "" {
		_ = s.redisClient.Del(ctx, RedisRefreshTokenPrefix+session.RefreshToken).Err()
	}
	_ = s.redisClient.SRem(ctx, RedisUserSessionsPrefix+session.UserID, session.ID).Err()
}

// getSessionFromCache retrieves session from Redis cache
func (s *SessionService) getSessionFromCache(ctx context.Context, token string) (*auth.Session, error) {
	res, err := s.redisClient.Get(ctx, RedisSessionPrefix+token).Result()
	if err != nil {
		return nil, err
	}
	var session auth.Session
	if err := json.Unmarshal([]byte(res), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession deletes a session and invalidates cache
func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	session, _ := s.sessionStore.GetByToken(ctx, token)
	err := s.sessionStore.Delete(ctx, token)
	if err == nil {
		s.invalidateSessionCache(ctx, &session)
		_ = s.auditLogger.LogAuthEvent(ctx, AuditEventLogout, session.UserID, session.IPAddress, session.UserAgent, true, nil)
	}
	return err
}

// RefreshSession refreshes a session using a refresh token
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*auth.Session, error) {
	session, err := s.sessionStore.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeTokenInvalid, "invalid refresh token", err)
	}

	s.invalidateSessionCache(ctx, &session)

	token, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate access token", err)
	}
	newRefreshToken, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate refresh token", err)
	}
	session.Token = token
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = time.Now().Add(s.config.SessionExpiry)

	if err := s.sessionStore.Update(ctx, session); err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.cacheSession(ctx, &session)
	}

	return &session, nil
}

func generateRandomToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// EnableBearerAuth enables Bearer token authentication for sessions.
func (s *SessionService) EnableBearerAuth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bearerAuthEnabled = true
}

// IsBearerAuthEnabled checks if Bearer token authentication is enabled.
func (s *SessionService) IsBearerAuthEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bearerAuthEnabled
}

// GetConfig returns the session configuration.
func (s *SessionService) GetConfig() *SessionConfig { return s.config }

// GetCookieManager returns the cookie manager.
func (s *SessionService) GetCookieManager() *CookieManager { return s.cookieManager }

// GetRedisClient returns the Redis client used for session storage.
func (s *SessionService) GetRedisClient() *redis.Client { return s.redisClient }

// Logout deletes a session by token (alias for DeleteSession)
func (s *SessionService) Logout(ctx context.Context, token string) error {
	return s.DeleteSession(ctx, token)
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	sessions, err := s.sessionStore.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*auth.Session, len(sessions))
	for i := range sessions {
		result[i] = &sessions[i]
	}
	return result, nil
}

// DeleteUserSessions deletes all sessions for a user
func (s *SessionService) DeleteUserSessions(ctx context.Context, userID string) error {
	return s.sessionStore.DeleteByUserID(ctx, userID)
}
