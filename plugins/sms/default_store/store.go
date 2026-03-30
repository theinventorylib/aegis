// Package defaultstore implements the SQL-backed default store for the sms plugin.
package defaultstore

// store.go — DefaultSMSStore: the switch happens once (in the constructor);
// all methods are dialect-agnostic and delegate to the querier interface.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/plugins"
	smstypes "github.com/theinventorylib/aegis/plugins/sms/types"
)

// DefaultSMSStore implements smstypes.Store using a SQL database.
//
// Supports PostgreSQL, MySQL, and SQLite through dialect-specific sqlc-generated
// queries. Safe for concurrent use.
type DefaultSMSStore struct{ q querier }

// NewDefaultSMSStore creates a DefaultSMSStore for the given dialect.
// The dialect switch happens exactly once here; all store methods call
// through the querier interface and are dialect-agnostic.
func NewDefaultSMSStore(db *sql.DB, dialect plugins.Dialect) (*DefaultSMSStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMySQLQuerier(db)
	case plugins.DialectSQLite:
		q = newSQLiteQuerier(db)
	default:
		return nil, fmt.Errorf("sms: unsupported dialect %q", dialect)
	}
	return &DefaultSMSStore{q: q}, nil
}

// CreateUser creates a new SMS-enabled user.
func (s *DefaultSMSStore) CreateUser(ctx context.Context, user smstypes.User) (*smstypes.User, error) {
	err := s.q.createUser(ctx,
		user.ID,
		toNullString(user.Avatar),
		user.Name,
		toNullString(user.Email),
		user.CreatedAt.Format(time.RFC3339),
		user.UpdatedAt.Format(time.RFC3339),
		user.Disabled,
		toNullString(derefStr(user.Phone)),
		user.PhoneVerified,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by their unique ID.
func (s *DefaultSMSStore) GetUserByID(ctx context.Context, id string) (*smstypes.User, error) {
	row, err := s.q.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return buildUser(row), nil
}

// GetUserByPhone retrieves a user by their phone number.
func (s *DefaultSMSStore) GetUserByPhone(ctx context.Context, phone string) (*smstypes.User, error) {
	row, err := s.q.getUserByPhone(ctx, toNullString(phone))
	if err != nil {
		return nil, err
	}
	return buildUser(row), nil
}

// UpdateUserPhone updates a user's phone number and verification status.
func (s *DefaultSMSStore) UpdateUserPhone(ctx context.Context, userID, phone string, verified bool) error {
	return s.q.updateUserPhone(ctx, userID, toNullString(phone), verified, time.Now().Format(time.RFC3339))
}

// buildUser constructs a *smstypes.User from a canonical smsUserRow.
func buildUser(row smsUserRow) *smstypes.User {
	var p *string
	if row.PhoneNumber.Valid {
		p = &row.PhoneNumber.String
	}
	return &smstypes.User{
		User: auth.User{
			ID:        row.ID,
			Avatar:    nullStringToString(row.Avatar),
			Name:      row.Name,
			Email:     nullStringToString(row.Email),
			CreatedAt: parseTime(row.CreatedAt),
			UpdatedAt: parseTime(row.UpdatedAt),
			Disabled:  row.Disabled,
		},
		Phone:         p,
		PhoneVerified: row.PhoneVerified,
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func boolToInt[T int8 | int32 | int64](b bool) T {
	if b {
		return 1
	}
	return 0
}
