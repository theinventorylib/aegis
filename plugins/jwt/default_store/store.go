// Package defaultstore implements the SQL-backed default store for the jwt plugin.
package defaultstore

// store.go — DefaultJWTStore: the dialect switch happens once (in the constructor);
// all methods are dialect-agnostic and delegate to the querier interface.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/theinventorylib/aegis/plugins"
	jwttypes "github.com/theinventorylib/aegis/plugins/jwt/types"
)

// DefaultJWTStore implements jwttypes.Store using a SQL database backend.
//
// Supports PostgreSQL, MySQL, and SQLite through dialect-specific sqlc-generated
// queries. Safe for concurrent use.
type DefaultJWTStore struct{ q querier }

// NewDefaultJWTStore creates a DefaultJWTStore for the given dialect.
// The dialect switch happens exactly once here; all store methods call
// through the querier interface and are dialect-agnostic.
func NewDefaultJWTStore(db *sql.DB, dialect plugins.Dialect) (*DefaultJWTStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMySQLQuerier(db)
	case plugins.DialectSQLite:
		q = newSQLiteQuerier(db)
	default:
		return nil, fmt.Errorf("jwt: unsupported dialect %q", dialect)
	}
	return &DefaultJWTStore{q: q}, nil
}

// GetCurrentJWK retrieves the most recent non-expired JWK matching algorithm and use.
func (s *DefaultJWTStore) GetCurrentJWK(ctx context.Context, algorithm, use string) (jwk.Key, error) {
	useParam := sql.NullString{String: use, Valid: true}
	keyData, err := s.q.getCurrentJWK(ctx, algorithm, useParam)
	if err != nil {
		return nil, err
	}
	return jwk.ParseKey([]byte(keyData))
}

// StoreJWK persists a JWK to the database with optional expiration.
func (s *DefaultJWTStore) StoreJWK(ctx context.Context, key jwk.Key, algorithm, use string, expiresAt *time.Time) error {
	keyData, err := json.Marshal(key)
	if err != nil {
		return err
	}
	kid, _ := key.KeyID()
	now := time.Now().Format(time.RFC3339)
	useParam := sql.NullString{String: use, Valid: true}
	var ea sql.NullString
	if expiresAt != nil {
		ea = sql.NullString{String: expiresAt.Format(time.RFC3339), Valid: true}
	}
	return s.q.storeJWK(ctx, kid, string(keyData), algorithm, useParam, now, ea)
}

// DeleteExpiredJWKS removes all expired keys from the database.
func (s *DefaultJWTStore) DeleteExpiredJWKS(ctx context.Context) error {
	return s.q.deleteExpiredJWKS(ctx)
}

// ListJWKS retrieves all non-expired JWKs from the database.
func (s *DefaultJWTStore) ListJWKS(ctx context.Context) ([]jwttypes.JWK, error) {
	rows, err := s.q.getAllCurrentJWKS(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]jwttypes.JWK, len(rows))
	for i, r := range rows {
		var expiresAt *time.Time
		if r.ExpiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, r.ExpiresAt.String) //nolint:errcheck
			expiresAt = &t
		}
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt) //nolint:errcheck
		result[i] = jwttypes.JWK{
			Kid:       r.Kid,
			KeyData:   []byte(r.KeyData),
			Algorithm: r.Algorithm,
			Use:       r.Use.String,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		}
	}
	return result, nil
}
