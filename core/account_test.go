package core

import (
	"context"
	"database/sql"
	"testing"

	"github.com/theinventorylib/aegis/v2/auth"
)

// fakeAccountStore is a minimal AccountStore for password-update tests.
type fakeAccountStore struct {
	accounts []auth.Account
}

func (f *fakeAccountStore) Create(_ context.Context, account auth.Account) error {
	f.accounts = append(f.accounts, account)
	return nil
}

func (f *fakeAccountStore) GetByID(_ context.Context, id string) (auth.Account, error) {
	for _, a := range f.accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return auth.Account{}, sql.ErrNoRows
}

func (f *fakeAccountStore) GetByUserID(_ context.Context, userID string) ([]auth.Account, error) {
	var out []auth.Account
	for _, a := range f.accounts {
		if a.UserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAccountStore) GetByProvider(_ context.Context, provider, providerAccountID string) (auth.Account, error) {
	for _, a := range f.accounts {
		if a.Provider == provider && a.ProviderAccountID == providerAccountID {
			return a, nil
		}
	}
	return auth.Account{}, sql.ErrNoRows
}

func (f *fakeAccountStore) Update(_ context.Context, account auth.Account) error {
	for i := range f.accounts {
		if f.accounts[i].ID == account.ID {
			f.accounts[i] = account
			return nil
		}
	}
	return sql.ErrNoRows
}

func (f *fakeAccountStore) Delete(_ context.Context, id string) error {
	for i := range f.accounts {
		if f.accounts[i].ID == id {
			f.accounts = append(f.accounts[:i], f.accounts[i+1:]...)
			return nil
		}
	}
	return nil
}

// TestUpdatePasswordPurgesCacheBeforeDelete is the regression guard for the
// ordering bug: the session cache purge discovers per-session cache keys by
// walking the session store, so it must run while the rows still exist.
// Previously the purge ran after DeleteByUserID and therefore found nothing.
func TestUpdatePasswordPurgesCacheBeforeDelete(t *testing.T) {
	ctx := context.Background()
	sessions := &mockSessionStore{}
	if err := sessions.Create(ctx, auth.Session{ID: "s1", UserID: "u1", Token: "tok"}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	accounts := &fakeAccountStore{accounts: []auth.Account{
		{ID: "a1", UserID: "u1", Provider: PasswordProvider, PasswordHash: "old-hash"},
	}}

	svc := newAccountService(accounts, sessions, defaultPasswordHasherConfig(), DefaultAuthConfig(), &NoOpAuditLogger{}, nil, nil)

	var sessionsSeenByPurge int
	svc.setSessionInvalidator(func(ctx context.Context, userID string) error {
		rows, err := sessions.GetByUserID(ctx, userID, 0, sessionPageSize)
		if err != nil {
			return err
		}
		sessionsSeenByPurge = len(rows)
		return nil
	})

	if err := svc.UpdatePassword(ctx, "u1", "NewPassword123!"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	if sessionsSeenByPurge != 1 {
		t.Errorf("cache purge saw %d sessions, want 1 (must run before the DB delete)", sessionsSeenByPurge)
	}

	rows, err := sessions.GetByUserID(ctx, "u1", 0, sessionPageSize)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("sessions not deleted after password change: %d remain", len(rows))
	}
}

// recordingAuditLogger lives in password_reset_test.go: it implements both
// LogEvent and LogAuthEvent and captures the event types.

// capableAccountStore is a fakeAccountStore that also implements the optional
// PasswordHashUpdater capability, recording which write path is used.
type capableAccountStore struct {
	fakeAccountStore
	narrowWrites []passwordWrite
	fullUpdates  int
}

type passwordWrite struct{ accountID, hash string }

func (c *capableAccountStore) Update(ctx context.Context, account auth.Account) error {
	c.fullUpdates++
	return c.fakeAccountStore.Update(ctx, account)
}

func (c *capableAccountStore) UpdatePasswordHash(_ context.Context, accountID, passwordHash string) error {
	c.narrowWrites = append(c.narrowWrites, passwordWrite{accountID: accountID, hash: passwordHash})
	return nil
}

// TestUpdatePasswordPrefersNarrowHashWrite pins the capability preference:
// with a store that supports PasswordHashUpdater, password rotation must not
// go through the full-row Update, which would rewrite token columns with the
// values captured at load time.
func TestUpdatePasswordPrefersNarrowHashWrite(t *testing.T) {
	ctx := context.Background()
	accounts := &capableAccountStore{fakeAccountStore: fakeAccountStore{accounts: []auth.Account{
		{ID: "a1", UserID: "u1", Provider: PasswordProvider, PasswordHash: "old-hash", AccessToken: "token"},
	}}}
	svc := newAccountService(accounts, &mockSessionStore{}, defaultPasswordHasherConfig(), DefaultAuthConfig(), &NoOpAuditLogger{}, nil, nil)

	if err := svc.UpdatePassword(ctx, "u1", "NewPassword123!"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	if accounts.fullUpdates != 0 {
		t.Errorf("full-row Update called %d times, want 0 (capability must be preferred)", accounts.fullUpdates)
	}
	if len(accounts.narrowWrites) != 1 {
		t.Fatalf("narrow writes = %v, want exactly one", accounts.narrowWrites)
	}
	if w := accounts.narrowWrites[0]; w.accountID != "a1" || w.hash == "" || w.hash == "old-hash" {
		t.Errorf("narrow write = %+v, want a1 with a rotated hash", w)
	}
}

// TestPasswordAuditEventTypes pins the difference between the change and
// reset flows: ResetPassword records password_reset, UpdatePassword records
// password_changed.
func TestPasswordAuditEventTypes(t *testing.T) {
	ctx := context.Background()
	audit := &recordingAuditLogger{}
	accounts := &fakeAccountStore{accounts: []auth.Account{
		{ID: "a1", UserID: "u1", Provider: PasswordProvider, PasswordHash: "old-hash"},
	}}
	svc := newAccountService(accounts, &mockSessionStore{}, defaultPasswordHasherConfig(), DefaultAuthConfig(), audit, nil, nil)

	if err := svc.ResetPassword(ctx, "u1", "NewPassword123!"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if err := svc.UpdatePassword(ctx, "u1", "AnotherPassword123!"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	if len(audit.events) != 2 || audit.events[0] != AuditEventPasswordReset || audit.events[1] != AuditEventPasswordChanged {
		t.Fatalf("audit events = %v, want [%s %s]", audit.events, AuditEventPasswordReset, AuditEventPasswordChanged)
	}
}
