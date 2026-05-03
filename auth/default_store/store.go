// Package defaultstore implements the SQL-backed default store for the Aegis authentication module.
package defaultstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	types "github.com/theinventorylib/aegis/auth/types"
)

// DefaultStore holds all default SQL-based store implementations.
// It provides a complete implementation of all storage interfaces using
// sqlc-generated database queries supporting PostgreSQL, MySQL, and SQLite.
//
// In addition to the four store interfaces, DefaultStore implements
// types.Transactor so callers can compose multiple writes into a single
// database transaction (used, for example, by AccountService.UpdatePassword
// to atomically rotate the password and revoke all active sessions).
type DefaultStore struct {
	q  querier
	db *sql.DB
}

// NewDefaultStore creates a new default store for the given dialect.
// The dialect switch happens exactly once here; all store methods call
// through the querier interface and are dialect-agnostic.
func NewDefaultStore(db *sql.DB, dialect types.Dialect) (*DefaultStore, error) {
	var q querier
	switch dialect {
	case types.DialectPostgres:
		q = newPostgresQuerier(db)
	case types.DialectMySQL:
		q = newMysqlQuerier(db)
	case types.DialectSQLite:
		q = newSqliteQuerier(db)
	default:
		return nil, fmt.Errorf("auth: unsupported dialect %q", dialect)
	}
	return &DefaultStore{q: q, db: db}, nil
}

// UserStore returns the default user store implementation.
func (s *DefaultStore) UserStore() types.UserStore { return &defaultUserStore{q: s.q} }

// AccountStore returns the default account store implementation.
func (s *DefaultStore) AccountStore() types.AccountStore { return &defaultAccountStore{q: s.q} }

// VerificationStore returns the default verification store implementation.
func (s *DefaultStore) VerificationStore() types.VerificationStore {
	return &defaultVerificationStore{q: s.q}
}

// SessionStore returns the default session store implementation.
func (s *DefaultStore) SessionStore() types.SessionStore { return &defaultSessionStore{q: s.q} }

// BeginTx opens a database transaction and returns a types.Tx whose
// store accessors are bound to the transaction. The caller must Commit
// or Rollback the returned Tx; using `defer tx.Rollback()` immediately
// after a successful BeginTx call is safe because Rollback after Commit
// is a no-op.
func (s *DefaultStore) BeginTx(ctx context.Context) (types.Tx, error) {
	if s.db == nil {
		return nil, fmt.Errorf("auth: default store has no *sql.DB; transactions unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &defaultTx{tx: tx, q: s.q.withTx(tx)}, nil
}

// defaultTx is the sql.Tx-backed implementation of types.Tx returned by
// DefaultStore.BeginTx. The Tx is single-use: once Commit or Rollback is
// called the tx is finished and additional calls to Rollback are no-ops.
type defaultTx struct {
	tx   *sql.Tx
	q    querier
	done bool
}

func (t *defaultTx) AccountStore() types.AccountStore { return &defaultAccountStore{q: t.q} }
func (t *defaultTx) SessionStore() types.SessionStore { return &defaultSessionStore{q: t.q} }
func (t *defaultTx) UserStore() types.UserStore       { return &defaultUserStore{q: t.q} }
func (t *defaultTx) VerificationStore() types.VerificationStore {
	return &defaultVerificationStore{q: t.q}
}

func (t *defaultTx) Commit() error {
	if t.done {
		return fmt.Errorf("auth: tx already finalized")
	}
	t.done = true
	return t.tx.Commit()
}

func (t *defaultTx) Rollback() error {
	if t.done {
		return nil
	}
	t.done = true
	return t.tx.Rollback()
}

type defaultUserStore struct{ q querier }

type defaultAccountStore struct{ q querier }

type defaultVerificationStore struct{ q querier }

type defaultSessionStore struct{ q querier }

// UserStore implementation

func (s *defaultUserStore) Create(ctx context.Context, user types.User) (types.User, error) {
	err := s.q.createUser(ctx,
		user.ID, toNullString(user.Avatar), user.Name, toNullString(user.Email),
		user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339), user.Disabled,
	)
	if err != nil {
		return types.User{}, err
	}
	return user, nil
}

func (s *defaultUserStore) GetByEmail(ctx context.Context, email string) (types.User, error) {
	row, err := s.q.getUserByEmail(ctx, toNullString(email))
	if err != nil {
		return types.User{}, err
	}
	return buildUser(row.ID, row.Avatar, row.Name, row.Email, row.CreatedAt, row.UpdatedAt, row.Disabled), nil
}

func (s *defaultUserStore) GetByID(ctx context.Context, id string) (types.User, error) {
	row, err := s.q.getUserByID(ctx, id)
	if err != nil {
		return types.User{}, err
	}
	return buildUser(row.ID, row.Avatar, row.Name, row.Email, row.CreatedAt, row.UpdatedAt, row.Disabled), nil
}

func (s *defaultUserStore) Update(ctx context.Context, user types.User) error {
	return s.q.updateUser(ctx,
		user.ID, toNullString(user.Avatar), user.Name, toNullString(user.Email),
		user.UpdatedAt.Format(time.RFC3339), user.Disabled,
	)
}

func (s *defaultUserStore) Delete(ctx context.Context, id string) error {
	return s.q.deleteUser(ctx, id, time.Now().Format(time.RFC3339))
}

func (s *defaultUserStore) List(ctx context.Context, offset, limit int) ([]types.User, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	// ParsePagination already bounds these values for HTTP callers, but this
	// store is also usable directly; keep the int32 clamps as defense-in-depth.
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	rows, err := s.q.listUsers(ctx, int32(offset), int32(limit)) // #nosec G115 – bounds checked above
	if err != nil {
		return nil, err
	}
	result := make([]types.User, len(rows))
	for i, row := range rows {
		result[i] = buildUser(row.ID, row.Avatar, row.Name, row.Email, row.CreatedAt, row.UpdatedAt, row.Disabled)
	}
	return result, nil
}

func (s *defaultUserStore) Count(ctx context.Context) (int, error) {
	n, err := s.q.countUsers(ctx)
	return int(n), err
}

// AccountStore implementation

func (s *defaultAccountStore) Create(ctx context.Context, account types.Account) error {
	return s.q.createAccount(ctx,
		account.ID, account.UserID, account.Provider,
		toNullString(account.ProviderAccountID), toNullString(account.PasswordHash),
		toNullString(account.AccessToken), toNullString(account.RefreshToken),
		toNullString(account.ExpiresAt.Format(time.RFC3339)),
		account.CreatedAt.Format(time.RFC3339), account.UpdatedAt.Format(time.RFC3339),
	)
}

func (s *defaultAccountStore) GetByID(ctx context.Context, id string) (types.Account, error) {
	row, err := s.q.getAccountByID(ctx, id)
	if err != nil {
		return types.Account{}, err
	}
	return buildAccount(row.ID, row.UserID, row.Provider, row.ProviderAccountID, row.PasswordHash, row.AccessToken, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.UpdatedAt), nil
}

func (s *defaultAccountStore) GetByUserID(ctx context.Context, userID string) ([]types.Account, error) {
	rows, err := s.q.getAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]types.Account, len(rows))
	for i, row := range rows {
		result[i] = buildAccount(row.ID, row.UserID, row.Provider, row.ProviderAccountID, row.PasswordHash, row.AccessToken, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.UpdatedAt)
	}
	return result, nil
}

func (s *defaultAccountStore) GetByProvider(ctx context.Context, provider, providerAccountID string) (types.Account, error) {
	row, err := s.q.getAccountByProvider(ctx, provider, toNullString(providerAccountID))
	if err != nil {
		return types.Account{}, err
	}
	return buildAccount(row.ID, row.UserID, row.Provider, row.ProviderAccountID, row.PasswordHash, row.AccessToken, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.UpdatedAt), nil
}

func (s *defaultAccountStore) Update(ctx context.Context, account types.Account) error {
	return s.q.updateAccount(ctx,
		account.ID,
		toNullString(account.AccessToken), toNullString(account.RefreshToken),
		toNullString(account.ExpiresAt.Format(time.RFC3339)),
		account.UpdatedAt.Format(time.RFC3339),
	)
}

func (s *defaultAccountStore) Delete(ctx context.Context, id string) error {
	return s.q.deleteAccount(ctx, id)
}

// VerificationStore implementation

func (s *defaultVerificationStore) Create(ctx context.Context, verification types.Verification) error {
	return s.q.createVerification(ctx,
		verification.ID, verification.Identifier, verification.Token, verification.Type,
		verification.ExpiresAt.Format(time.RFC3339), verification.CreatedAt.Format(time.RFC3339),
	)
}

func (s *defaultVerificationStore) GetByToken(ctx context.Context, token string) (types.Verification, error) {
	row, err := s.q.getVerificationByToken(ctx, token, time.Now().Format(time.RFC3339))
	if err != nil {
		return types.Verification{}, err
	}
	return buildVerification(row.ID, row.Identifier, row.Token, row.Type, row.ExpiresAt, row.CreatedAt), nil
}

func (s *defaultVerificationStore) GetByIdentifier(ctx context.Context, identifier string) ([]types.Verification, error) {
	rows, err := s.q.getVerificationsByIdentifier(ctx, identifier, time.Now().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	result := make([]types.Verification, len(rows))
	for i, row := range rows {
		result[i] = buildVerification(row.ID, row.Identifier, row.Token, row.Type, row.ExpiresAt, row.CreatedAt)
	}
	return result, nil
}

func (s *defaultVerificationStore) InvalidateByIdentifier(ctx context.Context, identifier, vType string) error {
	return s.q.invalidateVerificationByIdentifier(ctx, identifier, vType, time.Now().Format(time.RFC3339))
}

func (s *defaultVerificationStore) Delete(ctx context.Context, id string) error {
	return s.q.deleteVerification(ctx, id)
}

func (s *defaultVerificationStore) CleanupExpired(ctx context.Context) error {
	return s.q.cleanupExpiredVerifications(ctx, time.Now().Format(time.RFC3339))
}

// SessionStore implementation

func (s *defaultSessionStore) Create(ctx context.Context, session types.Session) error {
	return s.q.createSession(ctx,
		session.ID, session.UserID, session.Token, toNullString(session.RefreshToken),
		session.ExpiresAt.Format(time.RFC3339), session.CreatedAt.Format(time.RFC3339),
		toNullString(session.IPAddress), toNullString(session.UserAgent),
	)
}

func (s *defaultSessionStore) Get(ctx context.Context, id string) (types.Session, error) {
	row, err := s.q.getSession(ctx, id, time.Now().Format(time.RFC3339))
	if err != nil {
		return types.Session{}, err
	}
	return buildSession(row.ID, row.UserID, row.Token, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.IPAddress, row.UserAgent), nil
}

func (s *defaultSessionStore) GetByToken(ctx context.Context, token string) (types.Session, error) {
	row, err := s.q.getSessionByToken(ctx, token, time.Now().Format(time.RFC3339))
	if err != nil {
		return types.Session{}, err
	}
	return buildSession(row.ID, row.UserID, row.Token, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.IPAddress, row.UserAgent), nil
}

func (s *defaultSessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (types.Session, error) {
	row, err := s.q.getSessionByRefreshToken(ctx, toNullString(refreshToken), time.Now().Format(time.RFC3339))
	if err != nil {
		return types.Session{}, err
	}
	return buildSession(row.ID, row.UserID, row.Token, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.IPAddress, row.UserAgent), nil
}

func (s *defaultSessionStore) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]types.Session, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	rows, err := s.q.getSessionsByUserID(ctx, userID, time.Now().Format(time.RFC3339), int32(offset), int32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]types.Session, len(rows))
	for i, row := range rows {
		result[i] = buildSession(row.ID, row.UserID, row.Token, row.RefreshToken, row.ExpiresAt, row.CreatedAt, row.IPAddress, row.UserAgent)
	}
	return result, nil
}

func (s *defaultSessionStore) CountByUserID(ctx context.Context, userID string) (int, error) {
	n, err := s.q.countSessionsByUserID(ctx, userID, time.Now().Format(time.RFC3339))
	return int(n), err
}

func (s *defaultSessionStore) Update(ctx context.Context, session types.Session) error {
	return s.q.updateSession(ctx, session.ID, toNullString(session.RefreshToken), session.ExpiresAt.Format(time.RFC3339))
}

func (s *defaultSessionStore) Delete(ctx context.Context, id string) error {
	return s.q.deleteSession(ctx, id)
}

func (s *defaultSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	return s.q.deleteSessionsByUserID(ctx, userID)
}

func (s *defaultSessionStore) CleanupExpired(ctx context.Context) error {
	return s.q.cleanupExpiredSessions(ctx, time.Now().Format(time.RFC3339))
}

// Helper functions for converting row fields to domain models.

func buildUser(id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool) types.User {
	return types.User{
		ID:        id,
		Avatar:    fromNullString(avatar),
		Name:      name,
		Email:     fromNullString(email),
		CreatedAt: parseTime(createdAt),
		UpdatedAt: parseTime(updatedAt),
		Disabled:  disabled,
	}
}

func buildAccount(id, userID, provider string, providerAccountID, passwordHash, accessToken, refreshToken, expiresAt sql.NullString, createdAt, updatedAt string) types.Account {
	return types.Account{
		ID:                id,
		UserID:            userID,
		Provider:          provider,
		ProviderAccountID: fromNullString(providerAccountID),
		PasswordHash:      fromNullString(passwordHash),
		AccessToken:       fromNullString(accessToken),
		RefreshToken:      fromNullString(refreshToken),
		ExpiresAt:         parseTime(fromNullString(expiresAt)),
		CreatedAt:         parseTime(createdAt),
		UpdatedAt:         parseTime(updatedAt),
	}
}

func buildVerification(id, identifier, token, vType, expiresAt, createdAt string) types.Verification {
	return types.Verification{
		ID:         id,
		Identifier: identifier,
		Token:      token,
		Type:       vType,
		ExpiresAt:  parseTime(expiresAt),
		CreatedAt:  parseTime(createdAt),
	}
}

func buildSession(id, userID, token string, refreshToken sql.NullString, expiresAt, createdAt string, ipAddress, userAgent sql.NullString) types.Session {
	return types.Session{
		ID:           id,
		UserID:       userID,
		Token:        token,
		RefreshToken: fromNullString(refreshToken),
		ExpiresAt:    parseTime(expiresAt),
		CreatedAt:    parseTime(createdAt),
		IPAddress:    fromNullString(ipAddress),
		UserAgent:    fromNullString(userAgent),
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func boolToInt[T int8 | int32 | int64](b bool) T {
	if b {
		return 1
	}
	return 0
}
