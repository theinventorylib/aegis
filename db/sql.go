package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/models"
)

// Dialect represents SQL syntax variations between database systems
type Dialect string

const (
	// PostgreSQL dialect (supports $1, $2 placeholders and RETURNING)
	PostgreSQL Dialect = "postgres"
	// MySQL dialect (supports ? placeholders and LAST_INSERT_ID)
	MySQL Dialect = "mysql"
	// SQLite dialect (supports ? placeholders and LAST_INSERT_ID)
	SQLite Dialect = "sqlite"
)

// SQLProvider implements DBProvider using database/sql for any SQL database
type SQLProvider struct {
	db      *sql.DB
	dialect Dialect
}

// NewSQLProvider creates a new database-agnostic SQL provider
// db: a standard *sql.DB connection from any driver
// dialect: the SQL dialect for query syntax differences
func NewSQLProvider(db *sql.DB, dialect Dialect) *SQLProvider {
	return &SQLProvider{
		db:      db,
		dialect: dialect,
	}
}

// Close closes the database connection
func (p *SQLProvider) Close() error {
	return p.db.Close()
}

// ========== GENERIC QUERY INTERFACE ==========

// sqlRow wraps *sql.Row to implement db.Row
type sqlRow struct {
	row *sql.Row
}

func (r *sqlRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// sqlRows wraps *sql.Rows to implement db.Rows
type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Close() {
	_ = r.rows.Close() // Ignore error in Close
}

func (r *sqlRows) Err() error {
	return r.rows.Err()
}

// sqlResult wraps sql.Result to implement db.Result
type sqlResult struct {
	result sql.Result
}

func (r *sqlResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

func (r *sqlResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

// sqlTx wraps *sql.Tx to implement db.Tx
type sqlTx struct {
	tx *sql.Tx
}

func (t *sqlTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

func (t *sqlTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (t *sqlTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &sqlRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *sqlTx) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *sqlTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Query executes a query that returns rows
func (p *SQLProvider) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (p *SQLProvider) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &sqlRow{row: p.db.QueryRowContext(ctx, query, args...)}
}

// Exec executes a query that doesn't return rows
func (p *SQLProvider) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

// Begin starts a new transaction
func (p *SQLProvider) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

// ========== CORE USER OPERATIONS ==========

// CreateUser creates a new user
func (p *SQLProvider) CreateUser(ctx context.Context) (*models.User, error) {
	user := &models.User{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	switch p.dialect {
	case PostgreSQL:
		// PostgreSQL uses RETURNING clause
		err := p.db.QueryRowContext(ctx, `
			INSERT INTO auth.user (created_at, updated_at)
			VALUES ($1, $2)
			RETURNING id, created_at, updated_at
		`, user.CreatedAt, user.UpdatedAt).Scan(
			&user.ID, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

	case MySQL, SQLite:
		// MySQL and SQLite use LastInsertId
		result, err := p.db.ExecContext(ctx, `
			INSERT INTO auth.user (created_at, updated_at)
			VALUES (?, ?)
		`, user.CreatedAt, user.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
		user.ID = fmt.Sprintf("%d", id)

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (p *SQLProvider) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	query := `SELECT id, created_at, updated_at FROM auth.user WHERE id = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		err = p.db.QueryRowContext(ctx, query+`$1`, id).Scan(
			&user.ID, &user.CreatedAt, &user.UpdatedAt,
		)
	case MySQL, SQLite:
		err = p.db.QueryRowContext(ctx, query+`?`, id).Scan(
			&user.ID, &user.CreatedAt, &user.UpdatedAt,
		)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user
func (p *SQLProvider) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE auth.user SET updated_at = NOW() WHERE id = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`$1`, user.ID)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`?`, user.ID)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user and all associated data (cascades to sessions, etc.)
func (p *SQLProvider) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.user WHERE id = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`$1`, userID)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`?`, userID)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a paginated list of users
func (p *SQLProvider) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	query := `SELECT id, created_at, updated_at FROM auth.user ORDER BY created_at DESC `
	var rows *sql.Rows
	var err error

	switch p.dialect {
	case PostgreSQL:
		rows, err = p.db.QueryContext(ctx, query+`LIMIT $1 OFFSET $2`, limit, offset)
	case MySQL, SQLite:
		rows, err = p.db.QueryContext(ctx, query+`LIMIT ? OFFSET ?`, limit, offset)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() { _ = rows.Close() }() // Ignore close error

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CountUsers returns the total number of users
func (p *SQLProvider) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth.user").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// ========== SESSION OPERATIONS ==========

// CreateSession creates a new session
func (p *SQLProvider) CreateSession(ctx context.Context, session *models.Session) error {
	query := `INSERT INTO auth.session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent) VALUES `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`($1, $2, $3, $4, $5, $6, $7, $8)`,
			session.ID, session.UserID, session.Token, session.RefreshToken,
			session.ExpiresAt, session.CreatedAt, session.IPAddress, session.UserAgent)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`(?, ?, ?, ?, ?, ?, ?, ?)`,
			session.ID, session.UserID, session.Token, session.RefreshToken,
			session.ExpiresAt, session.CreatedAt, session.IPAddress, session.UserAgent)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (p *SQLProvider) GetSession(ctx context.Context, token string) (*models.Session, error) {
	session := &models.Session{}
	query := `SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM auth.session WHERE token = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		err = p.db.QueryRowContext(ctx, query+`$1`, token).Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken,
			&session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
		)
	case MySQL, SQLite:
		err = p.db.QueryRowContext(ctx, query+`?`, token).Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken,
			&session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
		)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (p *SQLProvider) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	session := &models.Session{}
	query := `SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM auth.session WHERE refresh_token = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		err = p.db.QueryRowContext(ctx, query+`$1`, refreshToken).Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken,
			&session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
		)
	case MySQL, SQLite:
		err = p.db.QueryRowContext(ctx, query+`?`, refreshToken).Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken,
			&session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
		)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// UpdateSession updates a session
func (p *SQLProvider) UpdateSession(ctx context.Context, session *models.Session) error {
	query := `UPDATE auth.session SET expires_at = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`$1 WHERE id = $2`, session.ExpiresAt, session.ID)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`? WHERE id = ?`, session.ExpiresAt, session.ID)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteSession deletes a session by token
func (p *SQLProvider) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM auth.session WHERE token = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`$1`, token)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`?`, token)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all sessions for a user
func (p *SQLProvider) GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error) {
	query := `SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent FROM auth.session WHERE user_id = `
	var rows *sql.Rows
	var err error

	switch p.dialect {
	case PostgreSQL:
		rows, err = p.db.QueryContext(ctx, query+`$1 ORDER BY created_at DESC`, userID)
	case MySQL, SQLite:
		rows, err = p.db.QueryContext(ctx, query+`? ORDER BY created_at DESC`, userID)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer func() { _ = rows.Close() }() // Ignore close error

	sessions := []*models.Session{}
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken,
			&session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// DeleteUserSessions deletes all sessions for a user
func (p *SQLProvider) DeleteUserSessions(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.session WHERE user_id = `
	var err error

	switch p.dialect {
	case PostgreSQL:
		_, err = p.db.ExecContext(ctx, query+`$1`, userID)
	case MySQL, SQLite:
		_, err = p.db.ExecContext(ctx, query+`?`, userID)
	default:
		return fmt.Errorf("unsupported dialect: %s", p.dialect)
	}

	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}
