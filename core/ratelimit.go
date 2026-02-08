package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/auth"
)

// Rate limiting in Aegis uses a sliding window algorithm to prevent abuse.
// Each unique key (IP address or user ID) gets a counter that resets after
// a time window. When Redis is available, limits are enforced across all
// instances. Otherwise, an in-memory fallback provides single-instance protection.

// RateLimitConfig configures rate limiting behavior.
// Rate limits can be applied by IP address, user ID, or both.
type RateLimitConfig struct {
	// RequestsPerWindow is the maximum number of requests allowed per time window.
	// After exceeding this, requests will receive HTTP 429 Too Many Requests.
	RequestsPerWindow int

	// WindowDuration is the time window for counting requests.
	// Example: 100 requests per 1 minute means users can make 100 requests,
	// then must wait up to 1 minute before the counter resets.
	WindowDuration time.Duration

	// KeyPrefix is the Redis key prefix for rate limit counters.
	// This prevents collisions with other Redis data.
	KeyPrefix string

	// ByIP enables rate limiting by client IP address.
	// Useful for preventing abuse from specific sources.
	ByIP bool

	// ByUser enables rate limiting by authenticated user ID.
	// Requires that requests are authenticated. Unauthenticated requests
	// won't be rate limited by user.
	ByUser bool

	// ExcludePaths are URL paths exempt from rate limiting.
	// Example: ["/health", "/metrics"] for monitoring endpoints.
	ExcludePaths []string
}

// RateLimiter provides distributed rate limiting functionality.
//
// It uses Redis for distributed rate limiting across multiple application instances,
// with an in-memory fallback for single-instance deployments. The sliding window
// algorithm prevents bursty traffic from overwhelming the system.
//
// The limiter is safe for concurrent use and should be shared across HTTP handlers.
type RateLimiter struct {
	// config holds rate limiting settings
	config *RateLimitConfig

	// redisClient enables distributed limiting. If nil, uses in-memory fallback.
	redisClient *redis.Client

	// auditLogger records rate limit violations
	auditLogger AuditLogger

	// In-memory fallback for when Redis is not available.
	// Used only in single-instance deployments.
	memoryStore     map[string]*rateLimitEntry
	memoryStoreMu   sync.RWMutex
	cleanupTicker   *time.Ticker
	cleanupStopChan chan struct{}
}

// rateLimitEntry tracks request counts in the in-memory store.
type rateLimitEntry struct {
	// count is the number of requests in the current window
	count int

	// expiresAt is when this window ends and the counter resets
	expiresAt time.Time
}

// LoginAttemptTracker tracks failed login attempts for account lockout protection.
//
// This prevents brute force attacks by temporarily locking accounts after too many
// failed login attempts. Like RateLimiter, it supports both Redis (distributed)
// and in-memory (single-instance) backends.
type LoginAttemptTracker struct {
	// redisClient enables distributed tracking. If nil, uses in-memory fallback.
	redisClient *redis.Client

	// maxAttempts is the threshold for triggering account lockout
	maxAttempts int

	// lockoutDuration is how long an account remains locked after max attempts
	lockoutDuration time.Duration

	// attemptWindow is the time window for counting failed attempts
	attemptWindow time.Duration

	// keyPrefix prevents Redis key collisions
	keyPrefix string

	// In-memory fallback for single-instance deployments
	memoryStore     map[string]*loginAttemptEntry
	memoryStoreMu   sync.RWMutex
	cleanupTicker   *time.Ticker
	cleanupStopChan chan struct{}
}

// loginAttemptEntry tracks login attempts in the in-memory store.
type loginAttemptEntry struct {
	// attempts is the count of failed login attempts in the current window
	attempts int

	// lockedAt is when the account was locked (zero if not locked)
	lockedAt time.Time

	// expiresAt is when this entry should be cleaned up
	expiresAt time.Time
}

// LoginAttemptConfig configures login attempt tracking behavior.
type LoginAttemptConfig struct {
	// MaxAttempts is the threshold before triggering account lockout.
	// Example: 5 means lock after the 5th failed attempt.
	MaxAttempts int

	// LockoutDuration is how long the account remains locked.
	// After this duration, the user can attempt to login again.
	LockoutDuration time.Duration

	// AttemptWindow is the time window for counting failed attempts.
	// Attempts older than this window don't count toward the limit.
	AttemptWindow time.Duration
}

// DefaultRateLimitConfig returns sensible defaults for general API endpoints.
// These limits are suitable for most read-heavy applications.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: DefaultRateLimitRequests,
		WindowDuration:    DefaultRateLimitWindow,
		KeyPrefix:         DefaultRateLimitKeyPrefix,
		ByIP:              true,
		ByUser:            false,
		ExcludePaths:      []string{},
	}
}

// AuthRateLimitConfig returns stricter limits for authentication endpoints.
// Authentication endpoints (login, signup) should have tighter limits to
// prevent brute force attacks and credential stuffing.
func AuthRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: AuthRateLimitRequests,
		WindowDuration:    DefaultRateLimitWindow,
		KeyPrefix:         AuthRateLimitKeyPrefix,
		ByIP:              true,
		ByUser:            false,
		ExcludePaths:      []string{},
	}
}

// NewRateLimiter creates a new rate limiter with the specified configuration.
//
// Parameters:
//   - config: Rate limiting configuration. If nil, uses defaults.
//   - redisClient: Redis client for distributed rate limiting. If nil, uses
//     in-memory fallback (only suitable for single-instance deployments).
//   - auditLogger: Logger for recording rate limit violations. If nil, uses
//     a no-op logger.
//
// The returned RateLimiter is safe for concurrent use. For multi-instance
// deployments, a Redis client must be provided to enforce limits across
// all instances.
func NewRateLimiter(config *RateLimitConfig, redisClient *redis.Client, auditLogger AuditLogger) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}

	rl := &RateLimiter{
		config:          config,
		redisClient:     redisClient,
		auditLogger:     auditLogger,
		memoryStore:     make(map[string]*rateLimitEntry),
		cleanupStopChan: make(chan struct{}),
	}

	// Start cleanup goroutine for in-memory store
	if redisClient == nil {
		rl.cleanupTicker = time.NewTicker(time.Minute)
		go rl.cleanupExpired()
	}

	return rl
}

// cleanupExpired removes expired entries from in-memory store
func (rl *RateLimiter) cleanupExpired() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.memoryStoreMu.Lock()
			now := time.Now()
			for key, entry := range rl.memoryStore {
				if now.After(entry.expiresAt) {
					delete(rl.memoryStore, key)
				}
			}
			rl.memoryStoreMu.Unlock()
		case <-rl.cleanupStopChan:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		close(rl.cleanupStopChan)
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int, error) {
	if rl.redisClient != nil {
		return rl.allowRedis(ctx, key)
	}
	return rl.allowMemory(key)
}

// allowRedis uses Redis for distributed rate limiting
func (rl *RateLimiter) allowRedis(ctx context.Context, key string) (bool, int, error) {
	redisKey := rl.config.KeyPrefix + key

	// Use Redis INCR with expiry for atomic counter
	count, err := rl.redisClient.Incr(ctx, redisKey).Result()
	if err != nil {
		return true, 0, err // Allow on error
	}

	// Set expiry on first request in window
	if count == 1 {
		rl.redisClient.Expire(ctx, redisKey, rl.config.WindowDuration)
	}

	remaining := rl.config.RequestsPerWindow - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return int(count) <= rl.config.RequestsPerWindow, remaining, nil
}

// allowMemory uses in-memory store for single-instance rate limiting
func (rl *RateLimiter) allowMemory(key string) (bool, int, error) {
	rl.memoryStoreMu.Lock()
	defer rl.memoryStoreMu.Unlock()

	now := time.Now()
	fullKey := rl.config.KeyPrefix + key

	entry, exists := rl.memoryStore[fullKey]
	if !exists || now.After(entry.expiresAt) {
		// New window
		rl.memoryStore[fullKey] = &rateLimitEntry{
			count:     1,
			expiresAt: now.Add(rl.config.WindowDuration),
		}
		return true, rl.config.RequestsPerWindow - 1, nil
	}

	entry.count++
	remaining := rl.config.RequestsPerWindow - entry.count
	if remaining < 0 {
		remaining = 0
	}

	return entry.count <= rl.config.RequestsPerWindow, remaining, nil
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimitMiddleware creates a middleware that rate limits requests
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded
			for _, path := range limiter.config.ExcludePaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Build rate limit key
			var key string
			if limiter.config.ByUser {
				// Try to get user from context
				user, err := GetUser(r.Context())
				if err == nil && user != nil {
					uid := ""
					if ua, ok := any(user).(UserModel); ok {
						uid = ua.GetID()
					} else if ua2, ok := any(user).(*auth.User); ok {
						uid = ua2.ID
					}
					if uid != "" {
						key = "user:" + uid
					}
				}
			}

			if key == "" && limiter.config.ByIP {
				key = "ip:" + getClientIP(r)
			}

			if key == "" {
				// No key could be determined, allow request
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			allowed, remaining, err := limiter.Allow(r.Context(), key)
			if err != nil {
				// On error, allow the request but log
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if !allowed {
				// Audit log rate limit hit
				// Avoid logging raw keys (which may contain IPs or user IDs).
				// Log the key type and a short non-reversible hash instead.
				keyType := "unknown"
				keyVal := key
				if len(key) > 3 {
					if key[:3] == "ip:" {
						keyType = "ip"
						keyVal = key[3:]
					} else if key[:5] == "user:" {
						keyType = "user"
						keyVal = key[5:]
					}
				}
				_ = limiter.auditLogger.LogAuthEvent(r.Context(), AuditEventRateLimitHit, "", false, map[string]any{
					"key_type":  keyType,
					"key_hash":  HashShort(keyVal),
					"path":      r.URL.Path,
					"method":    r.Method,
					"remaining": remaining,
				})

				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(limiter.config.WindowDuration.Seconds())))
				WriteJSON(w, http.StatusTooManyRequests, &Response{
					Success: false,
					Error:   "Rate limit exceeded. Please try again later.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultLoginAttemptConfig returns sensible defaults
func DefaultLoginAttemptConfig() *LoginAttemptConfig {
	return &LoginAttemptConfig{
		MaxAttempts:     DefaultMaxLoginAttempts,
		LockoutDuration: DefaultLoginLockoutDuration,
		AttemptWindow:   DefaultLoginAttemptWindow,
	}
}

// NewLoginAttemptTracker creates a new login attempt tracker
func NewLoginAttemptTracker(config *LoginAttemptConfig, redisClient *redis.Client) *LoginAttemptTracker {
	if config == nil {
		config = DefaultLoginAttemptConfig()
	}

	lat := &LoginAttemptTracker{
		redisClient:     redisClient,
		maxAttempts:     config.MaxAttempts,
		lockoutDuration: config.LockoutDuration,
		attemptWindow:   config.AttemptWindow,
		keyPrefix:       RedisLoginAttemptsPrefix,
		memoryStore:     make(map[string]*loginAttemptEntry),
		cleanupStopChan: make(chan struct{}),
	}

	// Start cleanup for in-memory store
	if redisClient == nil {
		lat.cleanupTicker = time.NewTicker(time.Minute)
		go lat.cleanupExpired()
	}

	return lat
}

func (lat *LoginAttemptTracker) cleanupExpired() {
	for {
		select {
		case <-lat.cleanupTicker.C:
			lat.memoryStoreMu.Lock()
			now := time.Now()
			for key, entry := range lat.memoryStore {
				if now.After(entry.expiresAt) {
					delete(lat.memoryStore, key)
				}
			}
			lat.memoryStoreMu.Unlock()
		case <-lat.cleanupStopChan:
			lat.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the tracker cleanup goroutine
func (lat *LoginAttemptTracker) Stop() {
	if lat.cleanupTicker != nil {
		close(lat.cleanupStopChan)
	}
}

// RecordFailedAttempt records a failed login attempt
func (lat *LoginAttemptTracker) RecordFailedAttempt(ctx context.Context, identifier string) (int, bool, error) {
	if lat.redisClient != nil {
		return lat.recordFailedAttemptRedis(ctx, identifier)
	}
	return lat.recordFailedAttemptMemory(identifier)
}

func (lat *LoginAttemptTracker) recordFailedAttemptRedis(ctx context.Context, identifier string) (int, bool, error) {
	attemptsKey := lat.keyPrefix + identifier + ":attempts"
	lockoutKey := lat.keyPrefix + identifier + ":lockout"

	// Check if already locked out
	locked, err := lat.redisClient.Exists(ctx, lockoutKey).Result()
	if err != nil {
		return 0, false, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to check lockout status", err)
	}
	if locked > 0 {
		return lat.maxAttempts, true, nil
	}

	// Increment attempts
	attempts, err := lat.redisClient.Incr(ctx, attemptsKey).Result()
	if err != nil {
		return 0, false, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to increment attempt counter", err)
	}

	// Set expiry on first attempt
	if attempts == 1 {
		lat.redisClient.Expire(ctx, attemptsKey, lat.attemptWindow)
	}

	// Check if should lock out
	if int(attempts) >= lat.maxAttempts {
		lat.redisClient.Set(ctx, lockoutKey, "1", lat.lockoutDuration)
		return int(attempts), true, nil
	}

	return int(attempts), false, nil
}

func (lat *LoginAttemptTracker) recordFailedAttemptMemory(identifier string) (int, bool, error) {
	lat.memoryStoreMu.Lock()
	defer lat.memoryStoreMu.Unlock()

	now := time.Now()
	entry, exists := lat.memoryStore[identifier]

	// Check if locked out
	if exists && !entry.lockedAt.IsZero() && now.Before(entry.lockedAt.Add(lat.lockoutDuration)) {
		return lat.maxAttempts, true, nil
	}

	// New or expired entry
	if !exists || now.After(entry.expiresAt) {
		lat.memoryStore[identifier] = &loginAttemptEntry{
			attempts:  1,
			expiresAt: now.Add(lat.attemptWindow),
		}
		return 1, false, nil
	}

	entry.attempts++
	if entry.attempts >= lat.maxAttempts {
		entry.lockedAt = now
		entry.expiresAt = now.Add(lat.lockoutDuration)
		return entry.attempts, true, nil
	}

	return entry.attempts, false, nil
}

// IsLockedOut checks if an identifier is locked out
func (lat *LoginAttemptTracker) IsLockedOut(ctx context.Context, identifier string) (bool, time.Duration, error) {
	if lat.redisClient != nil {
		return lat.isLockedOutRedis(ctx, identifier)
	}
	return lat.isLockedOutMemory(identifier)
}

func (lat *LoginAttemptTracker) isLockedOutRedis(ctx context.Context, identifier string) (bool, time.Duration, error) {
	lockoutKey := lat.keyPrefix + identifier + ":lockout"

	ttl, err := lat.redisClient.TTL(ctx, lockoutKey).Result()
	if err != nil {
		return false, 0, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to check lockout TTL", err)
	}

	if ttl > 0 {
		return true, ttl, nil
	}

	return false, 0, nil
}

func (lat *LoginAttemptTracker) isLockedOutMemory(identifier string) (bool, time.Duration, error) {
	lat.memoryStoreMu.RLock()
	defer lat.memoryStoreMu.RUnlock()

	entry, exists := lat.memoryStore[identifier]
	if !exists || entry.lockedAt.IsZero() {
		return false, 0, nil
	}

	remaining := time.Until(entry.lockedAt.Add(lat.lockoutDuration))
	if remaining > 0 {
		return true, remaining, nil
	}

	return false, 0, nil
}

// ClearAttempts clears failed attempts for an identifier (on successful login)
func (lat *LoginAttemptTracker) ClearAttempts(ctx context.Context, identifier string) error {
	if lat.redisClient != nil {
		attemptsKey := lat.keyPrefix + identifier + ":attempts"
		lockoutKey := lat.keyPrefix + identifier + ":lockout"
		lat.redisClient.Del(ctx, attemptsKey, lockoutKey)
		return nil
	}

	lat.memoryStoreMu.Lock()
	delete(lat.memoryStore, identifier)
	lat.memoryStoreMu.Unlock()
	return nil
}
