package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Booleans are stored as int32 in PostgreSQL.

import (
	"context"
	"database/sql"
	"time"

	sqlcpostgres "github.com/theinventorylib/aegis/v2/plugins/emailotp/internal/gen/postgres"
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
		CreatedAt:     pgParseTime(createdAt),
		UpdatedAt:     pgParseTime(updatedAt),
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
		CreatedAt:     pgFormatTime(u.CreatedAt),
		UpdatedAt:     pgFormatTime(u.UpdatedAt),
		Disabled:      u.Disabled != 0,
		EmailVerified: u.EmailVerified != 0,
	}, nil
}

func (p *postgresQuerier) updateUserEmail(ctx context.Context, userID string, email sql.NullString, verified bool, updatedAt string) error {
	return p.q.UpdateUserEmail(ctx, sqlcpostgres.UpdateUserEmailParams{
		ID:            userID,
		Email:         email,
		EmailVerified: boolToInt[int32](verified),
		UpdatedAt:     pgParseTime(updatedAt),
	})
}

// pgParseTime converts a canonical RFC3339 string to the time.Time the
// generated postgres queries expect.
func pgParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// pgFormatTime converts a time.Time back to the canonical RFC3339 string.
func pgFormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }
