package core

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyManager defines a general-purpose key-value storage interface.
//
// This interface is used by plugins for storing temporary data:
//   - OAuth state tokens (OAuth plugin)
//   - CSRF tokens (CSRF protection)
//   - Email verification codes (EmailOTP plugin)
//   - SMS verification codes (SMS plugin)
//   - JWT refresh token blacklists (JWT plugin)
//
// Implementations:
//   - StaticKeyManager: In-memory storage (development/testing)
//   - RedisKeyManager: Redis-backed storage (production)
//
// Unlike SessionService which uses Redis for session caching, KeyManager is
// a general-purpose abstraction that plugins can use for any temporary data.
//
// Example (OAuth state storage):
//
//	keyManager.Set(ctx, "oauth:state:"+state, []byte(redirectURL), 10*time.Minute)
//	redirectURL, _ := keyManager.Get(ctx, "oauth:state:"+state)
//	keyManager.Delete(ctx, "oauth:state:"+state)
type KeyManager interface {
	// Get retrieves a value by key
	// Returns error if key doesn't exist or retrieval fails
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with optional expiry
	// expiry=0 means no expiration (permanent storage)
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error

	// Delete removes a value by key
	Delete(ctx context.Context, key string) error
}

// StaticKeyManager provides in-memory key-value storage.
//
// WARNING: This is NOT suitable for production use because:
//   - Data is lost on server restart
//   - Not shared across multiple server instances
//   - No persistence or durability
//
// Use this for:
//   - Local development and testing
//   - Single-server deployments with non-critical data
//   - Unit tests that need fast in-memory storage
//
// For production, use RedisKeyManager instead.
//
// Note: Unlike Redis, expiry is NOT supported - all entries persist until
// manually deleted or the server restarts.
type StaticKeyManager struct {
	// storage is the in-memory map (not thread-safe, use with care)
	storage map[string][]byte
}

// NewStaticKeyManager creates a new in-memory key manager.
//
// This manager stores all data in a Go map with no persistence or expiry.
// Data is lost when the server stops.
//
// Example:
//
//	keyManager, _ := core.NewStaticKeyManager()
//	keyManager.Set(ctx, "key", []byte("value"), 0)
func NewStaticKeyManager() (*StaticKeyManager, error) {
	return &StaticKeyManager{
		storage: make(map[string][]byte),
	}, nil
}

// Get retrieves a value from in-memory storage.
//
// Returns an error if the key doesn't exist.
func (m *StaticKeyManager) Get(_ context.Context, key string) ([]byte, error) {
	value, exists := m.storage[key]
	if !exists {
		return nil, NewAuthError(AuthErrorCodeInternal, fmt.Sprintf("key not found: %s", key))
	}
	return value, nil
}

// Set stores a value in in-memory storage.
//
// WARNING: The expiry parameter is IGNORED by StaticKeyManager. All entries
// persist until manually deleted or the server restarts.
//
// For automatic expiry support, use RedisKeyManager instead.
func (m *StaticKeyManager) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.storage[key] = value
	return nil
}

// Delete removes a value from in-memory storage.
// Does nothing if the key doesn't exist.
func (m *StaticKeyManager) Delete(_ context.Context, key string) error {
	delete(m.storage, key)
	return nil
}

// RedisKeyManager provides Redis-backed key-value storage with automatic expiration.
//
// This is the recommended KeyManager implementation for production because:
//   - Data persists across server restarts (if Redis is configured for persistence)
//   - Shared across multiple server instances
//   - Automatic expiry with TTL support
//   - High performance with low latency
//
// Use this for:
//   - Production deployments
//   - Distributed systems with multiple servers
//   - Any data that needs automatic expiry (OAuth states, OTP codes, etc.)
//
// Example:
//
//	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	keyManager := core.NewRedisKeyManager(redisClient)
type RedisKeyManager struct {
	client *redis.Client
}

// NewRedisKeyManager creates a new Redis-backed key manager.
//
// The provided Redis client should be already configured and connected.
// The KeyManager will use the client for all operations.
//
// Example:
//
//	redisClient := redis.NewClient(&redis.Options{
//		Addr:     "localhost:6379",
//		Password: "",
//		DB:       0,
//	})
//	keyManager := core.NewRedisKeyManager(redisClient)
func NewRedisKeyManager(client *redis.Client) *RedisKeyManager {
	return &RedisKeyManager{
		client: client,
	}
}

// Get retrieves a value from Redis.
//
// Returns an error if:
//   - Key doesn't exist (redis.Nil → AuthErrorCodeInternal)
//   - Redis operation fails (network error, etc.)
//
// Example:
//
//	value, err := keyManager.Get(ctx, "oauth:state:abc123")
//	if err != nil {
//		// Key doesn't exist or Redis error
//	}
func (m *RedisKeyManager) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := m.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, NewAuthError(AuthErrorCodeInternal, fmt.Sprintf("key not found: %s", key))
		}
		return nil, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to get key from redis", err)
	}
	return []byte(result), nil
}

// Set stores a value in Redis with optional automatic expiration.
//
// The expiry parameter controls key lifetime:
//   - expiry > 0: Key expires after the specified duration
//   - expiry = 0: Key never expires (persists indefinitely)
//
// Redis will automatically delete expired keys using its eviction policy.
//
// Example:
//
//	// OAuth state valid for 10 minutes
//	keyManager.Set(ctx, "oauth:state:abc123", stateData, 10*time.Minute)
//
//	// Email verification code valid for 1 hour
//	keyManager.Set(ctx, "email:verify:user@example.com", []byte("123456"), time.Hour)
func (m *RedisKeyManager) Set(ctx context.Context, key string, value []byte, expiry time.Duration) error {
	return m.client.Set(ctx, key, value, expiry).Err()
}

// Delete removes a value from Redis.
//
// This is idempotent - deleting a non-existent key does nothing and returns no error.
//
// Example:
//
//	// Delete OAuth state after use to prevent replay attacks
//	keyManager.Delete(ctx, "oauth:state:abc123")
func (m *RedisKeyManager) Delete(ctx context.Context, key string) error {
	return m.client.Del(ctx, key).Err()
}
