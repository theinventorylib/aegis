package defaultstore

// querier.go — the single internal DB interface for the sms plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// Canonical row types — dialect-neutral, owned by this package.

// smsUserRow is the full user row returned by SMS user queries.
type smsUserRow struct {
	ID, Name, CreatedAt, UpdatedAt string
	Avatar, Email, PhoneNumber     sql.NullString
	Disabled, PhoneVerified        bool
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultSMSStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool, phoneNumber sql.NullString, phoneVerified bool) error
	getUserByID(ctx context.Context, id string) (smsUserRow, error)
	getUserByPhone(ctx context.Context, phoneNumber sql.NullString) (smsUserRow, error)
	updateUserPhone(ctx context.Context, id string, phoneNumber sql.NullString, phoneVerified bool, updatedAt string) error
}
