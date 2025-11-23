package core

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	accessKeysPrefix  = "auth:jwt:keys:access"
	refreshKeysPrefix = "auth:jwt:keys:refresh"
	currentKeySuffix  = "auth:current"

	// Key rotation settings
	keyExpiryDuration = 14 * 24 * time.Hour // 14 days
)

// KeyManager defines the interface for managing JWT keys
type KeyManager interface {
	GetAccessKey(ctx context.Context) (jwk.Key, error)
	GetRefreshKey(ctx context.Context) (jwk.Key, error)
	ValidateAccessKey(ctx context.Context, tokenKeyID string) (jwk.Key, error)
	ValidateRefreshKey(ctx context.Context, tokenKeyID string) (jwk.Key, error)
}

// StaticKeyManager uses a single static RSA key pair derived from a secret (or generated once)
// Note: For true static keys, we'd need to load them from PEM/file.
// Here we generate one on startup if not provided, effectively making it ephemeral per restart if not persisted.
// To support the requirement "derived from secret", we can use the secret to seed the RNG, but RSA generation needs random source.
// Better approach for "Static" without Redis: Generate one on startup and keep it in memory.
// This means tokens are invalidated on restart, which is acceptable for "No Redis" mode usually, or we can persist to DB.
// For this implementation, we'll generate in-memory keys.
type StaticKeyManager struct {
	accessKey  jwk.Key
	refreshKey jwk.Key
}

func NewStaticKeyManager() (*StaticKeyManager, error) {
	// Generate keys
	accessKey, err := generateRSAKey("access-static")
	if err != nil {
		return nil, err
	}
	refreshKey, err := generateRSAKey("refresh-static")
	if err != nil {
		return nil, err
	}

	return &StaticKeyManager{
		accessKey:  accessKey,
		refreshKey: refreshKey,
	}, nil
}

func (m *StaticKeyManager) GetAccessKey(ctx context.Context) (jwk.Key, error) {
	return m.accessKey, nil
}

func (m *StaticKeyManager) GetRefreshKey(ctx context.Context) (jwk.Key, error) {
	return m.refreshKey, nil
}

func (m *StaticKeyManager) ValidateAccessKey(ctx context.Context, tokenKeyID string) (jwk.Key, error) {
	// For static manager, we only have one key.
	// In a real static setup, we might check ID, but here we assume valid if signature matches.
	// However, jwx verification needs the key.
	return m.accessKey, nil
}

func (m *StaticKeyManager) ValidateRefreshKey(ctx context.Context, tokenKeyID string) (jwk.Key, error) {
	return m.refreshKey, nil
}

// RedisKeyManager manages keys in Redis with rotation
type RedisKeyManager struct {
	client *redis.Client
}

func NewRedisKeyManager(client *redis.Client) *RedisKeyManager {
	return &RedisKeyManager{
		client: client,
	}
}

func (m *RedisKeyManager) GetAccessKey(ctx context.Context) (jwk.Key, error) {
	return m.getOrCreateKeyPair(ctx, "access")
}

func (m *RedisKeyManager) GetRefreshKey(ctx context.Context) (jwk.Key, error) {
	return m.getOrCreateKeyPair(ctx, "refresh")
}

func (m *RedisKeyManager) ValidateAccessKey(ctx context.Context, tokenKeyID string) (jwk.Key, error) {
	return m.getKeyByID(ctx, "access", tokenKeyID)
}

func (m *RedisKeyManager) ValidateRefreshKey(ctx context.Context, tokenKeyID string) (jwk.Key, error) {
	return m.getKeyByID(ctx, "refresh", tokenKeyID)
}

func (m *RedisKeyManager) getOrCreateKeyPair(ctx context.Context, keyType string) (jwk.Key, error) {
	prefix := m.getKeyPrefix(keyType)
	currentKeyKey := prefix + ":" + currentKeySuffix

	// Try to get existing key
	keyData, err := m.client.Get(ctx, currentKeyKey).Result()
	if err == nil && keyData != "" {
		key, err := jwk.ParseKey([]byte(keyData))
		if err == nil {
			return key, nil
		}
	}

	// Generate new key
	return m.rotateKeyPair(ctx, keyType)
}

func (m *RedisKeyManager) rotateKeyPair(ctx context.Context, keyType string) (jwk.Key, error) {
	// Generate new RSA key
	key, err := generateRSAKey(keyType)
	if err != nil {
		return nil, err
	}

	keyID, _ := key.KeyID()
	prefix := m.getKeyPrefix(keyType)

	// Serialize
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	// Store specific key
	if err := m.client.Set(ctx, prefix+":"+keyID, keyJSON, keyExpiryDuration).Err(); err != nil {
		return nil, fmt.Errorf("failed to store key: %w", err)
	}

	// Update current key pointer
	if err := m.client.Set(ctx, prefix+":"+currentKeySuffix, keyJSON, 0).Err(); err != nil {
		return nil, fmt.Errorf("failed to update current key: %w", err)
	}

	return key, nil
}

func (m *RedisKeyManager) getKeyByID(ctx context.Context, keyType, keyID string) (jwk.Key, error) {
	prefix := m.getKeyPrefix(keyType)
	keyData, err := m.client.Get(ctx, prefix+":"+keyID).Result()
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}

	key, err := jwk.ParseKey([]byte(keyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	return key, nil
}

func (m *RedisKeyManager) getKeyPrefix(keyType string) string {
	switch keyType {
	case "access":
		return accessKeysPrefix
	case "refresh":
		return refreshKeysPrefix
	default:
		return "auth:jwt:keys:" + keyType
	}
}

// Helper to generate RSA key
func generateRSAKey(keyType string) (jwk.Key, error) {
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	key, err := jwk.Import(raw)
	if err != nil {
		return nil, err
	}

	keyID := fmt.Sprintf("%s-%d", keyType, time.Now().Unix())
	if err := key.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, err
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		return nil, err
	}

	return key, nil
}
