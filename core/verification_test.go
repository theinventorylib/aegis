package core

import (
	"context"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// mockVerificationStore is a minimal in-memory VerificationStore.
type mockVerificationStore struct {
	byID  map[string]auth.Verification
	byTok map[string]auth.Verification // key: token as stored (i.e. the hash)
}

func newMockVerificationStore() *mockVerificationStore {
	return &mockVerificationStore{
		byID:  map[string]auth.Verification{},
		byTok: map[string]auth.Verification{},
	}
}

func (m *mockVerificationStore) Create(_ context.Context, v auth.Verification) error {
	m.byID[v.ID] = v
	m.byTok[v.Token] = v
	return nil
}

func (m *mockVerificationStore) GetByToken(_ context.Context, token string) (auth.Verification, error) {
	if v, ok := m.byTok[token]; ok {
		return v, nil
	}
	return auth.Verification{}, errStopPage // any not-found error works
}

func (m *mockVerificationStore) GetByIdentifier(_ context.Context, identifier string) ([]auth.Verification, error) {
	var out []auth.Verification
	for _, v := range m.byID {
		if v.Identifier == identifier {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *mockVerificationStore) InvalidateByIdentifier(_ context.Context, identifier, vType string) error {
	for k, v := range m.byTok {
		if v.Identifier == identifier && v.Type == vType {
			delete(m.byTok, k)
		}
	}
	return nil
}

func (m *mockVerificationStore) Delete(_ context.Context, id string) error {
	v, ok := m.byID[id]
	if !ok {
		return nil
	}
	delete(m.byID, id)
	delete(m.byTok, v.Token)
	return nil
}

func (m *mockVerificationStore) CleanupExpired(_ context.Context) error { return nil }

func newTestVerificationService() (*VerificationService, *mockVerificationStore) {
	store := newMockVerificationStore()
	return newVerificationService(store, &NoOpAuditLogger{}), store
}

func TestCreateVerificationStoresHashOnly(t *testing.T) {
	svc, store := newTestVerificationService()
	v, err := svc.CreateVerification(context.Background(), "user@example.com", "email", time.Hour, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.Token == "" || v.Token == hashTokenHex(v.Token) {
		t.Fatalf("returned token should be raw: %q", v.Token)
	}
	if _, ok := store.byTok[v.Token]; ok {
		t.Fatal("raw token must not be stored at rest")
	}
	if _, ok := store.byTok[hashTokenHex(v.Token)]; !ok {
		t.Fatal("stored token should be the hash")
	}
	// Validation with the raw token still works.
	got, err := svc.ValidateVerification(context.Background(), v.Token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.ID != v.ID {
		t.Fatal("wrong record returned")
	}
}

func TestValidateVerificationForScopesAndConsumes(t *testing.T) {
	svc, _ := newTestVerificationService()
	ctx := context.Background()
	v, err := svc.CreateVerification(ctx, "user@example.com", "email", time.Hour, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Wrong identifier must not verify.
	if _, err := svc.ValidateVerificationFor(ctx, "attacker@example.com", "email", v.Token); err == nil {
		t.Fatal("expected mismatch failure for wrong identifier")
	}
	// Wrong purpose must not verify.
	if _, err := svc.ValidateVerificationFor(ctx, "user@example.com", "reset", v.Token); err == nil {
		t.Fatal("expected mismatch failure for wrong type")
	}
	// Correct identifier+type verifies once...
	if _, err := svc.ValidateVerificationFor(ctx, "user@example.com", "email", v.Token); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	// ...and is consumed: a replay must fail.
	if _, err := svc.ValidateVerificationFor(ctx, "user@example.com", "email", v.Token); err == nil {
		t.Fatal("token must be single-use")
	}
}

func TestValidateVerificationExpired(t *testing.T) {
	svc, store := newTestVerificationService()
	ctx := context.Background()
	v, err := svc.CreateVerification(ctx, "user@example.com", "email", -time.Minute, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.ValidateVerification(ctx, v.Token); err == nil {
		t.Fatal("expected expired error")
	}
	_ = store
}

func TestCreateVerificationCustomToken(t *testing.T) {
	svc, _ := newTestVerificationService()
	ctx := context.Background()
	code := "123456"
	v, err := svc.CreateVerification(ctx, "+15551234567", "otp", 10*time.Minute, &code)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.Token != code {
		t.Fatalf("custom token not used: %q", v.Token)
	}
	if _, err := svc.ValidateVerificationFor(ctx, "+15551234567", "otp", code); err != nil {
		t.Fatalf("scoped validate: %v", err)
	}
}
