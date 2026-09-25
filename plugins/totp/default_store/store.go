// Package defaultstore implements the SQL-backed default store for the totp plugin.
package defaultstore

// store.go — DefaultTOTPStore: the switch happens once (in the constructor);
// all methods are dialect-agnostic and delegate to the querier interface.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/v2/plugins"
	totptypes "github.com/theinventorylib/aegis/v2/plugins/totp/types"
)

// DefaultTOTPStore implements totptypes.Store using a SQL database.
//
// Supports PostgreSQL, MySQL, and SQLite through dialect-specific sqlc-generated
// queries. Safe for concurrent use.
type DefaultTOTPStore struct{ q querier }

// NewDefaultTOTPStore creates a DefaultTOTPStore for the given dialect.
// The dialect switch happens exactly once here; all store methods call through
// the querier interface and are dialect-agnostic.
func NewDefaultTOTPStore(db *sql.DB, dialect plugins.Dialect) (*DefaultTOTPStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMySQLQuerier(db)
	case plugins.DialectSQLite:
		q = newSQLiteQuerier(db)
	default:
		return nil, fmt.Errorf("totp: unsupported dialect %q", dialect)
	}
	return &DefaultTOTPStore{q: q}, nil
}

// GetCredential returns the user's TOTP credential.
func (s *DefaultTOTPStore) GetCredential(ctx context.Context, userID string) (totptypes.Credential, error) {
	row, err := s.q.getCredential(ctx, userID)
	if err != nil {
		return totptypes.Credential{}, err
	}
	return totptypes.Credential{UserID: userID, Secret: row.Secret, Enabled: row.Enabled}, nil
}

// SetCredential writes the credential; a nil secret clears it.
func (s *DefaultTOTPStore) SetCredential(ctx context.Context, userID string, secret *string, enabled bool) error {
	var stored sql.NullString
	if secret != nil {
		stored = sql.NullString{String: *secret, Valid: true}
	}
	return s.q.updateCredential(ctx, userID, stored, enabled, time.Now().UTC().Format(time.RFC3339))
}

// MarkSessionVerified records a verified session.
func (s *DefaultTOTPStore) MarkSessionVerified(ctx context.Context, sessionID, userID string, at time.Time) error {
	return s.q.markSessionVerified(ctx, sessionID, userID, at.UTC().Format(time.RFC3339))
}

// IsSessionVerified reports whether the session was verified after since.
func (s *DefaultTOTPStore) IsSessionVerified(ctx context.Context, sessionID, userID string, since time.Time) (bool, error) {
	return s.q.isSessionVerified(ctx, sessionID, userID, since.UTC().Format(time.RFC3339))
}

// DeleteUserSessionVerifications clears every session verification for a user.
func (s *DefaultTOTPStore) DeleteUserSessionVerifications(ctx context.Context, userID string) error {
	return s.q.deleteUserSessionVerifications(ctx, userID)
}

// DeleteSessionVerification clears one session's verification.
func (s *DefaultTOTPStore) DeleteSessionVerification(ctx context.Context, sessionID string) error {
	return s.q.deleteSessionVerification(ctx, sessionID)
}

var _ totptypes.Store = (*DefaultTOTPStore)(nil)

func boolToInt[T int8 | int32 | int64](b bool) T {
	if b {
		return 1
	}
	return 0
}
