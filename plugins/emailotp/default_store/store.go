// Package defaultstore implements the SQL-backed default store for the emailotp plugin.
package defaultstore

// store.go — DefaultEmailOTPStore, the concrete implementation of emailotptypes.Store.
//
// Dialect is selected once in NewDefaultEmailOTPStore; all methods delegate to the
// unexported querier interface so they are dialect-agnostic.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/plugins"
	emailotptypes "github.com/theinventorylib/aegis/plugins/emailotp/types"
)

// DefaultEmailOTPStore implements emailotptypes.Store using a SQL database.
type DefaultEmailOTPStore struct {
	q querier
}

// NewDefaultEmailOTPStore creates a DefaultEmailOTPStore for the given dialect.
func NewDefaultEmailOTPStore(db *sql.DB, dialect plugins.Dialect) (*DefaultEmailOTPStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMysqlQuerier(db)
	case plugins.DialectSQLite:
		q = newSqliteQuerier(db)
	default:
		return nil, fmt.Errorf("emailotp: unsupported dialect %q", dialect)
	}
	return &DefaultEmailOTPStore{q: q}, nil
}

// CreateUser creates a new user in the store and returns the created user.
func (s *DefaultEmailOTPStore) CreateUser(ctx context.Context, user emailotptypes.User) (*emailotptypes.User, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	avatar := strToNullString(user.Avatar)
	email := ptrToNullString(&user.Email)

	err := s.q.createUser(ctx,
		user.ID, user.Name, nowStr, nowStr,
		avatar, email,
		user.Disabled, user.EmailVerified,
	)
	if err != nil {
		return nil, err
	}
	user.CreatedAt = now
	user.UpdatedAt = now
	return &user, nil
}

// GetUserByEmail retrieves a user by their email address.
func (s *DefaultEmailOTPStore) GetUserByEmail(ctx context.Context, email string) (*emailotptypes.User, error) {
	row, err := s.q.getUserByEmail(ctx, sql.NullString{String: email, Valid: email != ""})
	if err != nil {
		return nil, err
	}
	return buildUser(row), nil
}

// UpdateUserEmail updates the email address and verification status for a user.
func (s *DefaultEmailOTPStore) UpdateUserEmail(ctx context.Context, userID, email string, verified bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return s.q.updateUserEmail(ctx, userID, sql.NullString{String: email, Valid: email != ""}, verified, now)
}

// buildUser converts the internal emailUserRow to the public emailotptypes.User.
func buildUser(r emailUserRow) *emailotptypes.User {
	var ep *string
	if r.Email.Valid {
		ep = &r.Email.String
	}
	avatar := ""
	if r.Avatar.Valid {
		avatar = r.Avatar.String
	}
	return &emailotptypes.User{
		User: auth.User{
			ID:        r.ID,
			Name:      r.Name,
			Avatar:    avatar,
			Disabled:  r.Disabled,
			CreatedAt: parseTime(r.CreatedAt),
			UpdatedAt: parseTime(r.UpdatedAt),
			Email:     *ep,
		},
		EmailVerified: r.EmailVerified,
	}
}

// strToNullString converts a string to sql.NullString (valid when non-empty).
func strToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ptrToNullString converts a *string to sql.NullString.
func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// parseTime parses an RFC3339 string to time.Time (returns zero on failure).
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s) //nolint:errcheck
	return t
}
