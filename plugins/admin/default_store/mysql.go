package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// All dialect-specific types (int8 for booleans, int32 for pagination) are
// handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/admin/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMySQLQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, role sql.NullString) error {
	return m.q.CreateUser(ctx, sqlcmysql.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int8](disabled), Role: role,
	})
}

func (m *mysqlQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (adminUserRow, error) {
	u, err := m.q.GetUserByEmail(ctx, email)
	if err != nil {
		return adminUserRow{}, err
	}
	return adminUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, Role: u.Role, Banned: u.Banned != 0,
		BanReason: u.BanReason, BanExpiry: u.BanExpiry, BanCounter: int(u.BanCounter),
	}, nil
}

func (m *mysqlQuerier) getUserByID(ctx context.Context, id string) (adminUserRow, error) {
	u, err := m.q.GetUserByID(ctx, id)
	if err != nil {
		return adminUserRow{}, err
	}
	return adminUserRow{
		ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		Disabled: u.Disabled != 0, Role: u.Role, Banned: u.Banned != 0,
		BanReason: u.BanReason, BanExpiry: u.BanExpiry, BanCounter: int(u.BanCounter),
	}, nil
}

func (m *mysqlQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return m.q.UpdateUser(ctx, sqlcmysql.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		UpdatedAt: updatedAt, Disabled: boolToInt[int8](disabled),
	})
}

func (m *mysqlQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return m.q.DeleteUser(ctx, sqlcmysql.DeleteUserParams{ID: id, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) listUsers(ctx context.Context, offset, limit int32) ([]adminUserRow, error) {
	rows, err := m.q.ListUsers(ctx, sqlcmysql.ListUsersParams{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]adminUserRow, len(rows))
	for i, u := range rows {
		out[i] = adminUserRow{
			ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email,
			CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
			Disabled: u.Disabled != 0, Role: u.Role, Banned: u.Banned != 0,
			BanReason: u.BanReason, BanExpiry: u.BanExpiry, BanCounter: int(u.BanCounter),
		}
	}
	return out, nil
}

func (m *mysqlQuerier) listUsersRaw(ctx context.Context, offset, limit int32) ([]adminRawRow, error) {
	rows, err := m.q.ListUsersRaw(ctx, sqlcmysql.ListUsersRawParams{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]adminRawRow, len(rows))
	for i, u := range rows {
		out[i] = adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (m *mysqlQuerier) getUserRaw(ctx context.Context, id string) (adminRawRow, error) {
	u, err := m.q.GetUserRaw(ctx, id)
	if err != nil {
		return adminRawRow{}, err
	}
	return adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}, nil
}

func (m *mysqlQuerier) countUsers(ctx context.Context) (int64, error) { return m.q.CountUsers(ctx) }

func (m *mysqlQuerier) updateUserRole(ctx context.Context, id string, role sql.NullString, updatedAt string) error {
	return m.q.UpdateUserRole(ctx, sqlcmysql.UpdateUserRoleParams{ID: id, Role: role, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) getRole(ctx context.Context, id string) (string, error) {
	return m.q.GetRole(ctx, id)
}

func (m *mysqlQuerier) banUser(ctx context.Context, id string, banReason, banExpiry sql.NullString, updatedAt string) error {
	return m.q.BanUser(ctx, sqlcmysql.BanUserParams{ID: id, BanReason: banReason, BanExpiry: banExpiry, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) unbanUser(ctx context.Context, id, updatedAt string) error {
	return m.q.UnbanUser(ctx, sqlcmysql.UnbanUserParams{ID: id, UpdatedAt: updatedAt})
}
