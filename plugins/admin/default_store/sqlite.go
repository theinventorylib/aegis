package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// All dialect-specific types (int64 for booleans, int64 for pagination) are
// handled here and nowhere else. Pagination params are widened from int32 to int64.

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/plugins/admin/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSQLiteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, role sql.NullString) error {
	return s.q.CreateUser(ctx, sqlcsqlite.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int64](disabled), Role: role,
	})
}

func (s *sqliteQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (adminUserRow, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
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

func (s *sqliteQuerier) getUserByID(ctx context.Context, id string) (adminUserRow, error) {
	u, err := s.q.GetUserByID(ctx, id)
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

func (s *sqliteQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return s.q.UpdateUser(ctx, sqlcsqlite.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		UpdatedAt: updatedAt, Disabled: boolToInt[int64](disabled),
	})
}

func (s *sqliteQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return s.q.DeleteUser(ctx, sqlcsqlite.DeleteUserParams{ID: id, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) listUsers(ctx context.Context, offset, limit int32) ([]adminUserRow, error) {
	rows, err := s.q.ListUsers(ctx, sqlcsqlite.ListUsersParams{Offset: int64(offset), Limit: int64(limit)})
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

func (s *sqliteQuerier) listUsersRaw(ctx context.Context, offset, limit int32) ([]adminRawRow, error) {
	rows, err := s.q.ListUsersRaw(ctx, sqlcsqlite.ListUsersRawParams{Offset: int64(offset), Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]adminRawRow, len(rows))
	for i, u := range rows {
		out[i] = adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (s *sqliteQuerier) getUserRaw(ctx context.Context, id string) (adminRawRow, error) {
	u, err := s.q.GetUserRaw(ctx, id)
	if err != nil {
		return adminRawRow{}, err
	}
	return adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}, nil
}

func (s *sqliteQuerier) countUsers(ctx context.Context) (int64, error) { return s.q.CountUsers(ctx) }

func (s *sqliteQuerier) updateUserRole(ctx context.Context, id string, role sql.NullString, updatedAt string) error {
	return s.q.UpdateUserRole(ctx, sqlcsqlite.UpdateUserRoleParams{ID: id, Role: role, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) getRole(ctx context.Context, id string) (string, error) {
	return s.q.GetRole(ctx, id)
}

func (s *sqliteQuerier) banUser(ctx context.Context, id string, banReason, banExpiry sql.NullString, updatedAt string) error {
	return s.q.BanUser(ctx, sqlcsqlite.BanUserParams{ID: id, BanReason: banReason, BanExpiry: banExpiry, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) unbanUser(ctx context.Context, id, updatedAt string) error {
	return s.q.UnbanUser(ctx, sqlcsqlite.UnbanUserParams{ID: id, UpdatedAt: updatedAt})
}
