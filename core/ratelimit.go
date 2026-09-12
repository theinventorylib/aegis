package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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

	// FailClosed controls behavior when the backing store (e.g. Redis)
	// returns an error from Allow().
	//
	//   - false (default): fail-open. The request is allowed and the error
	//     is logged at WARN level. Preserves availability if Redis is flaky
	//     but a determined attacker can defeat brute-force protection by
	//     deliberately straining the backing store.
	//   - true: fail-closed. The request is rejected with HTTP 429. Stronger
	//     security guarantee at the cost of availability when the backing
	//     store is unhealthy. Recommended for high-value auth endpoints.
	FailClosed bool

	// TrustedProxies is a list of CIDR ranges (or single IPs in CIDR form,
	// e.g. "10.0.0.5/32") whose X-Forwarded-For / X-Real-IP headers are
	// honored. When the request's RemoteAddr is NOT within one of these
	// ranges, proxy headers are ignored and the raw RemoteAddr is used as
	// the client IP. Default empty = no proxy is trusted (most secure).
	//
	// Set this to your real proxy/load-balancer addresses in production
	// when running behind a reverse proxy that sets these headers.
	TrustedProxies []string
}

// RateLimiter provides distributed rate limiting functionality.
//
// It uses Redis for distributed rate limiting across multiple application instances,
// with an in-memory fallback for single-instance deployments. The sliding window
// algorithm prevents bursty traffic from overwhelming the system.
//
// expiryHolder lets expiringStore know when an entry may be dropped.
type expiryHolder interface{ expiry() time.Time }

// expiringStore is a mutex-guarded map with per-entry expiry plus a periodic
// cleanup goroutine for single-instance (no Redis) mode. Shared by
// RateLimiter and LoginAttemptTracker, which previously duplicated this
// machinery verbatim.
type expiringStore[V expiryHolder] struct {
	mu       sync.RWMutex
	entries  map[string]V
	ticker   *time.Ticker
	stopCh   chan struct{}
	stopOnce sync.Once
}

// newExpiringStore creates the store and starts the cleanup goroutine.
func newExpiringStore[V expiryHolder]() *expiringStore[V] {
	s := &expiringStore[V]{entries: make(map[string]V), stopCh: make(chan struct{})}
	s.ticker = time.NewTicker(time.Minute)
	go s.cleanupExpired()
	return s
}

// stop halts the cleanup goroutine. Safe to call multiple times and on a
// nil store (which exists when Redis mode skips the in-memory fallback).
func (s *expiringStore[V]) stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			close(s.stopCh)
		}
	})
}

func (s *expiringStore[V]) cleanupExpired() {
	for {
		select {
		case <-s.ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, v := range s.entries {
				if now.After(v.expiry()) {
					delete(s.entries, k)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			s.ticker.Stop()
			return
		}
	}
}

// update runs fn with the write lock held (read-modify-write).
func (s *expiringStore[V]) update(fn func(m map[string]V)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.entries)
}

// view runs fn with the read lock held.
func (s *expiringStore[V]) view(fn func(m map[string]V)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.entries)
}

// RateLimiter is safe for concurrent use and should be shared across HTTP handlers.
type RateLimiter struct {
	// config holds rate limiting settings
	config *RateLimitConfig

	// redisClient enables distributed limiting. If nil, uses in-memory fallback.
	redisClient *redis.Client

	// auditLogger records rate limit violations
	auditLogger AuditLogger

	// logger receives diagnostic messages (backing-store errors, etc.).
	// May be nil; helpers handle that.
	logger Logger

	// trustedProxyNets is the parsed form of config.TrustedProxies. Built
	// once at construction so the hot path doesn't re-parse CIDRs.
	trustedProxyNets []*net.IPNet

	// In-memory fallback for when Redis is not available.
	memory *expiringStore[rateLimitEntry]
}

// rateLimitEntry tracks request counts in the in-memory store.
type rateLimitEntry struct {
	// count is the number of requests in the current window
	count int

	// expiresAt is when this window ends and the counter resets
	expiresAt time.Time
}

func (e rateLimitEntry) expiry() time.Time { return e.expiresAt }

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
	memory *expiringStore[loginAttemptEntry]
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

func (e loginAttemptEntry) expiry() time.Time { return e.expiresAt }

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

// NewRateLimiter creates a new rate limiter with the specified configuration.
//
// Parameters:
//   - config: Rate limiting configuration. If nil, uses defaults.
//   - redisClient: Redis client for distributed rate limiting. If nil, uses
//     in-memory fallback (only suitable for single-instance deployments).
//   - auditLogger: Logger for recording rate limit violations. If nil, uses
//     a no-op logger.
//   - logger: Optional diagnostic logger. May be nil.
//
// The returned RateLimiter is safe for concurrent use. For multi-instance
// deployments, a Redis client must be provided to enforce limits across
// all instances.
func NewRateLimiter(config *RateLimitConfig, redisClient *redis.Client, auditLogger AuditLogger, logger Logger) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}
	if logger == nil {
		logger = noopLogger{}
	}

	rl := &RateLimiter{
		config:           config,
		redisClient:      redisClient,
		auditLogger:      auditLogger,
		logger:           logger,
		trustedProxyNets: parseTrustedProxies(config.TrustedProxies, logger),
	}

	// Start cleanup goroutine for in-memory store
	if redisClient == nil {
		rl.memory = newExpiringStore[rateLimitEntry]()
	}

	return rl
}

// Stop stops the rate limiter cleanup goroutine. Safe to call multiple times.
func (rl *RateLimiter) Stop() {
	rl.memory.stop()
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int, error) {
	if rl.redisClient != nil {
		return rl.allowRedis(ctx, key)
	}
	return rl.allowMemory(key)
}

// allowRedis uses Redis for distributed rate limiting.
//
// On a backing-store error, behavior depends on rl.config.FailClosed:
//   - false (default): allow the request and signal the error to the
//     middleware so it can be logged. We deliberately do not bubble the
//     error up to the application.
//   - true: deny the request (return false). The middleware will return 429.
func (rl *RateLimiter) allowRedis(ctx context.Context, key string) (bool, int, error) {
	redisKey := rl.config.KeyPrefix + key

	// Initialize the counter with its window TTL atomically (SET NX EX).
	// INCR alone + conditional EXPIRE is not atomic: if the process dies
	// between INCR and EXPIRE the key persists without a TTL and the
	// client is rate-limited forever after crossing the limit.
	window := rl.config.WindowDuration
	created, err := rl.redisClient.SetNX(ctx, redisKey, 0, window).Result()
	if err != nil {
		if rl.config.FailClosed {
			return false, 0, err
		}
		return true, 0, err
	}

	count, err := rl.redisClient.Incr(ctx, redisKey).Result()
	if err != nil {
		if rl.config.FailClosed {
			return false, 0, err
		}
		return true, 0, err
	}

	// Safety net: if the key vanished between SetNX and INCR (race with
	// expiry), the INCR re-created it without a TTL — re-apply it.
	if !created {
		ttl, err := rl.redisClient.TTL(ctx, redisKey).Result()
		if err == nil && ttl < 0 {
			rl.redisClient.Expire(ctx, redisKey, window)
		}
	}

	remaining := max(rl.config.RequestsPerWindow-int(count), 0)

	return int(count) <= rl.config.RequestsPerWindow, remaining, nil
}

// allowMemory uses in-memory store for single-instance rate limiting
func (rl *RateLimiter) allowMemory(key string) (bool, int, error) {
	var allowed bool
	var remaining int
	rl.memory.update(func(m map[string]rateLimitEntry) {
		now := time.Now()
		fullKey := rl.config.KeyPrefix + key

		entry, exists := m[fullKey]
		if !exists || now.After(entry.expiresAt) {
			// New window
			m[fullKey] = rateLimitEntry{
				count:     1,
				expiresAt: now.Add(rl.config.WindowDuration),
			}
			allowed, remaining = true, rl.config.RequestsPerWindow-1
			return
		}

		entry.count++
		m[fullKey] = entry
		remaining = max(rl.config.RequestsPerWindow-entry.count, 0)
		allowed = entry.count <= rl.config.RequestsPerWindow
	})
	return allowed, remaining, nil
}

// parseTrustedProxies converts a list of CIDR strings into *net.IPNet values.
// Invalid entries are logged and skipped. A bare IP (no "/") is automatically
// treated as a /32 (IPv4) or /128 (IPv6) so callers can write
// "10.0.0.5" instead of "10.0.0.5/32".
func parseTrustedProxies(cidrs []string, logger Logger) []*net.IPNet {
	if len(cidrs) == 0 {
		return nil
	}
	if logger == nil {
		logger = noopLogger{}
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		entry := raw
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		_, ipNet, err := net.ParseCIDR(entry)
		if err != nil {
			logger.Error("rate limiter: ignoring invalid TrustedProxies entry",
				"value", raw, "error", err)
			continue
		}
		out = append(out, ipNet)
	}
	return out
}

// isTrustedProxy reports whether ipStr is within any of the configured
// trustedProxyNets. Returns false for an empty allowlist (the secure default).
func isTrustedProxy(ipStr string, trusted []*net.IPNet) bool {
	if len(trusted) == 0 {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIPFromRequest returns the best-guess client IP for r.
//
// X-Forwarded-For / X-Real-IP are honored ONLY when the immediate peer
// (r.RemoteAddr) is in the trustedProxyNets allowlist. Otherwise the raw
// remote-address IP is returned. This prevents an arbitrary client from
// spoofing their IP for rate-limit / audit purposes by sending forged
// proxy headers.
func clientIPFromRequest(r *http.Request, trusted []*net.IPNet) string {
	remote := GetClientIP(r)

	if !isTrustedProxy(remote, trusted) {
		return remote
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For: "client, proxy1, proxy2". Each proxy APPENDS the
		// address it received from, so the leftmost entry is fully
		// client-controlled (spoofable when the client's own proxy chain is
		// trusted). Walk from the right: the rightmost value is what the
		// trusted peer saw; keep skipping left while entries are trusted
		// proxies, and return the first untrusted hop.
		parts := strings.Split(xff, ",")
		for i, part := range slices.Backward(parts) {
			candidate := strings.TrimSpace(part)
			if candidate == "" {
				continue
			}
			if i > 0 && isTrustedProxy(candidate, trusted) {
				continue
			}
			return candidate
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	return remote
}

// getClientIP extracts the client IP from the request, honoring proxy
// headers only for peers in the rate-limiter's TrustedProxies allowlist.
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	return clientIPFromRequest(r, rl.trustedProxyNets)
}

// RateLimitMiddleware creates a middleware that rate limits requests
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded
			if slices.Contains(limiter.config.ExcludePaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Build rate limit key
			var key string
			if limiter.config.ByUser {
				// Try to get user from context
				if user, err := GetUser(r.Context()); err == nil && user != nil {
					if uid := UserIDOf(user); uid != "" {
						key = "user:" + uid
					}
				}
			}

			if key == "" && limiter.config.ByIP {
				key = "ip:" + limiter.getClientIP(r)
			}

			if key == "" {
				// No key could be determined, allow request
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			safePath := strings.ReplaceAll(r.URL.Path, "\n", "")
			safePath = strings.ReplaceAll(safePath, "\r", "")

			allowed, remaining, err := limiter.Allow(r.Context(), key)
			if err != nil {
				// Backing-store error (typically Redis). Log it; behavior
				// depends on FailClosed (already applied inside Allow):
				//   - fail-open: allowed=true, request proceeds
				//   - fail-closed: allowed=false, request is rejected below
				limiter.logger.Error("rate limiter: backing-store error",
					"path", safePath,
					"method", r.Method,
					"fail_closed", limiter.config.FailClosed,
					"error", err)
				if !limiter.config.FailClosed {
					next.ServeHTTP(w, r)
					return
				}
				// fall through; allowed==false → 429 below
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
					"key_hash":  hashShort(keyVal),
					"path":      safePath,
					"method":    r.Method,
					"remaining": remaining,
				})

				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(limiter.config.WindowDuration.Seconds())))
				WriteJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
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
	}

	// Start cleanup for in-memory store
	if redisClient == nil {
		lat.memory = newExpiringStore[loginAttemptEntry]()
	}

	return lat
}

// Stop stops the tracker cleanup goroutine. Safe to call multiple times.
func (lat *LoginAttemptTracker) Stop() {
	lat.memory.stop()
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
	var attempts int
	var lockedOut bool

	lat.memory.update(func(m map[string]loginAttemptEntry) {
		now := time.Now()
		entry, exists := m[identifier]

		// Check if locked out
		if exists && !entry.lockedAt.IsZero() && now.Before(entry.lockedAt.Add(lat.lockoutDuration)) {
			attempts, lockedOut = lat.maxAttempts, true
			return
		}

		// New or expired entry
		if !exists || now.After(entry.expiresAt) {
			m[identifier] = loginAttemptEntry{
				attempts:  1,
				expiresAt: now.Add(lat.attemptWindow),
			}
			attempts = 1
			return
		}

		entry.attempts++
		if entry.attempts >= lat.maxAttempts {
			entry.lockedAt = now
			entry.expiresAt = now.Add(lat.lockoutDuration)
			m[identifier] = entry
			attempts, lockedOut = entry.attempts, true
			return
		}

		m[identifier] = entry
		attempts = entry.attempts
	})

	return attempts, lockedOut, nil
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
	var locked bool
	var remaining time.Duration
	lat.memory.view(func(m map[string]loginAttemptEntry) {
		entry, exists := m[identifier]
		if !exists || entry.lockedAt.IsZero() {
			return
		}
		remaining = time.Until(entry.lockedAt.Add(lat.lockoutDuration))
		locked = remaining > 0
	})
	return locked, remaining, nil
}

// ClearAttempts clears failed attempts for an identifier (on successful login)
func (lat *LoginAttemptTracker) ClearAttempts(ctx context.Context, identifier string) error {
	if lat.redisClient != nil {
		attemptsKey := lat.keyPrefix + identifier + ":attempts"
		lockoutKey := lat.keyPrefix + identifier + ":lockout"
		lat.redisClient.Del(ctx, attemptsKey, lockoutKey)
		return nil
	}

	lat.memory.update(func(m map[string]loginAttemptEntry) { delete(m, identifier) })
	return nil
}
