package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// Booleans are stored as int8 in MySQL.

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/emailotp/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMysqlQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) createUser(ctx context.Context, id, name, createdAt, updatedAt string, avatar, email sql.NullString, disabled, emailVerified bool) error {
	return m.q.CreateUser(ctx, sqlcmysql.CreateUserParams{
		ID:            id,
		Avatar:        avatar,
		Name:          name,
		Email:         email,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Disabled:      boolToInt[int8](disabled),
		EmailVerified: boolToInt[int8](emailVerified),
	})
}

func (m *mysqlQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (emailUserRow, error) {
	u, err := m.q.GetUserByEmail(ctx, email)
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

func (m *mysqlQuerier) updateUserEmail(ctx context.Context, userID string, email sql.NullString, verified bool, updatedAt string) error {
	return m.q.UpdateUserEmail(ctx, sqlcmysql.UpdateUserEmailParams{
		ID:            userID,
		Email:         email,
		EmailVerified: boolToInt[int8](verified),
		UpdatedAt:     updatedAt,
	})
}
