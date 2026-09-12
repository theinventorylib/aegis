package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/auth"
)

// sessionPageSize is the page size used when walking a user's sessions.
const sessionPageSize = 100

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

	// bearerValidators is a chain of validators for non-session bearer tokens
	// (e.g., JWT access tokens). Each registered plugin appends to this slice.
	// Validators are tried in registration order; the first to succeed wins.
	bearerValidators []BearerTokenValidator

	// mu protects concurrent access to bearerAuthEnabled and bearerValidators
	mu sync.RWMutex

	// auditLogger records session creation and validation events
	auditLogger AuditLogger

	// logger surfaces non-fatal operational errors (e.g., Redis cache
	// failures). Defaults to a no-op logger so callers do not have to
	// pass one.
	logger Logger
}

// newSessionService creates a new session service with optional Redis caching.
//
// If cfg.Redis is provided, a Redis client is created for session caching,
// avoiding a database query for every authenticated request.
func newSessionService(userStore auth.UserStore, sessionStore auth.SessionStore, cfg *SessionConfig, auditLogger AuditLogger, logger Logger) *SessionService {
	if cfg == nil {
		cfg = DefaultSessionConfig()
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}
	if logger == nil {
		logger = noopLogger{}
	}

	return &SessionService{
		userStore:     userStore,
		sessionStore:  sessionStore,
		config:        cfg,
		cookieManager: NewCookieManager(cfg),
		redisClient:   newRedisClient(cfg.Redis),
		auditLogger:   auditLogger,
		logger:        logger,
	}
}

// newRedisClient builds a Redis client from session Redis config; nil config
// means caching is disabled.
func newRedisClient(cfg *RedisConfig) *redis.Client {
	if cfg == nil {
		return nil
	}
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

// CreateSession creates a new authenticated session for a user.
//
// Generates cryptographically secure random tokens for both session access
// and refresh tokens. The raw tokens are returned to the caller (for cookie
// or response use), but only their SHA-256 hashes are persisted to the
// database and to Redis. A DB read alone therefore cannot impersonate the
// user — an attacker would also need to compute a pre-image of the hash.
//
// IP address and user agent are extracted from the request context
// (populated by AegisContextMiddleware). Logs a successful login audit event.
func (s *SessionService) CreateSession(ctx context.Context, user *auth.User) (*auth.Session, error) {
	uid := user.GetID()
	session, err := s.newSessionRow(ctx, uid,
		SanitizeString(GetIPAddress(ctx), nil),
		SanitizeString(GetUserAgent(ctx), nil))
	if err != nil {
		return nil, err
	}
	logAuthEvent(ctx, s.logger, s.auditLogger, AuditEventLoginSuccess, uid, true, nil)
	return session, nil
}

// newSessionRow generates raw tokens, persists only their SHA-256 hashes at
// rest, caches the session, and returns it carrying the raw tokens for the
// caller to hand to the client. Shared by CreateSession and RefreshSession.
func (s *SessionService) newSessionRow(ctx context.Context, userID, ipAddress, userAgent string) (*auth.Session, error) {
	rawToken, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate access token", err)
	}
	rawRefreshToken, err := generateRandomToken()
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate refresh token", err)
	}

	persisted := auth.Session{
		ID:           GenerateID(),
		UserID:       userID,
		Token:        hashTokenHex(rawToken),
		RefreshToken: hashTokenHex(rawRefreshToken),
		ExpiresAt:    time.Now().Add(s.config.SessionExpiry),
		CreatedAt:    time.Now(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}
	if err := s.sessionStore.Create(ctx, persisted); err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to create session", err)
	}
	s.cacheSession(ctx, &persisted)

	// Return the session with raw tokens so the caller can hand them to
	// the client. The persisted copy keeps only hashes.
	out := persisted
	out.Token = rawToken
	out.RefreshToken = rawRefreshToken
	return &out, nil
}

// ValidateSession validates a session token and returns the session and user.
//
// The validation flow:
//  1. Hash the input token (SHA-256)
//  2. Check Redis cache if available (fast path, keyed by hash)
//  3. Fall back to database lookup if not cached
//  4. Verify session hasn't expired
//  5. Load associated user data
//  6. Cache the session in Redis for future requests
//
// This method is called on every authenticated request, so caching is critical
// for performance in production deployments.
//
// Returns:
//   - *auth.Session: The valid session. Token/RefreshToken on the returned
//     struct hold their **hashed** values (the raw token presented by the
//     client is not retained) — callers needing the raw token already have
//     it as the input argument.
//   - *auth.User: The user associated with this session
//   - error: AuthErrorCodeTokenExpired if expired, AuthErrorCodeSessionInvalid
//     if not found, AuthErrorCodeUserNotFound if user was deleted
func (s *SessionService) ValidateSession(ctx context.Context, tokenString string) (*auth.Session, *auth.User, error) {
	tokenHash := hashTokenHex(tokenString)

	if s.redisClient != nil {
		session, err := s.getSessionFromCache(ctx, tokenHash)
		if err == nil && session != nil {
			user, err := s.loadUserForSession(ctx, session)
			if err != nil {
				// A cached expired session is stale cache; drop it.
				if time.Now().After(session.ExpiresAt) {
					s.invalidateSessionCache(ctx, session)
				}
				return nil, nil, err
			}
			return session, user, nil
		}
	}

	dbSession, err := s.sessionStore.GetByToken(ctx, tokenHash)
	if err != nil {
		return nil, nil, NewAuthErrorWithCause(AuthErrorCodeSessionInvalid, "session not found", err)
	}

	if s.redisClient != nil {
		s.cacheSession(ctx, &dbSession)
	}

	user, err := s.loadUserForSession(ctx, &dbSession)
	if err != nil {
		return nil, nil, err
	}

	return &dbSession, user, nil
}

// loadUserForSession enforces expiry and loads the user for a fetched
// session (cache or DB path both end here).
func (s *SessionService) loadUserForSession(ctx context.Context, session *auth.Session) (*auth.User, error) {
	if time.Now().After(session.ExpiresAt) {
		return nil, NewAuthError(AuthErrorCodeTokenExpired, "session expired")
	}
	user, err := s.userStore.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeUserNotFound, "user not found", err)
	}
	return &user, nil
}

// cacheSession stores a session in Redis for fast validation lookups.
//
// Caching strategy (all keys derived from SHA-256 hash of the corresponding
// token — never the raw secret):
//   - sha256(session token) -> session data (TTL = session expiry)
//   - sha256(refresh token) -> session data (TTL = refresh expiry)
//   - User ID -> set of session IDs (for bulk invalidation)
//
// The session passed in is expected to carry **hashed** tokens in its
// Token/RefreshToken fields (this is the canonical persisted shape).
//
// This method is a no-op if Redis is not configured.
func (s *SessionService) cacheSession(ctx context.Context, session *auth.Session) {
	if s.redisClient == nil || session == nil {
		return
	}
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("session cache: failed to marshal session", "session_id", session.ID, "error", err)
		return
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return
	}
	// session.Token is the hash — use it directly as the cache key suffix.
	if err := s.redisClient.Set(ctx, RedisSessionPrefix+session.Token, sessionJSON, ttl).Err(); err != nil {
		s.logger.Error("session cache: failed to set session token", "session_id", session.ID, "error", err)
	}
	if session.RefreshToken != "" {
		if err := s.redisClient.Set(ctx, RedisRefreshTokenPrefix+session.RefreshToken, sessionJSON, s.config.RefreshExpiry).Err(); err != nil {
			s.logger.Error("session cache: failed to set refresh token", "session_id", session.ID, "error", err)
		}
	}
	if err := s.redisClient.SAdd(ctx, RedisUserSessionsPrefix+session.UserID, session.ID).Err(); err != nil {
		s.logger.Error("session cache: failed to add to user sessions set", "session_id", session.ID, "user_id", session.UserID, "error", err)
	}
}

// invalidateSessionCache removes session from Redis cache. The session is
// expected to carry hashed tokens (as stored).
func (s *SessionService) invalidateSessionCache(ctx context.Context, session *auth.Session) {
	if s.redisClient == nil || session == nil {
		return
	}
	if err := s.redisClient.Del(ctx, RedisSessionPrefix+session.Token).Err(); err != nil {
		s.logger.Error("session cache: failed to delete session token", "session_id", session.ID, "error", err)
	}
	if session.RefreshToken != "" {
		if err := s.redisClient.Del(ctx, RedisRefreshTokenPrefix+session.RefreshToken).Err(); err != nil {
			s.logger.Error("session cache: failed to delete refresh token", "session_id", session.ID, "error", err)
		}
	}
	if err := s.redisClient.SRem(ctx, RedisUserSessionsPrefix+session.UserID, session.ID).Err(); err != nil {
		s.logger.Error("session cache: failed to remove from user sessions set", "session_id", session.ID, "user_id", session.UserID, "error", err)
	}
}

// getSessionFromCache retrieves session from Redis cache by token hash.
func (s *SessionService) getSessionFromCache(ctx context.Context, tokenHash string) (*auth.Session, error) {
	res, err := s.redisClient.Get(ctx, RedisSessionPrefix+tokenHash).Result()
	if err != nil {
		return nil, err
	}
	var session auth.Session
	if err := json.Unmarshal([]byte(res), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession deletes a session and invalidates cache. Accepts the **raw**
// session token from the client; the token is hashed before any store/cache
// access.
func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	tokenHash := hashTokenHex(token)
	session, err := s.sessionStore.GetByToken(ctx, tokenHash)
	if err != nil {
		// Without the session record we cannot reliably invalidate the
		// cache or attribute the audit event. Surface the error so the
		// caller knows the logout was incomplete.
		return NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to load session for deletion", err)
	}
	if err := s.sessionStore.Delete(ctx, session.ID); err != nil {
		return err
	}
	s.invalidateSessionCache(ctx, &session)
	logAuthEvent(ctx, s.logger, s.auditLogger, AuditEventLogout, session.UserID, true, nil)
	return nil
}

// RefreshSession rotates an existing session using a refresh token.
//
// Implementation note: this is a delete-old + create-new operation rather
// than an in-place update. That guarantees the access token rotates at rest
// (the existing UpdateSession SQL does not touch the token column) and keeps
// session-token rotation atomic with refresh-token rotation.
//
// Accepts the **raw** refresh token from the client; the token is hashed
// before any store access. Returns a session with **raw** tokens in its
// Token/RefreshToken fields so the caller can hand them to the client.
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*auth.Session, error) {
	refreshHash := hashTokenHex(refreshToken)
	old, err := s.sessionStore.GetByRefreshToken(ctx, refreshHash)
	if err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeTokenInvalid, "invalid refresh token", err)
	}

	s.invalidateSessionCache(ctx, &old)

	if err := s.sessionStore.Delete(ctx, old.ID); err != nil {
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to revoke old session", err)
	}

	out, err := s.newSessionRow(ctx, old.UserID, old.IPAddress, old.UserAgent)
	if err != nil {
		return nil, err
	}

	// Token rotation is a security event: record it like login/logout.
	logAuthEvent(ctx, s.logger, s.auditLogger, AuditEventSessionRefresh, old.UserID, true, nil)

	return out, nil
}

func generateRandomToken() (string, error) {
	bytes, err := randomBytes(TokenLength)
	if err != nil {
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

// AddBearerTokenValidator appends a validator to the bearer token validation
// chain. Multiple plugins can each register their own validator; they are tried
// in registration order and the first to return a non-nil user wins.
func (s *SessionService) AddBearerTokenValidator(v BearerTokenValidator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bearerValidators = append(s.bearerValidators, v)
}

// GetBearerTokenValidators returns a snapshot of the registered validators.
func (s *SessionService) GetBearerTokenValidators() []BearerTokenValidator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := make([]BearerTokenValidator, len(s.bearerValidators))
	copy(snap, s.bearerValidators)
	return snap
}

// GetConfig returns the session configuration.
func (s *SessionService) GetConfig() *SessionConfig { return s.config }

// Configure replaces the session configuration and Redis client. Intended to
// be called once at startup (before any traffic) when the service was built
// without a config, e.g. via NewAuthService. Rebuilds the cookie manager so
// cookie settings take effect.
func (s *SessionService) Configure(cfg *SessionConfig) {
	if cfg == nil {
		return
	}
	s.config = cfg
	s.cookieManager = NewCookieManager(cfg)
	s.redisClient = newRedisClient(cfg.Redis)
}

// GetCookieManager returns the cookie manager.
func (s *SessionService) GetCookieManager() *CookieManager { return s.cookieManager }

// GetRedisClient returns the Redis client used for session storage.
func (s *SessionService) GetRedisClient() *redis.Client { return s.redisClient }

// Logout deletes a session by token (alias for DeleteSession)
func (s *SessionService) Logout(ctx context.Context, token string) error {
	return s.DeleteSession(ctx, token)
}

// GetUserSessions retrieves a paginated list of active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID string, offset, limit int) ([]*auth.Session, error) {
	sessions, err := s.sessionStore.GetByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*auth.Session, len(sessions))
	for i := range sessions {
		result[i] = &sessions[i]
	}
	return result, nil
}

// CountUserSessions returns the total number of active sessions for a user
func (s *SessionService) CountUserSessions(ctx context.Context, userID string) (int, error) {
	return s.sessionStore.CountByUserID(ctx, userID)
}

// purgeUserSessionCache removes every cached session entry for a user from
// Redis without touching the database.
//
// The per-session cache keys are derived from token hashes, so the purge
// walks the store for the user's sessions to discover them. It therefore
// MUST run while those rows still exist — i.e. before the DB session purge.
// Callers that delete the DB rows first (password change) would otherwise
// purge nothing and leave revoked sessions authenticating from cache until
// their TTL expires (ValidateSession prefers the cache over the database).
//
// No-op when Redis is not configured. Per-session Redis failures are logged
// by invalidateSessionCache; only the store walk can return an error.
func (s *SessionService) purgeUserSessionCache(ctx context.Context, userID string) error {
	if s.redisClient == nil {
		return nil
	}
	if err := s.forEachUserSession(ctx, userID, func(sess *auth.Session) error {
		s.invalidateSessionCache(ctx, sess)
		return nil
	}); err != nil {
		return err
	}
	// Drop the user-session set entirely — it is now empty.
	if err := s.redisClient.Del(ctx, RedisUserSessionsPrefix+userID).Err(); err != nil {
		s.logger.Error("session cache: failed to delete user session set", "user_id", userID, "error", err)
	}
	return nil
}

// DeleteUserSessions deletes all sessions for a user and purges every
// Redis cache entry for those sessions. Without the cache purge, revoked
// sessions would keep authenticating from Redis until their cached TTL
// expires (ValidateSession prefers the cache over the database).
func (s *SessionService) DeleteUserSessions(ctx context.Context, userID string) error {
	// Purge the cache first: it needs the session rows to exist.
	if err := s.purgeUserSessionCache(ctx, userID); err != nil {
		return err
	}
	return s.sessionStore.DeleteByUserID(ctx, userID)
}

// errStopPage aborts forEachUserSession's walk without an error.
var errStopPage = errors.New("stop page walk")

// forEachUserSession pages through all of a user's sessions, calling fn for
// each. Return errStopPage from fn to stop the walk early.
func (s *SessionService) forEachUserSession(ctx context.Context, userID string, fn func(*auth.Session) error) error {
	offset := 0
	for {
		page, err := s.sessionStore.GetByUserID(ctx, userID, offset, sessionPageSize)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		for i := range page {
			if err := fn(&page[i]); err != nil {
				if errors.Is(err, errStopPage) {
					return nil
				}
				return err
			}
		}
		offset += len(page)
	}
}

// RevokeSessionByID revokes a specific session for a user by its ID,
// verifying ownership first.
func (s *SessionService) RevokeSessionByID(ctx context.Context, userID, sessionID string) error {
	// Find the session in the user's active sessions to confirm ownership.
	// We delete by session ID directly because sessions loaded from the store
	// carry hashed tokens — we cannot feed those back into DeleteSession (which
	// expects a raw token and would hash again).
	var target *auth.Session
	err := s.forEachUserSession(ctx, userID, func(sess *auth.Session) error {
		if sess.ID == sessionID {
			t := *sess
			target = &t
			return errStopPage
		}
		return nil
	})
	if err != nil {
		return err
	}
	if target == nil {
		return ErrSessionNotFound
	}

	if err := s.sessionStore.Delete(ctx, target.ID); err != nil {
		return err
	}
	s.invalidateSessionCache(ctx, target)
	logAuthEvent(ctx, s.logger, s.auditLogger, AuditEventLogout, target.UserID, true, nil)
	return nil
}

// MigrateHashSessionTokensForUser rewrites any plaintext session/refresh tokens in
// the underlying store as their SHA-256 hex hashes (the at-rest format used
// by CreateSession). It is idempotent: rows whose token columns already look
// like SHA-256 hex digests (see isHashedToken) are skipped.
//
// Run this once after upgrading to the hashed-at-rest scheme. After a
// successful run, all subsequent ValidateSession / DeleteSession /
// RefreshSession calls will continue to work because they hash inputs before
// querying the store. Plaintext tokens issued before the migration become
// invalid the moment they are rewritten — clients holding only those tokens
// must re-authenticate.
//
// The walk uses GetByUserID per user, requiring callers to provide a userID
// iterator. We expose two entry points:
//
//   - MigrateHashSessionTokensForUser: rotates a single user's rows. Use this
//     for incremental migrations or scripted one-offs.
//
// A bulk migration across all users is intentionally not provided here:
// SessionStore does not expose a "list all session IDs" query, and adding one
// would expand the public store contract for a one-time operation. Callers
// that need a global migration should iterate user IDs from their UserStore
// and invoke MigrateHashSessionTokensForUser per user.
func (s *SessionService) MigrateHashSessionTokensForUser(ctx context.Context, userID string) (migrated int, err error) {
	// Snapshot all of the user's sessions before mutating. The migration
	// delete+re-creates rows (changing their IDs), so reading while writing
	// would shift rows between pages; collecting first makes every row
	// reachable regardless of how many the user has.
	var all []auth.Session
	for offset := 0; ; {
		page, err := s.sessionStore.GetByUserID(ctx, userID, offset, sessionPageSize)
		if err != nil {
			return migrated, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		offset += len(page)
		// Defensive cap to bound memory for pathological stores.
		if len(all) > 1_000_000 {
			break
		}
	}

	for i := range all {
		row := all[i]
		needsToken := row.Token != "" && !isHashedToken(row.Token)
		needsRefresh := row.RefreshToken != "" && !isHashedToken(row.RefreshToken)
		if !needsToken && !needsRefresh {
			continue
		}
		// Re-issue the row with hashed values. We cannot change the
		// access token via Update (the existing UpdateSession SQL only
		// touches refresh_token + expires_at), so legacy access tokens
		// are invalidated by deleting and re-creating with the hashed
		// value. Refresh tokens are similarly hashed.
		updated := row
		if needsToken {
			updated.Token = hashTokenHex(row.Token)
		}
		if needsRefresh {
			updated.RefreshToken = hashTokenHex(row.RefreshToken)
		}
		if err := s.sessionStore.Delete(ctx, row.ID); err != nil {
			return migrated, err
		}
		if err := s.sessionStore.Create(ctx, updated); err != nil {
			return migrated, err
		}
		s.invalidateSessionCache(ctx, &updated)
		migrated++
	}
	return migrated, nil
}
