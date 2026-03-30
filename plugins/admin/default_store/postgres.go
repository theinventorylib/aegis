package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// All dialect-specific types (int32 for booleans, int32 for pagination) are
// handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/admin/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, role sql.NullString) error {
	return p.q.CreateUser(ctx, sqlcpostgres.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Disabled: boolToInt[int32](disabled), Role: role,
	})
}

func (p *postgresQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (adminUserRow, error) {
	u, err := p.q.GetUserByEmail(ctx, email)
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

func (p *postgresQuerier) getUserByID(ctx context.Context, id string) (adminUserRow, error) {
	u, err := p.q.GetUserByID(ctx, id)
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

func (p *postgresQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return p.q.UpdateUser(ctx, sqlcpostgres.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		UpdatedAt: updatedAt, Disabled: boolToInt[int32](disabled),
	})
}

func (p *postgresQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return p.q.DeleteUser(ctx, sqlcpostgres.DeleteUserParams{ID: id, UpdatedAt: updatedAt})
}

func (p *postgresQuerier) listUsers(ctx context.Context, offset, limit int32) ([]adminUserRow, error) {
	rows, err := p.q.ListUsers(ctx, sqlcpostgres.ListUsersParams{Offset: offset, Limit: limit})
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

func (p *postgresQuerier) listUsersRaw(ctx context.Context, offset, limit int32) ([]adminRawRow, error) {
	rows, err := p.q.ListUsersRaw(ctx, sqlcpostgres.ListUsersRawParams{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]adminRawRow, len(rows))
	for i, u := range rows {
		out[i] = adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (p *postgresQuerier) getUserRaw(ctx context.Context, id string) (adminRawRow, error) {
	u, err := p.q.GetUserRaw(ctx, id)
	if err != nil {
		return adminRawRow{}, err
	}
	return adminRawRow{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, Role: u.Role, Disabled: u.Disabled != 0}, nil
}

func (p *postgresQuerier) countUsers(ctx context.Context) (int64, error) { return p.q.CountUsers(ctx) }

func (p *postgresQuerier) updateUserRole(ctx context.Context, id string, role sql.NullString, updatedAt string) error {
	return p.q.UpdateUserRole(ctx, sqlcpostgres.UpdateUserRoleParams{ID: id, Role: role, UpdatedAt: updatedAt})
}

func (p *postgresQuerier) getRole(ctx context.Context, id string) (string, error) {
	return p.q.GetRole(ctx, id)
}

func (p *postgresQuerier) banUser(ctx context.Context, id string, banReason, banExpiry sql.NullString, updatedAt string) error {
	return p.q.BanUser(ctx, sqlcpostgres.BanUserParams{ID: id, BanReason: banReason, BanExpiry: banExpiry, UpdatedAt: updatedAt})
}

func (p *postgresQuerier) unbanUser(ctx context.Context, id, updatedAt string) error {
	return p.q.UnbanUser(ctx, sqlcpostgres.UnbanUserParams{ID: id, UpdatedAt: updatedAt})
}
