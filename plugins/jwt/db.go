package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/theinventorylib/aegis/db"
)

// DB handles database operations for the JWT plugin.
type DB struct {
	provider db.Provider
}

// NewDB creates a new database handler for JWT operations.
func NewDB(provider db.Provider) *DB {
	return &DB{provider: provider}
}

// CreateJWK stores a new JWK in the database.
func (d *DB) CreateJWK(ctx context.Context, jwkKey jwk.Key, keyType string, use string, expiresAt *time.Time) error {
	kid, _ := jwkKey.KeyID()

	// Marshal the JWK to JSON
	keyJSON, err := json.Marshal(jwkKey)
	if err != nil {
		return fmt.Errorf("failed to marshal JWK: %w", err)
	}

	alg, _ := jwkKey.Algorithm()
	algorithm := keyType // Default to keyType
	if alg != nil {
		algorithm = alg.String()
	}

	query := `
		INSERT INTO jwt.jwks (kid, key_data, algorithm, use, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kid) DO UPDATE 
		SET key_data = $2, algorithm = $3, use = $4, expires_at = $5
	`

	_, err = d.provider.Exec(ctx, query, kid, keyJSON, algorithm, use, expiresAt)
	return err
}

// GetJWK retrieves a specific JWK by its kid.
func (d *DB) GetJWK(ctx context.Context, kid string) (jwk.Key, error) {
	query := `
		SELECT key_data FROM jwt.jwks 
		WHERE kid = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	row := d.provider.QueryRow(ctx, query, kid)

	var keyData []byte
	if err := row.Scan(&keyData); err != nil {
		return nil, fmt.Errorf("failed to get JWK: %w", err)
	}

	key, err := jwk.ParseKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWK: %w", err)
	}

	return key, nil
}

// GetCurrentJWK retrieves the most recent non-expired key for a specific algorithm and use.
func (d *DB) GetCurrentJWK(ctx context.Context, algorithm string, use string) (jwk.Key, error) {
	query := `
		SELECT key_data FROM jwt.jwks 
		WHERE algorithm = $1 AND use = $2 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC 
		LIMIT 1
	`

	row := d.provider.QueryRow(ctx, query, algorithm, use)

	var keyData []byte
	if err := row.Scan(&keyData); err != nil {
		return nil, fmt.Errorf("failed to get current JWK: %w", err)
	}

	key, err := jwk.ParseKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWK: %w", err)
	}

	return key, nil
}

// ListJWKS retrieves all non-expired JWKs (for JWKS endpoint).
func (d *DB) ListJWKS(ctx context.Context) ([]jwk.Key, error) {
	query := `
		SELECT key_data FROM jwt.jwks 
		WHERE expires_at IS NULL OR expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := d.provider.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list JWKs: %w", err)
	}
	defer rows.Close()

	var keys []jwk.Key
	for rows.Next() {
		var keyData []byte
		if err := rows.Scan(&keyData); err != nil {
			return nil, fmt.Errorf("failed to scan JWK: %w", err)
		}

		key, err := jwk.ParseKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JWK: %w", err)
		}

		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return keys, nil
}

// DeleteExpiredJWKS removes all expired keys from the database.
func (d *DB) DeleteExpiredJWKS(ctx context.Context) error {
	query := `DELETE FROM jwt.jwks WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	_, err := d.provider.Exec(ctx, query)
	return err
}

// UpdateJWKExpiry marks a key for expiration.
func (d *DB) UpdateJWKExpiry(ctx context.Context, kid string, expiresAt time.Time) error {
	query := `UPDATE jwt.jwks SET expires_at = $1 WHERE kid = $2`
	_, err := d.provider.Exec(ctx, query, expiresAt, kid)
	return err
}
