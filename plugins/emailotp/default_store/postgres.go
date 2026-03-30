package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Booleans are stored as int32 in PostgreSQL.

import (
	"context"
	"database/sql"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/emailotp/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) createUser(ctx context.Context, id, name, createdAt, updatedAt string, avatar, email sql.NullString, disabled, emailVerified bool) error {
	return p.q.CreateUser(ctx, sqlcpostgres.CreateUserParams{
		ID:            id,
		Avatar:        avatar,
		Name:          name,
		Email:         email,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Disabled:      boolToInt[int32](disabled),
		EmailVerified: boolToInt[int32](emailVerified),
	})
}

func (p *postgresQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (emailUserRow, error) {
	u, err := p.q.GetUserByEmail(ctx, email)
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

func (p *postgresQuerier) updateUserEmail(ctx context.Context, userID string, email sql.NullString, verified bool, updatedAt string) error {
	return p.q.UpdateUserEmail(ctx, sqlcpostgres.UpdateUserEmailParams{
		ID:            userID,
		Email:         email,
		EmailVerified: boolToInt[int32](verified),
		UpdatedAt:     updatedAt,
	})
}
