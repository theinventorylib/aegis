package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// Booleans are stored as int8; pagination uses int32 (same as postgres).

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/auth/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMysqlQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

// User

func (m *mysqlQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool) error {
	return m.q.CreateUser(ctx, sqlcmysql.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt, Disabled: boolToInt[int8](disabled),
	})
}

func (m *mysqlQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (userRow, error) {
	u, err := m.q.GetUserByEmail(ctx, email)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}, nil
}

func (m *mysqlQuerier) getUserByID(ctx context.Context, id string) (userRow, error) {
	u, err := m.q.GetUserByID(ctx, id)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}, nil
}

func (m *mysqlQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return m.q.UpdateUser(ctx, sqlcmysql.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email, UpdatedAt: updatedAt, Disabled: boolToInt[int8](disabled),
	})
}

func (m *mysqlQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return m.q.DeleteUser(ctx, sqlcmysql.DeleteUserParams{ID: id, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) listUsers(ctx context.Context, offset, limit int32) ([]userRow, error) {
	rows, err := m.q.ListUsers(ctx, sqlcmysql.ListUsersParams{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]userRow, len(rows))
	for i, u := range rows {
		out[i] = userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (m *mysqlQuerier) countUsers(ctx context.Context) (int64, error) { return m.q.CountUsers(ctx) }

// Account

func (m *mysqlQuerier) createAccount(ctx context.Context, id, userID, provider string, providerAccountID, passwordHash, accessToken, refreshToken, expiresAt sql.NullString, createdAt, updatedAt string) error {
	return m.q.CreateAccount(ctx, sqlcmysql.CreateAccountParams{
		ID: id, UserID: userID, Provider: provider,
		ProviderAccountID: providerAccountID, PasswordHash: passwordHash,
		AccessToken: accessToken, RefreshToken: refreshToken,
		ExpiresAt: expiresAt, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getAccountByID(ctx context.Context, id string) (accountRow, error) {
	r, err := m.q.GetAccountByID(ctx, id)
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (m *mysqlQuerier) getAccountsByUserID(ctx context.Context, userID string) ([]accountRow, error) {
	rows, err := m.q.GetAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]accountRow, len(rows))
	for i, r := range rows {
		out[i] = accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return out, nil
}

func (m *mysqlQuerier) getAccountByProvider(ctx context.Context, provider string, providerAccountID sql.NullString) (accountRow, error) {
	r, err := m.q.GetAccountByProvider(ctx, sqlcmysql.GetAccountByProviderParams{Provider: provider, ProviderAccountID: providerAccountID})
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (m *mysqlQuerier) updateAccount(ctx context.Context, id string, accessToken, refreshToken, expiresAt sql.NullString, updatedAt string) error {
	return m.q.UpdateAccount(ctx, sqlcmysql.UpdateAccountParams{ID: id, AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) deleteAccount(ctx context.Context, id string) error {
	return m.q.DeleteAccount(ctx, id)
}

// Verification

func (m *mysqlQuerier) createVerification(ctx context.Context, id, identifier, token, vType, expiresAt, createdAt string) error {
	return m.q.CreateVerification(ctx, sqlcmysql.CreateVerificationParams{
		ID: id, Identifier: identifier, Token: token, Type: vType, ExpiresAt: expiresAt, CreatedAt: createdAt,
	})
}

func (m *mysqlQuerier) getVerificationByToken(ctx context.Context, token, expiresAt string) (verificationRow, error) {
	v, err := m.q.GetVerificationByToken(ctx, sqlcmysql.GetVerificationByTokenParams{Token: token, ExpiresAt: expiresAt})
	if err != nil {
		return verificationRow{}, err
	}
	return verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: v.ExpiresAt, CreatedAt: v.CreatedAt}, nil
}

func (m *mysqlQuerier) getVerificationsByIdentifier(ctx context.Context, identifier, expiresAt string) ([]verificationRow, error) {
	rows, err := m.q.GetVerificationsByIdentifier(ctx, sqlcmysql.GetVerificationsByIdentifierParams{Identifier: identifier, ExpiresAt: expiresAt})
	if err != nil {
		return nil, err
	}
	out := make([]verificationRow, len(rows))
	for i, v := range rows {
		out[i] = verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: v.ExpiresAt, CreatedAt: v.CreatedAt}
	}
	return out, nil
}

func (m *mysqlQuerier) invalidateVerificationByIdentifier(ctx context.Context, identifier, vType, expiresAt string) error {
	return m.q.InvalidateVerificationByIdentifier(ctx, sqlcmysql.InvalidateVerificationByIdentifierParams{Identifier: identifier, Type: vType, ExpiresAt: expiresAt})
}

func (m *mysqlQuerier) deleteVerification(ctx context.Context, id string) error {
	return m.q.DeleteVerification(ctx, id)
}

func (m *mysqlQuerier) cleanupExpiredVerifications(ctx context.Context, now string) error {
	return m.q.CleanupExpiredVerifications(ctx, now)
}

// Session

func (m *mysqlQuerier) createSession(ctx context.Context, id, userID, token string, refreshToken sql.NullString, expiresAt, createdAt string, ipAddress, userAgent sql.NullString) error {
	return m.q.CreateSession(ctx, sqlcmysql.CreateSessionParams{
		ID: id, UserID: userID, Token: token, RefreshToken: refreshToken,
		ExpiresAt: expiresAt, CreatedAt: createdAt, IpAddress: ipAddress, UserAgent: userAgent,
	})
}

func (m *mysqlQuerier) getSession(ctx context.Context, id, expiresAt string) (sessionRow, error) {
	s, err := m.q.GetSession(ctx, sqlcmysql.GetSessionParams{ID: id, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (m *mysqlQuerier) getSessionByToken(ctx context.Context, token, expiresAt string) (sessionRow, error) {
	s, err := m.q.GetSessionByToken(ctx, sqlcmysql.GetSessionByTokenParams{Token: token, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (m *mysqlQuerier) getSessionByRefreshToken(ctx context.Context, refreshToken sql.NullString, expiresAt string) (sessionRow, error) {
	s, err := m.q.GetSessionByRefreshToken(ctx, sqlcmysql.GetSessionByRefreshTokenParams{RefreshToken: refreshToken, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (m *mysqlQuerier) getSessionsByUserID(ctx context.Context, userID, expiresAt string, offset, limit int32) ([]sessionRow, error) {
	rows, err := m.q.GetSessionsByUserID(ctx, sqlcmysql.GetSessionsByUserIDParams{UserID: userID, ExpiresAt: expiresAt, Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]sessionRow, len(rows))
	for i, s := range rows {
		out[i] = sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, IPAddress: s.IpAddress, UserAgent: s.UserAgent}
	}
	return out, nil
}

func (m *mysqlQuerier) countSessionsByUserID(ctx context.Context, userID, expiresAt string) (int64, error) {
	return m.q.CountSessionsByUserID(ctx, sqlcmysql.CountSessionsByUserIDParams{UserID: userID, ExpiresAt: expiresAt})
}

func (m *mysqlQuerier) updateSession(ctx context.Context, id string, refreshToken sql.NullString, expiresAt string) error {
	return m.q.UpdateSession(ctx, sqlcmysql.UpdateSessionParams{ID: id, RefreshToken: refreshToken, ExpiresAt: expiresAt})
}

func (m *mysqlQuerier) deleteSession(ctx context.Context, id string) error {
	return m.q.DeleteSession(ctx, id)
}

func (m *mysqlQuerier) deleteSessionsByUserID(ctx context.Context, userID string) error {
	return m.q.DeleteSessionsByUserID(ctx, userID)
}

func (m *mysqlQuerier) cleanupExpiredSessions(ctx context.Context, now string) error {
	return m.q.CleanupExpiredSessions(ctx, now)
}

func (m *mysqlQuerier) withTx(tx *sql.Tx) querier {
	return &mysqlQuerier{q: m.q.WithTx(tx)}
}
