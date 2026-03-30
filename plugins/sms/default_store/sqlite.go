package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// Dialect-specific types (int64 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/plugins/sms/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSQLiteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, phoneNumber sql.NullString, phoneVerified bool) error {
	return s.q.CreateUser(ctx, sqlcsqlite.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int64](disabled), PhoneNumber: phoneNumber,
		PhoneVerified: boolToInt[int64](phoneVerified),
	})
}

func (s *sqliteQuerier) getUserByID(ctx context.Context, id string) (smsUserRow, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (s *sqliteQuerier) getUserByPhone(ctx context.Context, phoneNumber sql.NullString) (smsUserRow, error) {
	u, err := s.q.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (s *sqliteQuerier) updateUserPhone(ctx context.Context, id string, phoneNumber sql.NullString, phoneVerified bool, updatedAt string) error {
	return s.q.UpdateUserPhone(ctx, sqlcsqlite.UpdateUserPhoneParams{
		ID: id, PhoneNumber: phoneNumber, PhoneVerified: boolToInt[int64](phoneVerified), UpdatedAt: updatedAt,
	})
}
