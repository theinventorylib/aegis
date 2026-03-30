package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// Dialect-specific types (int8 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/sms/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMySQLQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, phoneNumber sql.NullString, phoneVerified bool) error {
	return m.q.CreateUser(ctx, sqlcmysql.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int8](disabled), PhoneNumber: phoneNumber,
		PhoneVerified: boolToInt[int8](phoneVerified),
	})
}

func (m *mysqlQuerier) getUserByID(ctx context.Context, id string) (smsUserRow, error) {
	u, err := m.q.GetUserByID(ctx, id)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (m *mysqlQuerier) getUserByPhone(ctx context.Context, phoneNumber sql.NullString) (smsUserRow, error) {
	u, err := m.q.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		return smsUserRow{}, err
	}
	return smsUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, PhoneNumber: u.PhoneNumber, PhoneVerified: u.PhoneVerified != 0,
	}, nil
}

func (m *mysqlQuerier) updateUserPhone(ctx context.Context, id string, phoneNumber sql.NullString, phoneVerified bool, updatedAt string) error {
	return m.q.UpdateUserPhone(ctx, sqlcmysql.UpdateUserPhoneParams{
		ID: id, PhoneNumber: phoneNumber, PhoneVerified: boolToInt[int8](phoneVerified), UpdatedAt: updatedAt,
	})
}
