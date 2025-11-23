package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/theinventorylib/aegis/models"
)

// PostgresProvider implements DBProvider for PostgreSQL
type PostgresProvider struct {
	pool *pgxpool.Pool
}

// NewPostgresProvider creates a new PostgreSQL database provider
func NewPostgresProvider(connString string) (*PostgresProvider, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresProvider{pool: pool}, nil
}

// Close closes the database connection pool
func (p *PostgresProvider) Close() {
	p.pool.Close()
}

// ========== GENERIC QUERY INTERFACE ==========

// pgxRow wraps pgx.Row to implement db.Row
type pgxRow struct {
	row pgx.Row
}

func (r *pgxRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// pgxRows wraps pgx.Rows to implement db.Rows
type pgxRows struct {
	rows pgx.Rows
}

func (r *pgxRows) Next() bool {
	return r.rows.Next()
}

func (r *pgxRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *pgxRows) Close() {
	r.rows.Close()
}

func (r *pgxRows) Err() error {
	return r.rows.Err()
}

// pgxResult wraps pgconn.CommandTag to implement db.Result
type pgxResult struct {
	tag pgconn.CommandTag
}

func (r *pgxResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

func (r *pgxResult) LastInsertId() (int64, error) {
	// PostgreSQL doesn't support LastInsertId in the same way as MySQL
	// Users should use RETURNING clause instead
	return 0, fmt.Errorf("LastInsertId not supported in PostgreSQL, use RETURNING clause")
}

// pgxTx wraps pgx.Tx to implement db.Tx
type pgxTx struct {
	tx pgx.Tx
}

func (t *pgxTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	tag, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

func (t *pgxTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

func (t *pgxTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &pgxRow{row: t.tx.QueryRow(ctx, query, args...)}
}

func (t *pgxTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgxTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Query executes a query that returns rows
func (p *PostgresProvider) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (p *PostgresProvider) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &pgxRow{row: p.pool.QueryRow(ctx, query, args...)}
}

// Exec executes a query that doesn't return rows
func (p *PostgresProvider) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	tag, err := p.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

// Begin starts a new transaction
func (p *PostgresProvider) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{tx: tx}, nil
}

// ========== CORE USER OPERATIONS ==========

// CreateUser creates a new user
func (p *PostgresProvider) CreateUser(ctx context.Context) (*models.User, error) {
	user := &models.User{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := p.pool.QueryRow(ctx, `
		INSERT INTO auth.user (created_at, updated_at)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`, user.CreatedAt, user.UpdatedAt).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (p *PostgresProvider) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	err := p.pool.QueryRow(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user
func (p *PostgresProvider) UpdateUser(ctx context.Context, user *models.User) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE auth.user
		SET updated_at = NOW()
		WHERE id = $1
	`, user.ID)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user and all associated data (cascades to sessions, etc.)
func (p *PostgresProvider) DeleteUser(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx, "DELETE FROM auth.user WHERE id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a paginated list of users
func (p *PostgresProvider) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
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
func (p *PostgresProvider) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM auth.user").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// ========== SESSION OPERATIONS ==========

// CreateSession creates a new session
func (p *PostgresProvider) CreateSession(ctx context.Context, session *models.Session) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO auth.session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, session.ID, session.UserID, session.Token, session.RefreshToken, session.ExpiresAt, session.CreatedAt, session.IPAddress, session.UserAgent)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (p *PostgresProvider) GetSession(ctx context.Context, token string) (*models.Session, error) {
	session := &models.Session{}

	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE token = $1
	`, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (p *PostgresProvider) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	session := &models.Session{}

	err := p.pool.QueryRow(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE refresh_token = $1
	`, refreshToken).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// UpdateSession updates a session
func (p *PostgresProvider) UpdateSession(ctx context.Context, session *models.Session) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE auth.session
		SET expires_at = $1
		WHERE id = $2
	`, session.ExpiresAt, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteSession deletes a session by token
func (p *PostgresProvider) DeleteSession(ctx context.Context, token string) error {
	_, err := p.pool.Exec(ctx, "DELETE FROM auth.session WHERE token = $1", token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all sessions for a user
func (p *PostgresProvider) GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent
		FROM auth.session
		WHERE user_id = $1
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
func (p *PostgresProvider) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx, "DELETE FROM auth.session WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}
