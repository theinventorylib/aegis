package defaultstore

// querier.go — the single internal DB interface for the admin plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// Canonical row types — dialect-neutral, owned by this package.

// adminUserRow is the full user row returned by lookup queries.
type adminUserRow struct {
	ID, Name, CreatedAt, UpdatedAt string
	Avatar, Email, Role            sql.NullString
	BanReason, BanExpiry           sql.NullString
	Disabled, Banned               bool
	BanCounter                     int
}

// adminRawRow is the reduced row returned by the *Raw queries.
type adminRawRow struct {
	ID, CreatedAt, UpdatedAt, Email, Role string
	Disabled                              bool
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultAdminStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	// User queries
	createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, role sql.NullString) error
	getUserByEmail(ctx context.Context, email sql.NullString) (adminUserRow, error)
	getUserByID(ctx context.Context, id string) (adminUserRow, error)
	updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error
	deleteUser(ctx context.Context, id, updatedAt string) error
	listUsers(ctx context.Context, offset, limit int32) ([]adminUserRow, error)
	listUsersRaw(ctx context.Context, offset, limit int32) ([]adminRawRow, error)
	getUserRaw(ctx context.Context, id string) (adminRawRow, error)
	countUsers(ctx context.Context) (int64, error)

	// Role queries
	updateUserRole(ctx context.Context, id string, role sql.NullString, updatedAt string) error
	getRole(ctx context.Context, id string) (string, error)

	// Ban queries
	banUser(ctx context.Context, id string, banReason, banExpiry sql.NullString, updatedAt string) error
	unbanUser(ctx context.Context, id, updatedAt string) error
}
