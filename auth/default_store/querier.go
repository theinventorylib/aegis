package defaultstore

// querier.go — the single internal DB interface for the auth package.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// Canonical row types — dialect-neutral, owned by this package.

type userRow struct {
	ID, Name, CreatedAt, UpdatedAt string
	Avatar, Email                  sql.NullString
	Disabled                       bool
}

type accountRow struct {
	ID, UserID, Provider                                                  string
	ProviderAccountID, PasswordHash, AccessToken, RefreshToken, ExpiresAt sql.NullString
	CreatedAt, UpdatedAt                                                  string
}

type verificationRow struct {
	ID, Identifier, Token, Type, ExpiresAt, CreatedAt string
}

type sessionRow struct {
	ID, UserID, Token, ExpiresAt, CreatedAt string
	RefreshToken, IPAddress, UserAgent      sql.NullString
}

// Querier is the one interface all store methods use. It covers all four
// sub-stores (user, account, verification, session) so the dialect is chosen
// exactly once in NewDefaultStore and nothing else needs to know about it.
type querier interface {
	// User queries
	createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool) error
	getUserByEmail(ctx context.Context, email sql.NullString) (userRow, error)
	getUserByID(ctx context.Context, id string) (userRow, error)
	updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error
	deleteUser(ctx context.Context, id, updatedAt string) error
	listUsers(ctx context.Context, offset, limit int32) ([]userRow, error)
	countUsers(ctx context.Context) (int64, error)

	// Account queries
	createAccount(ctx context.Context, id, userID, provider string, providerAccountID, passwordHash, accessToken, refreshToken, expiresAt sql.NullString, createdAt, updatedAt string) error
	getAccountByID(ctx context.Context, id string) (accountRow, error)
	getAccountsByUserID(ctx context.Context, userID string) ([]accountRow, error)
	getAccountByProvider(ctx context.Context, provider string, providerAccountID sql.NullString) (accountRow, error)
	updateAccount(ctx context.Context, id string, accessToken, refreshToken, expiresAt sql.NullString, updatedAt string) error
	deleteAccount(ctx context.Context, id string) error

	// Verification queries
	createVerification(ctx context.Context, id, identifier, token, vType, expiresAt, createdAt string) error
	getVerificationByToken(ctx context.Context, token, expiresAt string) (verificationRow, error)
	getVerificationsByIdentifier(ctx context.Context, identifier, expiresAt string) ([]verificationRow, error)
	invalidateVerificationByIdentifier(ctx context.Context, identifier, vType, expiresAt string) error
	deleteVerification(ctx context.Context, id string) error
	cleanupExpiredVerifications(ctx context.Context, now string) error

	// Session queries
	createSession(ctx context.Context, id, userID, token string, refreshToken sql.NullString, expiresAt, createdAt string, ipAddress, userAgent sql.NullString) error
	getSession(ctx context.Context, id, expiresAt string) (sessionRow, error)
	getSessionByToken(ctx context.Context, token, expiresAt string) (sessionRow, error)
	getSessionByRefreshToken(ctx context.Context, refreshToken sql.NullString, expiresAt string) (sessionRow, error)
	getSessionsByUserID(ctx context.Context, userID, expiresAt string, offset, limit int32) ([]sessionRow, error)
	countSessionsByUserID(ctx context.Context, userID, expiresAt string) (int64, error)
	updateSession(ctx context.Context, id string, refreshToken sql.NullString, expiresAt string) error
	deleteSession(ctx context.Context, id string) error
	deleteSessionsByUserID(ctx context.Context, userID string) error
	cleanupExpiredSessions(ctx context.Context, now string) error
}
