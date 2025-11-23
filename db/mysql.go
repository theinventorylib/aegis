package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/theinventorylib/aegis/models"
)

// MySQLProvider implements DBProvider for MySQL
type MySQLProvider struct {
	db *sql.DB
}

// NewMySQLProvider creates a new MySQL database provider
// connString format: "user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true"
func NewMySQLProvider(connString string) (*MySQLProvider, error) {
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	return &MySQLProvider{db: db}, nil
}

// Close closes the database connection
func (p *MySQLProvider) Close() error {
	return p.db.Close()
}

// ========== GENERIC QUERY INTERFACE ==========

// mysqlRow wraps *sql.Row to implement db.Row
type mysqlRow struct {
	row *sql.Row
}

func (r *mysqlRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// mysqlRows wraps *sql.Rows to implement db.Rows
type mysqlRows struct {
	rows *sql.Rows
}

func (r *mysqlRows) Next() bool {
	return r.rows.Next()
}

func (r *mysqlRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *mysqlRows) Close() {
	r.rows.Close()
}

func (r *mysqlRows) Err() error {
	return r.rows.Err()
}

// mysqlResult wraps sql.Result to implement db.Result
type mysqlResult struct {
	result sql.Result
}

func (r *mysqlResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

func (r *mysqlResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

// mysqlTx wraps *sql.Tx to implement db.Tx
type mysqlTx struct {
	tx *sql.Tx
}

func (t *mysqlTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &mysqlResult{result: result}, nil
}

func (t *mysqlTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &mysqlRows{rows: rows}, nil
}

func (t *mysqlTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &mysqlRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *mysqlTx) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *mysqlTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Query executes a query that returns rows
func (p *MySQLProvider) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &mysqlRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (p *MySQLProvider) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &mysqlRow{row: p.db.QueryRowContext(ctx, query, args...)}
}

// Exec executes a query that doesn't return rows
func (p *MySQLProvider) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &mysqlResult{result: result}, nil
}

// Begin starts a new transaction
func (p *MySQLProvider) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &mysqlTx{tx: tx}, nil
}

// ========== CORE USER OPERATIONS ==========

// CreateUser creates a new user
func (p *MySQLProvider) CreateUser(ctx context.Context) (*models.User, error) {
	user := &models.User{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

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
	return user, nil
}

// GetUserByID retrieves a user by ID
func (p *MySQLProvider) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	err := p.db.QueryRowContext(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		WHERE id = ?
	`, id).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user
func (p *MySQLProvider) UpdateUser(ctx context.Context, user *models.User) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE auth.user
		SET updated_at = NOW()
		WHERE id = ?
	`, user.ID)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user and all associated data (cascades to sessions, etc.)
func (p *MySQLProvider) DeleteUser(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM auth.user WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a paginated list of users
func (p *MySQLProvider) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

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
func (p *MySQLProvider) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth.user").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// ========== SESSION OPERATIONS ==========

// CreateSession creates a new session
func (p *MySQLProvider) CreateSession(ctx context.Context, session *models.Session) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO auth.session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Token, session.RefreshToken, session.ExpiresAt, session.CreatedAt, session.IPAddress, session.UserAgent)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (p *MySQLProvider) GetSession(ctx context.Context, token string) (*models.Session, error) {
	session := &models.Session{}

	err := p.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE token = ?
	`, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (p *MySQLProvider) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	session := &models.Session{}

	err := p.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE refresh_token = ?
	`, refreshToken).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// UpdateSession updates a session
func (p *MySQLProvider) UpdateSession(ctx context.Context, session *models.Session) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE auth.session
		SET expires_at = ?
		WHERE id = ?
	`, session.ExpiresAt, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteSession deletes a session by token
func (p *MySQLProvider) DeleteSession(ctx context.Context, token string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM auth.session WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all sessions for a user
func (p *MySQLProvider) GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

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
func (p *MySQLProvider) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM auth.session WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}
