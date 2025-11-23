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

// Tx represents a database transaction
type Tx interface {
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// DBProvider defines the interface for database operations
// Core operations only - plugins use generic query methods
type DBProvider interface {
	// User operations
	CreateUser(ctx context.Context) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)
	CountUsers(ctx context.Context) (int, error)

	// Session operations
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, token string) (*models.Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error)
	UpdateSession(ctx context.Context, session *models.Session) error
	DeleteSession(ctx context.Context, token string) error
	GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error)
	DeleteUserSessions(ctx context.Context, userID string) error

	// Generic query methods for plugins
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Begin(ctx context.Context) (Tx, error)
}
