package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// Booleans are stored as int64 in SQLite.

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/plugins/emailotp/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSqliteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) createUser(ctx context.Context, id, name, createdAt, updatedAt string, avatar, email sql.NullString, disabled, emailVerified bool) error {
	return s.q.CreateUser(ctx, sqlcsqlite.CreateUserParams{
		ID:            id,
		Avatar:        avatar,
		Name:          name,
		Email:         email,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Disabled:      boolToInt[int64](disabled),
		EmailVerified: boolToInt[int64](emailVerified),
	})
}

func (s *sqliteQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (emailUserRow, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return emailUserRow{}, err
	}
	return emailUserRow{
		ID:            u.ID,
		Avatar:        u.Avatar,
		Name:          u.Name,
		Email:         u.Email,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		Disabled:      u.Disabled != 0,
		EmailVerified: u.EmailVerified != 0,
	}, nil
}

func (s *sqliteQuerier) updateUserEmail(ctx context.Context, userID string, email sql.NullString, verified bool, updatedAt string) error {
	return s.q.UpdateUserEmail(ctx, sqlcsqlite.UpdateUserEmailParams{
		ID:            userID,
		Email:         email,
		EmailVerified: boolToInt[int64](verified),
		UpdatedAt:     updatedAt,
	})
}
