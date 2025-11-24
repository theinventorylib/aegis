package core

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyManager defines the interface for generic key-value storage.
// This is a general-purpose interface that plugins can use for managing keys.
type KeyManager interface {
	// Generic key-value operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error
	Delete(ctx context.Context, key string) error
}

// StaticKeyManager provides in-memory key-value storage.
// This is useful for development or when no persistence is needed.
type StaticKeyManager struct {
	storage map[string][]byte // In-memory storage
}

// NewStaticKeyManager creates a new static key manager with in-memory storage.
func NewStaticKeyManager() (*StaticKeyManager, error) {
	return &StaticKeyManager{
		storage: make(map[string][]byte),
	}, nil
}

// Get retrieves a value from in-memory storage.
func (m *StaticKeyManager) Get(_ context.Context, key string) ([]byte, error) {
	value, exists := m.storage[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

// Set stores a value in in-memory storage (expiry is ignored for static manager).
func (m *StaticKeyManager) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.storage[key] = value
	return nil
}

// Delete removes a value from in-memory storage.
func (m *StaticKeyManager) Delete(_ context.Context, key string) error {
	delete(m.storage, key)
	return nil
}

// RedisKeyManager provides Redis-backed key-value storage with expiration support.
type RedisKeyManager struct {
	client *redis.Client
}

// NewRedisKeyManager creates a new Redis-backed key manager.
func NewRedisKeyManager(client *redis.Client) *RedisKeyManager {
	return &RedisKeyManager{
		client: client,
	}
}

// Get retrieves a value from Redis.
func (m *RedisKeyManager) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := m.client.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get key from redis: %w", err)
	}
	return []byte(result), nil
}

// Set stores a value in Redis with optional expiry.
func (m *RedisKeyManager) Set(ctx context.Context, key string, value []byte, expiry time.Duration) error {
	return m.client.Set(ctx, key, value, expiry).Err()
}

// Delete removes a value from Redis.
func (m *RedisKeyManager) Delete(ctx context.Context, key string) error {
	return m.client.Del(ctx, key).Err()
}
