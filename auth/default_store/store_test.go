package defaultstore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	types "github.com/theinventorylib/aegis/v2/auth/types"
)

// stubQuerier records the account update. The embedded querier keeps the
// interface's other ~25 methods out of this test; calling one would panic,
// which is fine because the test only exercises Update.
type stubQuerier struct {
	querier
	gotPasswordHash       sql.NullString
	gotNarrowPasswordHash sql.NullString
}

func (s *stubQuerier) updateAccount(_ context.Context, _ string, passwordHash, _, _, _ sql.NullString, _ string) error {
	s.gotPasswordHash = passwordHash
	return nil
}

func (s *stubQuerier) updateAccountPassword(_ context.Context, _ string, passwordHash sql.NullString, _ string) error {
	s.gotNarrowPasswordHash = passwordHash
	return nil
}

// TestAccountStoreUpdateWritesPasswordHash is the regression guard for the
// password-rotation bug: UpdateAccount used to write only the token columns,
// so a password reset or change reported success while the old hash stayed in
// the database and the new password could never sign in.
func TestAccountStoreUpdateWritesPasswordHash(t *testing.T) {
	stub := &stubQuerier{}
	store := &defaultAccountStore{q: stub}

	err := store.Update(context.Background(), types.Account{
		ID:           "acc-1",
		UserID:       "user-1",
		Provider:     "password",
		PasswordHash: "new-hash",
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !stub.gotPasswordHash.Valid || stub.gotPasswordHash.String != "new-hash" {
		t.Fatalf("password hash passed to the store = %+v, want new-hash", stub.gotPasswordHash)
	}
}

// TestAccountStoreUpdatePasswordHash is the regression guard for the narrow
// hash-only write used by password rotation: it must reach the dedicated
// query with the new hash so the update cannot touch token columns.
func TestAccountStoreUpdatePasswordHash(t *testing.T) {
	stub := &stubQuerier{}
	store := &defaultAccountStore{q: stub}

	if err := store.UpdatePasswordHash(context.Background(), "acc-1", "new-hash"); err != nil {
		t.Fatalf("UpdatePasswordHash: %v", err)
	}
	if !stub.gotNarrowPasswordHash.Valid || stub.gotNarrowPasswordHash.String != "new-hash" {
		t.Fatalf("hash passed to UpdateAccountPassword = %+v, want new-hash", stub.gotNarrowPasswordHash)
	}
}
