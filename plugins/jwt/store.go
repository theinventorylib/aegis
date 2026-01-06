package jwt

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
)

// Store defines the interface for JWT token and key storage operations.
//
// This interface abstracts database operations for JWK (JSON Web Key) storage,
// allowing different storage backends:
//   - SQL databases: PostgreSQL, MySQL, SQLite (default)
//   - NoSQL databases: MongoDB, DynamoDB (custom implementation)
//   - Cloud key vaults: AWS KMS, Google Cloud KMS (custom implementation)
//
// The default implementation (DefaultStore) uses SQL with sqlc-generated queries.
//
// Key Storage Requirements:
//   - Persistence: Keys must survive application restarts
//   - Multi-server: Keys must be shared across application instances
//   - Key rotation: Support storing multiple active keys
//   - Key expiry: Automatic cleanup of expired keys
//
// Thread Safety:
//
// Implementations must be thread-safe for concurrent access from:
//   - Token generation requests (read current key)
//   - Key rotation background job (write new keys)
//   - Token validation requests (read all active keys)
//   - JWKS endpoint requests (list public keys)
type Store interface {
	// GetCurrentJWK retrieves the most recent active key for signing.
	//
	// This is used during token generation to get the private key for signing.
	// Returns the newest non-expired key matching the algorithm and use.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - algorithm: Key algorithm (e.g., "RS256")
	//   - use: Key use ("sig" for signing, "enc" for encryption)
	//
	// Returns:
	//   - jwk.Key: The most recent active key
	//   - error: Database errors or "key not found"
	//
	// Example:
	//
	//	privateKey, err := store.GetCurrentJWK(ctx, "RS256", "sig")
	//	if err != nil {
	//		// No active key - generate new one
	//	}
	GetCurrentJWK(ctx context.Context, algorithm, use string) (jwk.Key, error)

	// StoreJWK persists a JWK to the database with expiry.
	//
	// This is used during:
	//   - Initial key generation (plugin initialization)
	//   - Key rotation (periodic background job)
	//
	// The key is stored with an expiry time for automatic cleanup.
	// Expired keys are kept to verify old tokens until cleanup.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - key: JWK to store (includes private/public key material)
	//   - algorithm: Key algorithm (e.g., "RS256")
	//   - use: Key use ("sig" or "enc")
	//   - expiresAt: When to delete this key (nil = never expires)
	//
	// Returns:
	//   - error: Database errors or duplicate key ID
	//
	// Example:
	//
	//	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	//	err := store.StoreJWK(ctx, privateKey, "RS256", "sig", &expiresAt)
	StoreJWK(ctx context.Context, key jwk.Key, algorithm, use string, expiresAt *time.Time) error

	// DeleteExpiredJWKS removes all expired keys from storage.
	//
	// This is called periodically to clean up old keys that are no longer needed.
	// Only deletes keys where:
	//   - expires_at IS NOT NULL (keys with expiry set)
	//   - expires_at < NOW() (expiry time has passed)
	//
	// Note: Keys are kept past their "active" period to verify old tokens.
	// The expiry should be set to created_at + KeyRetention duration.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns:
	//   - error: Database errors
	//
	// Example:
	//
	//	if err := store.DeleteExpiredJWKS(ctx); err != nil {
	//		log.Printf("Failed to cleanup expired keys: %v", err)
	//	}
	DeleteExpiredJWKS(ctx context.Context) error

	// ListJWKS returns all active (non-expired) keys.
	//
	// This is used by:
	//   - JWKS endpoint: Expose public keys for token verification
	//   - Token validation: Find key by kid (key ID) for signature verification
	//
	// Only returns keys where:
	//   - expires_at IS NULL (never expires), OR
	//   - expires_at > NOW() (not yet expired)
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns:
	//   - []JWK: List of active keys (includes both public and private keys)
	//   - error: Database errors
	//
	// Example:
	//
	//	keys, err := store.ListJWKS(ctx)
	//	for _, key := range keys {
	//		fmt.Printf("Key: %s, Algorithm: %s\n", key.Kid, key.Algorithm)
	//	}
	ListJWKS(ctx context.Context) ([]JWK, error)
}
