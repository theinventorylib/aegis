// Package db provides database provider interfaces and implementations for Aegis.
package db

import (
	"context"

	"github.com/theinventorylib/aegis/models"
)

// Row represents a single row from a query result.
type Row interface {
	Scan(dest ...interface{}) error
}

// Rows represents multiple rows from a query result.
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close()
	Err() error
}

// Result represents the result of an Exec operation.
type Result interface {
	RowsAffected() (int64, error)
	LastInsertId() (int64, error)
}

// Tx represents a database transaction.
type Tx interface {
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Provider is the interface for database operations.
type Provider interface {
	// User operations

	// CreateUser creates a new user record.
	CreateUser(ctx context.Context) (*models.User, error)
	// GetUserByID retrieves a user by their unique ID.
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	// UpdateUser updates an existing user's information.
	UpdateUser(ctx context.Context, user *models.User) error
	// ListUsers retrieves a paginated list of users.
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)
	// DeleteUser deletes a user and all associated data.
	DeleteUser(ctx context.Context, id string) error
	// CountUsers returns the total number of users.
	CountUsers(ctx context.Context) (int, error)

	// Session operations

	// CreateSession creates a new session record.
	CreateSession(ctx context.Context, session *models.Session) error
	// GetSession retrieves a session by its access token.
	GetSession(ctx context.Context, token string) (*models.Session, error)
	// GetSessionByRefreshToken retrieves a session by its refresh token.
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error)
	// UpdateSession updates an existing session (e.g., extending expiration).
	UpdateSession(ctx context.Context, session *models.Session) error
	// DeleteSession deletes a session by its access token.
	DeleteSession(ctx context.Context, token string) error
	// GetUserSessions retrieves all active sessions for a specific user.
	GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error)
	// DeleteUserSessions deletes all sessions for a specific user.
	DeleteUserSessions(ctx context.Context, userID string) error

	// Generic query methods for plugins

	// Query executes a query that returns multiple rows.
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	// QueryRow executes a query that is expected to return at most one row.
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	// Exec executes a query without returning any rows (e.g., INSERT, UPDATE, DELETE).
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	// Begin starts a new transaction.
	Begin(ctx context.Context) (Tx, error)
}
