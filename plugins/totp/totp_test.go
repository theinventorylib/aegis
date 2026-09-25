package totp

import (
	"context"
	"encoding/base32"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/core"
	"github.com/theinventorylib/aegis/v2/plugins"
	totptypes "github.com/theinventorylib/aegis/v2/plugins/totp/types"
)

// fakeStore is an in-memory totptypes.Store.
type fakeStore struct {
	creds    map[string]totptypes.Credential
	sessions map[string]time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{creds: map[string]totptypes.Credential{}, sessions: map[string]time.Time{}}
}

func (f *fakeStore) GetCredential(_ context.Context, userID string) (totptypes.Credential, error) {
	if c, ok := f.creds[userID]; ok {
		return c, nil
	}
	return totptypes.Credential{UserID: userID}, nil
}

func (f *fakeStore) SetCredential(_ context.Context, userID string, secret *string, enabled bool) error {
	c := totptypes.Credential{UserID: userID, Enabled: enabled}
	if secret != nil {
		c.Secret = *secret
	}
	f.creds[userID] = c
	return nil
}

func (f *fakeStore) MarkSessionVerified(_ context.Context, sessionID, _ string, at time.Time) error {
	f.sessions[sessionID] = at
	return nil
}

func (f *fakeStore) IsSessionVerified(_ context.Context, sessionID, _ string, since time.Time) (bool, error) {
	at, ok := f.sessions[sessionID]
	return ok && at.After(since), nil
}

func (f *fakeStore) DeleteUserSessionVerifications(_ context.Context, _ string) error {
	f.sessions = map[string]time.Time{}
	return nil
}

func (f *fakeStore) DeleteSessionVerification(_ context.Context, sessionID string) error {
	delete(f.sessions, sessionID)
	return nil
}

// fakeVerifier is an in-memory verificationAPI.
type fakeVerifier struct {
	codes map[string]map[string]bool
}

func newFakeVerifier() *fakeVerifier {
	return &fakeVerifier{codes: map[string]map[string]bool{}}
}

func verifyKey(identifier, vType string) string { return identifier + "|" + vType }

func (f *fakeVerifier) CreateVerification(_ context.Context, identifier, vType string, _ time.Duration, customToken *string) (auth.Verification, error) {
	k := verifyKey(identifier, vType)
	if f.codes[k] == nil {
		f.codes[k] = map[string]bool{}
	}
	if customToken != nil {
		f.codes[k][*customToken] = true
	}
	return auth.Verification{Identifier: identifier, Type: vType}, nil
}

func (f *fakeVerifier) InvalidateVerification(_ context.Context, identifier, vType string) error {
	delete(f.codes, verifyKey(identifier, vType))
	return nil
}

func (f *fakeVerifier) ValidateVerificationFor(_ context.Context, identifier, vType, token string) (auth.Verification, error) {
	k := verifyKey(identifier, vType)
	if f.codes[k][token] {
		delete(f.codes[k], token)
		return auth.Verification{Identifier: identifier, Type: vType}, nil
	}
	return auth.Verification{}, core.ErrInvalidToken
}

func newTestPlugin() (*Plugin, *fakeStore, *fakeVerifier) {
	store := newFakeStore()
	verifier := newFakeVerifier()
	p := New(nil, store)
	p.verification = verifier
	return p, store, verifier
}

func currentCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := hotp(secret, uint64(time.Now().Unix()/30), 6)
	if err != nil {
		t.Fatalf("hotp: %v", err)
	}
	return code
}

// The plugin's migrations must not collide with another plugin's tables or
// columns (the framework fails registration on conflicts).
func TestMigrationsRegisterWithoutConflicts(t *testing.T) {
	registry := plugins.NewMigrationRegistry()
	p := New(nil, nil)
	if err := registry.Register(p.Name(), p.GetMigrations()); err != nil {
		t.Fatalf("register migrations: %v", err)
	}
}

// RFC 6238 Appendix B vectors (HMAC-SHA1, 8 digits).
func TestHOTPMatchesRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	vectors := []struct {
		unix int64
		want string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, v := range vectors {
		got, err := hotp(secret, uint64(v.unix/30), 8)
		if err != nil {
			t.Fatalf("hotp(%d): %v", v.unix, err)
		}
		if got != v.want {
			t.Errorf("hotp at %d = %s, want %s", v.unix, got, v.want)
		}
	}
}

func TestSetupEnableVerifyDisableFlow(t *testing.T) {
	ctx := context.Background()
	p, store, verifier := newTestPlugin()

	setup, err := p.Setup(ctx, "u1")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if setup.Secret == "" || !strings.HasPrefix(setup.URL, "otpauth://totp/") {
		t.Fatalf("unexpected setup result: %+v", setup)
	}
	if store.creds["u1"].Enabled {
		t.Fatal("setup must leave TOTP disabled until a code is confirmed")
	}

	if _, err := p.Enable(ctx, "u1", "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("wrong code: got %v, want ErrInvalidCode", err)
	}
	if store.creds["u1"].Enabled {
		t.Fatal("failed enable must not activate TOTP")
	}

	codes, err := p.Enable(ctx, "u1", currentCode(t, setup.Secret))
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("recovery codes = %d, want 10", len(codes))
	}
	if !store.creds["u1"].Enabled {
		t.Fatal("TOTP should be enabled")
	}

	ok, err := p.Verify(ctx, "u1", currentCode(t, setup.Secret))
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}
	ok, err = p.Verify(ctx, "u1", "000000")
	if err != nil || ok {
		t.Fatalf("wrong verify: ok=%v err=%v", ok, err)
	}

	// Recovery codes are single-use.
	ok, err = p.VerifyRecoveryCode(ctx, "u1", codes[0])
	if err != nil || !ok {
		t.Fatalf("recovery verify: ok=%v err=%v", ok, err)
	}
	ok, _ = p.VerifyRecoveryCode(ctx, "u1", codes[0])
	if ok {
		t.Fatal("recovery code must be single-use")
	}
	if len(verifier.codes[verifyKey("u1", verificationTypeRecovery)]) != 9 {
		t.Fatal("consumed recovery code should be removed from the store")
	}

	if err := p.Disable(ctx, "u1"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if store.creds["u1"].Enabled || store.creds["u1"].Secret != "" {
		t.Fatal("disable must clear the credential")
	}
	if len(verifier.codes[verifyKey("u1", verificationTypeRecovery)]) != 0 {
		t.Fatal("disable must invalidate recovery codes")
	}
}

func TestSetupRefusesWhileEnabled(t *testing.T) {
	ctx := context.Background()
	p, _, _ := newTestPlugin()
	setup, err := p.Setup(ctx, "u1")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := p.Enable(ctx, "u1", currentCode(t, setup.Secret)); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if _, err := p.Setup(ctx, "u1"); !errors.Is(err, ErrAlreadyEnabled) {
		t.Fatalf("setup while enabled: got %v, want ErrAlreadyEnabled", err)
	}
}

func TestEnableRequiresSetup(t *testing.T) {
	p, _, _ := newTestPlugin()
	if _, err := p.Enable(context.Background(), "u1", "123456"); !errors.Is(err, ErrNotSetup) {
		t.Fatalf("got %v, want ErrNotSetup", err)
	}
}

func TestRegenerateRecoveryCodesRequiresValidCode(t *testing.T) {
	ctx := context.Background()
	p, _, verifier := newTestPlugin()
	setup, err := p.Setup(ctx, "u1")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := p.Enable(ctx, "u1", currentCode(t, setup.Secret)); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if _, err := p.RegenerateRecoveryCodes(ctx, "u1", "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("got %v, want ErrInvalidCode", err)
	}
	codes, err := p.RegenerateRecoveryCodes(ctx, "u1", currentCode(t, setup.Secret))
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("regenerated codes = %d, want 10", len(codes))
	}
	if len(verifier.codes[verifyKey("u1", verificationTypeRecovery)]) != 10 {
		t.Fatal("regeneration should replace the previous set")
	}
}

func TestRequireVerificationMiddleware(t *testing.T) {
	ctx := context.Background()
	p, store, _ := newTestPlugin()
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if err := store.SetCredential(ctx, "u1", &secret, true); err != nil {
		t.Fatalf("seed credential: %v", err)
	}
	handler := p.RequireVerification(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	authed := core.WithUser(core.WithSession(context.Background(), &auth.Session{ID: "s1"}), &auth.User{ID: "u1"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/anything", nil).WithContext(authed))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "totp_required") {
		t.Fatalf("unverified session: status=%d body=%s", rec.Code, rec.Body.String())
	}

	if err := store.MarkSessionVerified(ctx, "s1", "u1", time.Now()); err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/anything", nil).WithContext(authed))
	if rec.Code != http.StatusOK {
		t.Fatalf("verified session: status=%d", rec.Code)
	}

	// Unauthenticated requests pass through so this composes with auth middleware.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/anything", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("anonymous request: status=%d", rec.Code)
	}
}
