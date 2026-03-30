// Package defaultstore implements the SQL-backed default store for the admin plugin.
package defaultstore

// store.go — DefaultAdminStore: the switch happens once (in the constructor);
// all methods are dialect-agnostic and delegate to the querier interface.

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	admintypes "github.com/theinventorylib/aegis/plugins/admin/types"
)

// DefaultAdminStore implements admintypes.Store using a SQL database.
//
// Supports PostgreSQL, MySQL, and SQLite through dialect-specific sqlc-generated
// queries. Safe for concurrent use.
type DefaultAdminStore struct{ q querier }

// NewDefaultAdminStore creates a DefaultAdminStore for the given dialect.
// The dialect switch happens exactly once here; all store methods call
// through the querier interface and are dialect-agnostic.
func NewDefaultAdminStore(db *sql.DB, dialect plugins.Dialect) (*DefaultAdminStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMySQLQuerier(db)
	case plugins.DialectSQLite:
		q = newSQLiteQuerier(db)
	default:
		return nil, fmt.Errorf("admin: unsupported dialect %q", dialect)
	}
	return &DefaultAdminStore{q: q}, nil
}

// Create creates a new user.
func (s *DefaultAdminStore) Create(ctx context.Context, user admintypes.User) (admintypes.User, error) {
	err := s.q.createUser(ctx,
		user.ID,
		toNullString(user.Avatar),
		user.Name,
		toNullString(user.Email),
		user.CreatedAt.Format(time.RFC3339),
		user.UpdatedAt.Format(time.RFC3339),
		user.Disabled,
		toNullString(user.Role),
	)
	if err != nil {
		return admintypes.User{}, err
	}
	return user, nil
}

// GetByEmail retrieves a user by email.
func (s *DefaultAdminStore) GetByEmail(ctx context.Context, email string) (admintypes.User, error) {
	row, err := s.q.getUserByEmail(ctx, toNullString(email))
	if err != nil {
		return admintypes.User{}, err
	}
	return buildUser(row), nil
}

// GetByID retrieves a user by ID.
func (s *DefaultAdminStore) GetByID(ctx context.Context, id string) (admintypes.User, error) {
	row, err := s.q.getUserByID(ctx, id)
	if err != nil {
		return admintypes.User{}, err
	}
	return buildUser(row), nil
}

// Update updates user information.
func (s *DefaultAdminStore) Update(ctx context.Context, user admintypes.User) error {
	return s.q.updateUser(ctx,
		user.ID,
		toNullString(user.Avatar),
		user.Name,
		toNullString(user.Email),
		user.UpdatedAt.Format(time.RFC3339),
		user.Disabled,
	)
}

// Delete soft-deletes a user.
func (s *DefaultAdminStore) Delete(ctx context.Context, id string) error {
	return s.q.deleteUser(ctx, id, time.Now().Format(time.RFC3339))
}

// List retrieves a paginated list of users.
func (s *DefaultAdminStore) List(ctx context.Context, offset, limit int) ([]admintypes.User, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	rows, err := s.q.listUsers(ctx, core.ClampIntToInt32(offset), core.ClampIntToInt32(limit))
	if err != nil {
		return nil, err
	}
	users := make([]admintypes.User, len(rows))
	for i, row := range rows {
		users[i] = buildUser(row)
	}
	return users, nil
}

// ListUsersRaw retrieves a paginated list of users as raw map data.
func (s *DefaultAdminStore) ListUsersRaw(ctx context.Context, offset, limit int) ([]map[string]any, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	rows, err := s.q.listUsersRaw(ctx, core.ClampIntToInt32(offset), core.ClampIntToInt32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, len(rows))
	for i, u := range rows {
		result[i] = rawToMap(u)
	}
	return result, nil
}

// GetUserRaw retrieves a single user as raw map data.
func (s *DefaultAdminStore) GetUserRaw(ctx context.Context, userID string) (map[string]any, error) {
	row, err := s.q.getUserRaw(ctx, userID)
	if err != nil {
		return nil, err
	}
	return rawToMap(row), nil
}

// Count returns the total number of users.
func (s *DefaultAdminStore) Count(ctx context.Context) (int, error) {
	count, err := s.q.countUsers(ctx)
	return int(count), err
}

// AssignRole assigns a role to a user.
func (s *DefaultAdminStore) AssignRole(ctx context.Context, userID, role string) error {
	return s.q.updateUserRole(ctx, userID, toNullString(role), time.Now().Format(time.RFC3339))
}

// GetRole retrieves a user's role.
func (s *DefaultAdminStore) GetRole(ctx context.Context, userID string) (string, error) {
	return s.q.getRole(ctx, userID)
}

// RemoveRole removes a role from a user by resetting it to "user".
func (s *DefaultAdminStore) RemoveRole(ctx context.Context, userID string, _ string) error {
	return s.AssignRole(ctx, userID, "user")
}

// BanUser bans a user.
func (s *DefaultAdminStore) BanUser(ctx context.Context, userID, reason string, expiry *time.Time) error {
	var expiryStr sql.NullString
	if expiry != nil {
		expiryStr = sql.NullString{String: expiry.Format(time.RFC3339), Valid: true}
	}
	return s.q.banUser(ctx, userID, toNullString(reason), expiryStr, time.Now().Format(time.RFC3339))
}

// UnbanUser unbans a user.
func (s *DefaultAdminStore) UnbanUser(ctx context.Context, userID string) error {
	return s.q.unbanUser(ctx, userID, time.Now().Format(time.RFC3339))
}

// GetStats retrieves system statistics.
func (s *DefaultAdminStore) GetStats(ctx context.Context) (admintypes.StatsResponse, error) {
	count, err := s.Count(ctx)
	if err != nil {
		return admintypes.StatsResponse{}, err
	}
	return admintypes.StatsResponse{TotalUsers: count}, nil
}

// buildUser constructs an admintypes.User from a canonical adminUserRow.
func buildUser(row adminUserRow) admintypes.User {
	var be *time.Time
	if row.BanExpiry.Valid {
		if t, err := time.Parse(time.RFC3339, row.BanExpiry.String); err == nil {
			be = &t
		}
	}
	u := admintypes.User{}
	u.SetID(row.ID)
	u.Avatar = fromNullString(row.Avatar)
	u.SetName(row.Name)
	u.SetEmail(fromNullString(row.Email))
	u.SetCreatedAt(parseTime(row.CreatedAt))
	u.SetUpdatedAt(parseTime(row.UpdatedAt))
	u.Disabled = row.Disabled
	u.Role = fromNullString(row.Role)
	u.Banned = row.Banned
	u.BanReason = fromNullString(row.BanReason)
	u.BanExpiry = be
	u.BanCounter = row.BanCounter
	return u
}

// rawToMap converts an adminRawRow to the map format expected by *Raw methods.
func rawToMap(row adminRawRow) map[string]any {
	return map[string]any{
		"id":        row.ID,
		"createdAt": row.CreatedAt,
		"updatedAt": row.UpdatedAt,
		"email":     row.Email,
		"role":      row.Role,
		"disabled":  row.Disabled,
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
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
