package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// All dialect-specific types (int32 for booleans, time.Time for TIMESTAMPTZ)
// are handled here and nowhere else. Canonical rows keep timestamps as
// RFC3339 strings so mysql/sqlite translators stay unchanged.

import (
	"context"
	"database/sql"
	"log"
	"time"

	sqlcpostgres "github.com/theinventorylib/aegis/v2/auth/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

// parseTS converts a canonical RFC3339 string to the time.Time the generated
// postgres queries expect.
func parseTS(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Printf("aegis: ignoring malformed RFC3339 timestamp %q: %v", s, err)
		return time.Time{}
	}
	return t
}

// parseNullTS converts a nullable canonical string to sql.NullTime.
func parseNullTS(ns sql.NullString) sql.NullTime {
	if !ns.Valid {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parseTS(ns.String), Valid: true}
}

// formatTS converts a time.Time back to the canonical RFC3339 string.
func formatTS(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// formatNullTS converts sql.NullTime back to a nullable canonical string.
func formatNullTS(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTS(nt.Time), Valid: true}
}

// User

func (p *postgresQuerier) createUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, createdAt, updatedAt string, disabled bool) error {
	return p.q.CreateUser(ctx, sqlcpostgres.CreateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email,
		CreatedAt: parseTS(createdAt), UpdatedAt: parseTS(updatedAt), Disabled: boolToInt[int32](disabled),
	})
}

func (p *postgresQuerier) getUserByEmail(ctx context.Context, email sql.NullString) (userRow, error) {
	u, err := p.q.GetUserByEmail(ctx, email)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: formatTS(u.CreatedAt), UpdatedAt: formatTS(u.UpdatedAt), Disabled: u.Disabled != 0}, nil
}

func (p *postgresQuerier) getUserByID(ctx context.Context, id string) (userRow, error) {
	u, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		return userRow{}, err
	}
	return userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: formatTS(u.CreatedAt), UpdatedAt: formatTS(u.UpdatedAt), Disabled: u.Disabled != 0}, nil
}

func (p *postgresQuerier) updateUser(ctx context.Context, id string, avatar sql.NullString, name string, email sql.NullString, updatedAt string, disabled bool) error {
	return p.q.UpdateUser(ctx, sqlcpostgres.UpdateUserParams{
		ID: id, Avatar: avatar, Name: name, Email: email, UpdatedAt: parseTS(updatedAt), Disabled: boolToInt[int32](disabled),
	})
}

func (p *postgresQuerier) deleteUser(ctx context.Context, id, updatedAt string) error {
	return p.q.DeleteUser(ctx, sqlcpostgres.DeleteUserParams{ID: id, UpdatedAt: parseTS(updatedAt)})
}

func (p *postgresQuerier) listUsers(ctx context.Context, offset, limit int32) ([]userRow, error) {
	rows, err := p.q.ListUsers(ctx, sqlcpostgres.ListUsersParams{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]userRow, len(rows))
	for i, u := range rows {
		out[i] = userRow{ID: u.ID, Avatar: u.Avatar, Name: u.Name, Email: u.Email, CreatedAt: formatTS(u.CreatedAt), UpdatedAt: formatTS(u.UpdatedAt), Disabled: u.Disabled != 0}
	}
	return out, nil
}

func (p *postgresQuerier) countUsers(ctx context.Context) (int64, error) { return p.q.CountUsers(ctx) }

// Account

func (p *postgresQuerier) createAccount(ctx context.Context, id, userID, provider string, providerAccountID, passwordHash, accessToken, refreshToken, expiresAt sql.NullString, createdAt, updatedAt string) error {
	return p.q.CreateAccount(ctx, sqlcpostgres.CreateAccountParams{
		ID: id, UserID: userID, Provider: provider,
		ProviderAccountID: providerAccountID, PasswordHash: passwordHash,
		AccessToken: accessToken, RefreshToken: refreshToken,
		ExpiresAt: parseNullTS(expiresAt), CreatedAt: parseTS(createdAt), UpdatedAt: parseTS(updatedAt),
	})
}

func (p *postgresQuerier) getAccountByID(ctx context.Context, id string) (accountRow, error) {
	r, err := p.q.GetAccountByID(ctx, id)
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: formatNullTS(r.ExpiresAt), CreatedAt: formatTS(r.CreatedAt), UpdatedAt: formatTS(r.UpdatedAt)}, nil
}

func (p *postgresQuerier) getAccountsByUserID(ctx context.Context, userID string) ([]accountRow, error) {
	rows, err := p.q.GetAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]accountRow, len(rows))
	for i, r := range rows {
		out[i] = accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: formatNullTS(r.ExpiresAt), CreatedAt: formatTS(r.CreatedAt), UpdatedAt: formatTS(r.UpdatedAt)}
	}
	return out, nil
}

func (p *postgresQuerier) getAccountByProvider(ctx context.Context, provider string, providerAccountID sql.NullString) (accountRow, error) {
	r, err := p.q.GetAccountByProvider(ctx, sqlcpostgres.GetAccountByProviderParams{Provider: provider, ProviderAccountID: providerAccountID})
	if err != nil {
		return accountRow{}, err
	}
	return accountRow{ID: r.ID, UserID: r.UserID, Provider: r.Provider, ProviderAccountID: r.ProviderAccountID, PasswordHash: r.PasswordHash, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: formatNullTS(r.ExpiresAt), CreatedAt: formatTS(r.CreatedAt), UpdatedAt: formatTS(r.UpdatedAt)}, nil
}

func (p *postgresQuerier) updateAccount(ctx context.Context, id string, accessToken, refreshToken, expiresAt sql.NullString, updatedAt string) error {
	return p.q.UpdateAccount(ctx, sqlcpostgres.UpdateAccountParams{ID: id, AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: parseNullTS(expiresAt), UpdatedAt: parseTS(updatedAt)})
}

func (p *postgresQuerier) deleteAccount(ctx context.Context, id string) error {
	return p.q.DeleteAccount(ctx, id)
}

// Verification

func (p *postgresQuerier) createVerification(ctx context.Context, id, identifier, token, vType, expiresAt, createdAt string) error {
	return p.q.CreateVerification(ctx, sqlcpostgres.CreateVerificationParams{
		ID: id, Identifier: identifier, Token: token, Type: vType, ExpiresAt: parseTS(expiresAt), CreatedAt: parseTS(createdAt),
	})
}

func (p *postgresQuerier) getVerificationByToken(ctx context.Context, token, expiresAt string) (verificationRow, error) {
	v, err := p.q.GetVerificationByToken(ctx, sqlcpostgres.GetVerificationByTokenParams{Token: token, ExpiresAt: parseTS(expiresAt)})
	if err != nil {
		return verificationRow{}, err
	}
	return verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: formatTS(v.ExpiresAt), CreatedAt: formatTS(v.CreatedAt)}, nil
}

func (p *postgresQuerier) getVerificationsByIdentifier(ctx context.Context, identifier, expiresAt string) ([]verificationRow, error) {
	rows, err := p.q.GetVerificationsByIdentifier(ctx, sqlcpostgres.GetVerificationsByIdentifierParams{Identifier: identifier, ExpiresAt: parseTS(expiresAt)})
	if err != nil {
		return nil, err
	}
	out := make([]verificationRow, len(rows))
	for i, v := range rows {
		out[i] = verificationRow{ID: v.ID, Identifier: v.Identifier, Token: v.Token, Type: v.Type, ExpiresAt: formatTS(v.ExpiresAt), CreatedAt: formatTS(v.CreatedAt)}
	}
	return out, nil
}

func (p *postgresQuerier) invalidateVerificationByIdentifier(ctx context.Context, identifier, vType, expiresAt string) error {
	return p.q.InvalidateVerificationByIdentifier(ctx, sqlcpostgres.InvalidateVerificationByIdentifierParams{Identifier: identifier, Type: vType, ExpiresAt: parseTS(expiresAt)})
}

func (p *postgresQuerier) deleteVerification(ctx context.Context, id string) error {
	return p.q.DeleteVerification(ctx, id)
}

func (p *postgresQuerier) cleanupExpiredVerifications(ctx context.Context, now string) error {
	return p.q.CleanupExpiredVerifications(ctx, parseTS(now))
}

// Session

func (p *postgresQuerier) createSession(ctx context.Context, id, userID, token string, refreshToken sql.NullString, expiresAt, createdAt string, ipAddress, userAgent sql.NullString) error {
	return p.q.CreateSession(ctx, sqlcpostgres.CreateSessionParams{
		ID: id, UserID: userID, Token: token, RefreshToken: refreshToken,
		ExpiresAt: parseTS(expiresAt), CreatedAt: parseTS(createdAt), IpAddress: ipAddress, UserAgent: userAgent,
	})
}

func (p *postgresQuerier) getSession(ctx context.Context, id, expiresAt string) (sessionRow, error) {
	s, err := p.q.GetSession(ctx, sqlcpostgres.GetSessionParams{ID: id, ExpiresAt: parseTS(expiresAt)})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: formatTS(s.ExpiresAt), CreatedAt: formatTS(s.CreatedAt), IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (p *postgresQuerier) getSessionByToken(ctx context.Context, token, expiresAt string) (sessionRow, error) {
	s, err := p.q.GetSessionByToken(ctx, sqlcpostgres.GetSessionByTokenParams{Token: token, ExpiresAt: parseTS(expiresAt)})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: formatTS(s.ExpiresAt), CreatedAt: formatTS(s.CreatedAt), IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (p *postgresQuerier) getSessionByRefreshToken(ctx context.Context, refreshToken sql.NullString, expiresAt string) (sessionRow, error) {
	s, err := p.q.GetSessionByRefreshToken(ctx, sqlcpostgres.GetSessionByRefreshTokenParams{RefreshToken: refreshToken, ExpiresAt: parseTS(expiresAt)})
	if err != nil {
		return sessionRow{}, err
	}
	return sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: formatTS(s.ExpiresAt), CreatedAt: formatTS(s.CreatedAt), IPAddress: s.IpAddress, UserAgent: s.UserAgent}, nil
}

func (p *postgresQuerier) getSessionsByUserID(ctx context.Context, userID, expiresAt string, offset, limit int32) ([]sessionRow, error) {
	rows, err := p.q.GetSessionsByUserID(ctx, sqlcpostgres.GetSessionsByUserIDParams{UserID: userID, ExpiresAt: parseTS(expiresAt), Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]sessionRow, len(rows))
	for i, s := range rows {
		out[i] = sessionRow{ID: s.ID, UserID: s.UserID, Token: s.Token, RefreshToken: s.RefreshToken, ExpiresAt: formatTS(s.ExpiresAt), CreatedAt: formatTS(s.CreatedAt), IPAddress: s.IpAddress, UserAgent: s.UserAgent}
	}
	return out, nil
}

func (p *postgresQuerier) countSessionsByUserID(ctx context.Context, userID, expiresAt string) (int64, error) {
	return p.q.CountSessionsByUserID(ctx, sqlcpostgres.CountSessionsByUserIDParams{UserID: userID, ExpiresAt: parseTS(expiresAt)})
}

func (p *postgresQuerier) updateSession(ctx context.Context, id string, refreshToken sql.NullString, expiresAt string) error {
	return p.q.UpdateSession(ctx, sqlcpostgres.UpdateSessionParams{ID: id, RefreshToken: refreshToken, ExpiresAt: parseTS(expiresAt)})
}

func (p *postgresQuerier) deleteSession(ctx context.Context, id string) error {
	return p.q.DeleteSession(ctx, id)
}

func (p *postgresQuerier) deleteSessionsByUserID(ctx context.Context, userID string) error {
	return p.q.DeleteSessionsByUserID(ctx, userID)
}

func (p *postgresQuerier) cleanupExpiredSessions(ctx context.Context, now string) error {
	return p.q.CleanupExpiredSessions(ctx, parseTS(now))
}

func (p *postgresQuerier) withTx(tx *sql.Tx) querier {
	return &postgresQuerier{q: p.q.WithTx(tx)}
}
