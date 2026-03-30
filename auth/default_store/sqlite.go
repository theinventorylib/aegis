package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// Booleans are stored as int64; pagination also uses int64 (cast from int32).

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/auth/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSqliteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

// User

func (s *sqliteQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool) error {
	return s.q.CreateUser(ctx, sqlcsqlite.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: createdAt, UpdatedAt: updatedAt, Disabled: boolToInt[int64](disabled),
	})
}

func (s *sqliteQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (userRow, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}, nil
}

func (s *sqliteQuerier) getUserByID(ctx context.Context, id string) (userRow, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}, nil
}

func (s *sqliteQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return s.q.UpdateUser(ctx, sqlcsqlite.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email, UpdatedAt: updatedAt, Disabled: boolToInt[int64](disabled),
	})
}

func (s *sqliteQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return s.q.DeleteUser(ctx, sqlcsqlite.DeleteUserParams{ID: id, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) listUsers(ctx context.Context, offset, limit int32) ([]userRow, error) {
	rows, err := s.q.ListUsers(ctx, sqlcsqlite.ListUsersParams{Offset: int64(offset), Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]userRow, len(rows))
	for i, u := range rows {
		out[i] = userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (s *sqliteQuerier) countUsers(ctx context.Context) (int64, error) { return s.q.CountUsers(ctx) }

// Account

func (s *sqliteQuerier) createAccount(ctx context.Context, id, userID, provider string, providerAccountID, passwordHash, accessToken, refreshToken, expiresAt sql.NullString, createdAt, updatedAt string) error {
	return s.q.CreateAccount(ctx, sqlcsqlite.CreateAccountParams{
		ID: id, UserID: userID, Provider: provider,
		ProviderAccountID: providerAccountID, PasswordHash: passwordHash,
		AccessToken: accessToken, RefreshToken: refreshToken,
		ExpiresAt: expiresAt, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) getAccountByID(ctx context.Context, id string) (accountRow, error) {
	r, err := s.q.GetAccountByID(ctx, id)
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (s *sqliteQuerier) getAccountsByUserID(ctx context.Context, userID string) ([]accountRow, error) {
	rows, err := s.q.GetAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]accountRow, len(rows))
	for i, r := range rows {
		out[i] = accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return out, nil
}

func (s *sqliteQuerier) getAccountByProvider(ctx context.Context, provider string, providerAccountID sql.NullString) (accountRow, error) {
	r, err := s.q.GetAccountByProvider(ctx, sqlcsqlite.GetAccountByProviderParams{Provider: provider, ProviderAccountID: providerAccountID})
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (s *sqliteQuerier) updateAccount(ctx context.Context, id string, accessToken, refreshToken, expiresAt sql.NullString, updatedAt string) error {
	return s.q.UpdateAccount(ctx, sqlcsqlite.UpdateAccountParams{ID: id, AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) deleteAccount(ctx context.Context, id string) error {
	return s.q.DeleteAccount(ctx, id)
}

// Verification

func (s *sqliteQuerier) createVerification(ctx context.Context, id, identifier, token, vType, expiresAt, createdAt string) error {
	return s.q.CreateVerification(ctx, sqlcsqlite.CreateVerificationParams{
		ID: id, Identifier: identifier, Token: token, Type: vType, ExpiresAt: expiresAt, CreatedAt: createdAt,
	})
}

func (s *sqliteQuerier) getVerificationByToken(ctx context.Context, token, expiresAt string) (verificationRow, error) {
	v, err := s.q.GetVerificationByToken(ctx, sqlcsqlite.GetVerificationByTokenParams{Token: token, ExpiresAt: expiresAt})
	if err != nil {
		return verificationRow{}, err
	}
	return verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: v.ExpiresAt, CreatedAt: v.CreatedAt}, nil
}

func (s *sqliteQuerier) getVerificationsByIdentifier(ctx context.Context, identifier, expiresAt string) ([]verificationRow, error) {
	rows, err := s.q.GetVerificationsByIdentifier(ctx, sqlcsqlite.GetVerificationsByIdentifierParams{Identifier: identifier, ExpiresAt: expiresAt})
	if err != nil {
		return nil, err
	}
	out := make([]verificationRow, len(rows))
	for i, v := range rows {
		out[i] = verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: v.ExpiresAt, CreatedAt: v.CreatedAt}
	}
	return out, nil
}

func (s *sqliteQuerier) invalidateVerificationByIdentifier(ctx context.Context, identifier, vType, expiresAt string) error {
	return s.q.InvalidateVerificationByIdentifier(ctx, sqlcsqlite.InvalidateVerificationByIdentifierParams{Identifier: identifier, Type: vType, ExpiresAt: expiresAt})
}

func (s *sqliteQuerier) deleteVerification(ctx context.Context, id string) error {
	return s.q.DeleteVerification(ctx, id)
}

func (s *sqliteQuerier) cleanupExpiredVerifications(ctx context.Context, now string) error {
	return s.q.CleanupExpiredVerifications(ctx, now)
}

// Session

func (s *sqliteQuerier) createSession(ctx context.Context, id, userID, token string, refreshToken sql.NullString, expiresAt, createdAt string, ipAddress, userAgent sql.NullString) error {
	return s.q.CreateSession(ctx, sqlcsqlite.CreateSessionParams{
		ID: id, UserID: userID, Token: token, RefreshToken: refreshToken,
		ExpiresAt: expiresAt, CreatedAt: createdAt, IpAddress: ipAddress, UserAgent: userAgent,
	})
}

func (s *sqliteQuerier) getSession(ctx context.Context, id, expiresAt string) (sessionRow, error) {
	r, err := s.q.GetSession(ctx, sqlcsqlite.GetSessionParams{ID: id, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: r.ID, UserID: r.UserID, Token: r.Token, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, IPAddress: r.IpAddress, UserAgent: r.UserAgent}, nil
}

func (s *sqliteQuerier) getSessionByToken(ctx context.Context, token, expiresAt string) (sessionRow, error) {
	r, err := s.q.GetSessionByToken(ctx, sqlcsqlite.GetSessionByTokenParams{Token: token, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: r.ID, UserID: r.UserID, Token: r.Token, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, IPAddress: r.IpAddress, UserAgent: r.UserAgent}, nil
}

func (s *sqliteQuerier) getSessionByRefreshToken(ctx context.Context, refreshToken sql.NullString, expiresAt string) (sessionRow, error) {
	r, err := s.q.GetSessionByRefreshToken(ctx, sqlcsqlite.GetSessionByRefreshTokenParams{RefreshToken: refreshToken, ExpiresAt: expiresAt})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: r.ID, UserID: r.UserID, Token: r.Token, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, IPAddress: r.IpAddress, UserAgent: r.UserAgent}, nil
}

func (s *sqliteQuerier) getSessionsByUserID(ctx context.Context, userID, expiresAt string, offset, limit int32) ([]sessionRow, error) {
	rows, err := s.q.GetSessionsByUserID(ctx, sqlcsqlite.GetSessionsByUserIDParams{UserID: userID, ExpiresAt: expiresAt, Offset: int64(offset), Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]sessionRow, len(rows))
	for i, r := range rows {
		out[i] = sessionRow{ID: r.ID, UserID: r.UserID, Token: r.Token, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, IPAddress: r.IpAddress, UserAgent: r.UserAgent}
	}
	return out, nil
}

func (s *sqliteQuerier) countSessionsByUserID(ctx context.Context, userID, expiresAt string) (int64, error) {
	return s.q.CountSessionsByUserID(ctx, sqlcsqlite.CountSessionsByUserIDParams{UserID: userID, ExpiresAt: expiresAt})
}

func (s *sqliteQuerier) updateSession(ctx context.Context, id string, refreshToken sql.NullString, expiresAt string) error {
	return s.q.UpdateSession(ctx, sqlcsqlite.UpdateSessionParams{ID: id, RefreshToken: refreshToken, ExpiresAt: expiresAt})
}

func (s *sqliteQuerier) deleteSession(ctx context.Context, id string) error {
	return s.q.DeleteSession(ctx, id)
}

func (s *sqliteQuerier) deleteSessionsByUserID(ctx context.Context, userID string) error {
	return s.q.DeleteSessionsByUserID(ctx, userID)
}

func (s *sqliteQuerier) cleanupExpiredSessions(ctx context.Context, now string) error {
	return s.q.CleanupExpiredSessions(ctx, now)
}
