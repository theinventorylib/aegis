package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Dialect-specific types (int32 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/sms/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, phoneNumber sql.NullString, phoneVerified bool) error {
	return p.q.CreateUser(ctx, sqlcpostgres.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int32](disabled), PhoneNumber: phoneNumber,
		PhoneVerified: boolToInt[int32](phoneVerified),
	})
}

func (p *postgresQuerier) getUserByID(ctx context.Context, id string) (smsUserRow, error) {
	u, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (p *postgresQuerier) getUserByPhone(ctx context.Context, phoneNumber sql.NullString) (smsUserRow, error) {
	u, err := p.q.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (p *postgresQuerier) updateUserPhone(ctx context.Context, id string, phoneNumber sql.NullString, phoneVerified bool, updatedAt string) error {
	return p.q.UpdateUserPhone(ctx, sqlcpostgres.UpdateUserPhoneParams{
		ID: id, PhoneNumber: phoneNumber, PhoneVerified: boolToInt[int32](phoneVerified), UpdatedAt: updatedAt,
	})
}
