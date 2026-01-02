package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/theinventorylib/aegis/auth/internal/gen/sqlc"
)

// DefaultStore holds all default SQL-based store implementations.
// It provides a complete implementation of all storage interfaces using
// sqlc-generated database queries.
//
// This is the standard storage backend for Aegis and supports both
// PostgreSQL and MySQL databases through dialect-specific SQL schemas.
type DefaultStore struct {
	userStore         UserStore
	accountStore      AccountStore
	verificationStore VerificationStore
	sessionStore      SessionStore
}

// NewDefaultStore creates a new default store with all SQL-based implementations.
// The provided database connection is used for all queries. The connection should
// already be configured with the appropriate driver (postgres or mysql) and the
// schema tables should exist (use GetMigrations to set up the schema).
func NewDefaultStore(db *sql.DB) *DefaultStore {
	q := sqlc.New(db)
	return &DefaultStore{
		userStore:         &defaultUserStore{q: q},
		accountStore:      &defaultAccountStore{q: q},
		verificationStore: &defaultVerificationStore{q: q},
		sessionStore:      &defaultSessionStore{q: q},
	}
}

// UserStore returns the default user store implementation.
func (s *DefaultStore) UserStore() UserStore {
	return s.userStore
}

// AccountStore returns the default account store implementation.
func (s *DefaultStore) AccountStore() AccountStore {
	return s.accountStore
}

// VerificationStore returns the default verification store implementation.
func (s *DefaultStore) VerificationStore() VerificationStore {
	return s.verificationStore
}

// SessionStore returns the default session store implementation.
func (s *DefaultStore) SessionStore() SessionStore {
	return s.sessionStore
}

// defaultUserStore is the SQL-based implementation of UserStore.
type defaultUserStore struct {
	q *sqlc.Queries
}

// defaultAccountStore is the SQL-based implementation of AccountStore.
type defaultAccountStore struct {
	q *sqlc.Queries
}

// defaultVerificationStore is the SQL-based implementation of VerificationStore.
type defaultVerificationStore struct {
	q *sqlc.Queries
}

// defaultSessionStore is the SQL-based implementation of SessionStore.
type defaultSessionStore struct {
	q *sqlc.Queries
}

// UserStore implementation

func (s *defaultUserStore) Create(ctx context.Context, user User) (User, error) {
	params := sqlc.CreateUserParams{
		ID:        user.ID,
		Avatar:    toNullString(user.Avatar),
		Name:      user.Name,
		Email:     toNullString(user.Email),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		Disabled:  boolToInt(user.Disabled),
	}
	err := s.q.CreateUser(ctx, params)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *defaultUserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	u, err := s.q.GetUserByEmail(ctx, toNullString(email))
	if err != nil {
		return User{}, err
	}
	return sqlcUserToUser(u), nil
}

func (s *defaultUserStore) GetByID(ctx context.Context, id string) (User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return sqlcUserToUser(u), nil
}

func (s *defaultUserStore) Update(ctx context.Context, user User) error {
	params := sqlc.UpdateUserParams{
		ID:        user.ID,
		Avatar:    toNullString(user.Avatar),
		Name:      user.Name,
		Email:     toNullString(user.Email),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		Disabled:  boolToInt(user.Disabled),
	}
	return s.q.UpdateUser(ctx, params)
}

func (s *defaultUserStore) Delete(ctx context.Context, id string) error {
	return s.q.DeleteUser(ctx, sqlc.DeleteUserParams{
		ID:        id,
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
}

func (s *defaultUserStore) List(ctx context.Context, offset, limit int) ([]User, error) {
	users, err := s.q.ListUsers(ctx, sqlc.ListUsersParams{
		Offset: int32(offset),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]User, len(users))
	for i, u := range users {
		result[i] = sqlcUserToUser(u)
	}
	return result, nil
}

func (s *defaultUserStore) Count(ctx context.Context) (int, error) {
	count, err := s.q.CountUsers(ctx)
	return int(count), err
}

// AccountStore implementation

func (s *defaultAccountStore) Create(ctx context.Context, account Account) error {
	params := sqlc.CreateAccountParams{
		ID:                account.ID,
		UserID:            account.UserID,
		Provider:          account.Provider,
		ProviderAccountID: toNullString(account.ProviderAccountID),
		PasswordHash:      toNullString(account.PasswordHash),
		AccessToken:       toNullString(account.AccessToken),
		RefreshToken:      toNullString(account.RefreshToken),
		ExpiresAt:         toNullString(account.ExpiresAt.Format(time.RFC3339)),
		CreatedAt:         account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.Format(time.RFC3339),
	}
	return s.q.CreateAccount(ctx, params)
}

func (s *defaultAccountStore) GetByID(ctx context.Context, id string) (Account, error) {
	a, err := s.q.GetAccountByID(ctx, id)
	if err != nil {
		return Account{}, err
	}
	return sqlcAccountToAccount(a), nil
}

func (s *defaultAccountStore) GetByUserID(ctx context.Context, userID string) ([]Account, error) {
	accounts, err := s.q.GetAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Account, len(accounts))
	for i, a := range accounts {
		result[i] = sqlcAccountToAccount(a)
	}
	return result, nil
}

func (s *defaultAccountStore) GetByProvider(ctx context.Context, provider, providerAccountID string) (Account, error) {
	a, err := s.q.GetAccountByProvider(ctx, sqlc.GetAccountByProviderParams{
		Provider:          provider,
		ProviderAccountID: toNullString(providerAccountID),
	})
	if err != nil {
		return Account{}, err
	}
	return sqlcAccountToAccount(a), nil
}

func (s *defaultAccountStore) Update(ctx context.Context, account Account) error {
	params := sqlc.UpdateAccountParams{
		ID:           account.ID,
		AccessToken:  toNullString(account.AccessToken),
		RefreshToken: toNullString(account.RefreshToken),
		ExpiresAt:    toNullString(account.ExpiresAt.Format(time.RFC3339)),
		UpdatedAt:    account.UpdatedAt.Format(time.RFC3339),
	}
	return s.q.UpdateAccount(ctx, params)
}

func (s *defaultAccountStore) Delete(ctx context.Context, id string) error {
	return s.q.DeleteAccount(ctx, id)
}

// VerificationStore implementation

func (s *defaultVerificationStore) Create(ctx context.Context, verification Verification) error {
	params := sqlc.CreateVerificationParams{
		ID:         verification.ID,
		Identifier: verification.Identifier,
		Token:      verification.Token,
		Type:       verification.Type,
		ExpiresAt:  verification.ExpiresAt.Format(time.RFC3339),
		CreatedAt:  verification.CreatedAt.Format(time.RFC3339),
	}
	return s.q.CreateVerification(ctx, params)
}

func (s *defaultVerificationStore) GetByToken(ctx context.Context, token string) (Verification, error) {
	v, err := s.q.GetVerificationByToken(ctx, sqlc.GetVerificationByTokenParams{
		Token:     token,
		ExpiresAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return Verification{}, err
	}
	return sqlcVerificationToVerification(v), nil
}

func (s *defaultVerificationStore) GetByIdentifier(ctx context.Context, identifier string) ([]Verification, error) {
	verifications, err := s.q.GetVerificationsByIdentifier(ctx, sqlc.GetVerificationsByIdentifierParams{
		Identifier: identifier,
		ExpiresAt:  time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	result := make([]Verification, len(verifications))
	for i, v := range verifications {
		result[i] = sqlcVerificationToVerification(v)
	}
	return result, nil
}

func (s *defaultVerificationStore) InvalidateByIdentifier(ctx context.Context, identifier, vType string) error {
	return s.q.InvalidateVerificationByIdentifier(ctx, sqlc.InvalidateVerificationByIdentifierParams{
		Identifier: identifier,
		Type:       vType,
		ExpiresAt:  time.Now().Format(time.RFC3339),
	})
}

func (s *defaultVerificationStore) Delete(ctx context.Context, id string) error {
	return s.q.DeleteVerification(ctx, id)
}

func (s *defaultVerificationStore) CleanupExpired(ctx context.Context) error {
	return s.q.CleanupExpiredVerifications(ctx, time.Now().Format(time.RFC3339))
}

// SessionStore implementation

func (s *defaultSessionStore) Create(ctx context.Context, session Session) error {
	params := sqlc.CreateSessionParams{
		ID:           session.ID,
		UserID:       session.UserID,
		Token:        session.Token,
		RefreshToken: toNullString(session.RefreshToken),
		ExpiresAt:    session.ExpiresAt.Format(time.RFC3339),
		CreatedAt:    session.CreatedAt.Format(time.RFC3339),
		IpAddress:    toNullString(session.IPAddress),
		UserAgent:    toNullString(session.UserAgent),
	}
	return s.q.CreateSession(ctx, params)
}

func (s *defaultSessionStore) Get(ctx context.Context, id string) (Session, error) {
	sess, err := s.q.GetSession(ctx, sqlc.GetSessionParams{
		ID:        id,
		ExpiresAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return Session{}, err
	}
	return sqlcSessionToSession(sess), nil
}

func (s *defaultSessionStore) GetByToken(ctx context.Context, token string) (Session, error) {
	sess, err := s.q.GetSessionByToken(ctx, sqlc.GetSessionByTokenParams{
		Token:     token,
		ExpiresAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return Session{}, err
	}
	return sqlcSessionToSession(sess), nil
}

func (s *defaultSessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (Session, error) {
	sess, err := s.q.GetSessionByRefreshToken(ctx, sqlc.GetSessionByRefreshTokenParams{
		RefreshToken: toNullString(refreshToken),
		ExpiresAt:    time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return Session{}, err
	}
	return sqlcSessionToSession(sess), nil
}

func (s *defaultSessionStore) GetByUserID(ctx context.Context, userID string) ([]Session, error) {
	sessions, err := s.q.GetSessionsByUserID(ctx, sqlc.GetSessionsByUserIDParams{
		UserID:    userID,
		ExpiresAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	result := make([]Session, len(sessions))
	for i, sess := range sessions {
		result[i] = sqlcSessionToSession(sess)
	}
	return result, nil
}

func (s *defaultSessionStore) Update(ctx context.Context, session Session) error {
	params := sqlc.UpdateSessionParams{
		ID:           session.ID,
		RefreshToken: toNullString(session.RefreshToken),
		ExpiresAt:    session.ExpiresAt.Format(time.RFC3339),
	}
	return s.q.UpdateSession(ctx, params)
}

func (s *defaultSessionStore) Delete(ctx context.Context, id string) error {
	return s.q.DeleteSession(ctx, id)
}

func (s defaultSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	return s.q.DeleteSessionsByUserID(ctx, userID)
}

func (s *defaultSessionStore) CleanupExpired(ctx context.Context) error {
	return s.q.CleanupExpiredSessions(ctx, time.Now().Format(time.RFC3339))
}

// Helper functions for converting between domain models and sqlc-generated types.
// These functions handle the impedance mismatch between Go's type system and SQL's
// nullable types, as well as time.Time formatting for database storage.

// sqlcUserToUser converts a sqlc-generated User to the domain User model.
// Parses RFC3339 timestamps and converts SQL nullable fields to Go zero values.
func sqlcUserToUser(u sqlc.User) User {
	createdAt, _ := time.Parse(time.RFC3339, u.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, u.UpdatedAt)
	return User{
		ID:        u.ID,
		Avatar:    fromNullString(u.Avatar),
		Name:      u.Name,
		Email:     fromNullString(u.Email),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Disabled:  intToBool(u.Disabled),
	}
}

// sqlcAccountToAccount converts a sqlc-generated Account to the domain Account model.
// Handles nullable fields and timestamp parsing. ExpiresAt is parsed from a nullable
// string field (zero time if NULL).
func sqlcAccountToAccount(a sqlc.Account) Account {
	createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, a.UpdatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, fromNullString(a.ExpiresAt))
	return Account{
		ID:                a.ID,
		UserID:            a.UserID,
		Provider:          a.Provider,
		ProviderAccountID: fromNullString(a.ProviderAccountID),
		PasswordHash:      fromNullString(a.PasswordHash),
		AccessToken:       fromNullString(a.AccessToken),
		RefreshToken:      fromNullString(a.RefreshToken),
		ExpiresAt:         expiresAt,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

// sqlcVerificationToVerification converts a sqlc-generated Verification to the
// domain Verification model. Parses RFC3339 timestamps for expiry and creation.
func sqlcVerificationToVerification(v sqlc.Verification) Verification {
	expiresAt, _ := time.Parse(time.RFC3339, v.ExpiresAt)
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	return Verification{
		ID:         v.ID,
		Identifier: v.Identifier,
		Token:      v.Token,
		Type:       v.Type,
		ExpiresAt:  expiresAt,
		CreatedAt:  createdAt,
	}
}

// sqlcSessionToSession converts a sqlc-generated Session to the domain Session model.
// Handles nullable refresh tokens, IP addresses, and user agents.
func sqlcSessionToSession(s sqlc.Session) Session {
	expiresAt, _ := time.Parse(time.RFC3339, s.ExpiresAt)
	createdAt, _ := time.Parse(time.RFC3339, s.CreatedAt)
	return Session{
		ID:           s.ID,
		UserID:       s.UserID,
		Token:        s.Token,
		RefreshToken: fromNullString(s.RefreshToken),
		ExpiresAt:    expiresAt,
		CreatedAt:    createdAt,
		IPAddress:    fromNullString(s.IpAddress),
		UserAgent:    fromNullString(s.UserAgent),
	}
}

// toNullString converts a Go string to a SQL nullable string.
// Empty strings are represented as SQL NULL to distinguish between
// "not provided" and "explicitly empty".
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// fromNullString converts a SQL nullable string to a Go string.
// NULL values become empty strings. This provides a zero-value
// representation for optional fields.
func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// boolToInt converts a Go bool to int32 for SQL storage.
// Many SQL databases don't have a native boolean type, so we use
// integers with 1 for true and 0 for false.
func boolToInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// intToBool converts an int32 from SQL storage to a Go bool.
// Any non-zero value is considered true.
func intToBool(i int32) bool {
	return i != 0
}
