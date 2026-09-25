package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// recordingAuditLogger captures emitted event types for assertions.
type recordingAuditLogger struct {
	events []AuditEventType
}

func (r *recordingAuditLogger) LogEvent(_ context.Context, e *AuditEvent) error {
	r.events = append(r.events, e.EventType)
	return nil
}

func (r *recordingAuditLogger) LogAuthEvent(_ context.Context, eventType AuditEventType, _ string, _ bool, _ map[string]any) error {
	r.events = append(r.events, eventType)
	return nil
}

func (r *recordingAuditLogger) has(eventType AuditEventType) bool {
	for _, e := range r.events {
		if e == eventType {
			return true
		}
	}
	return false
}

// newSecurityTestAuth builds an AuthService wired like production (security
// stores + verification) over in-memory stores.
func newSecurityTestAuth() (*AuthService, *mockUserStore, *mockSessionStore, *mockVerificationStore, *recordingAuditLogger) {
	users := &mockUserStore{}
	accounts := &fakeAccountStore{}
	sessions := &mockSessionStore{}
	verification, vstore := newTestVerificationService()
	audit := &recordingAuditLogger{}
	cfg := DefaultAuthConfig()

	as := &AuthService{
		hashConfig:        defaultPasswordHasherConfig(),
		auditLogger:       audit,
		authConfig:        cfg,
		userStore:         users,
		accountStore:      accounts,
		sessionStore:      sessions,
		verificationStore: vstore,
	}
	as.User = newUserService(users, accounts, sessions, as.hashConfig, cfg, audit, nil, nil)
	as.Account = newAccountService(accounts, sessions, as.hashConfig, cfg, audit, nil, nil)
	as.Session = newSessionService(users, sessions, nil, audit, nil)
	as.Verification = verification
	as.Account.setSecurityStores(users, verification)
	as.User.setVerificationService(verification)
	as.EmailPassword = NewEmailPasswordHandlers(as)
	return as, users, sessions, vstore, audit
}

func TestPasswordResetEndToEnd(t *testing.T) {
	ctx := context.Background()
	as, _, sessions, vstore, audit := newSecurityTestAuth()

	user, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "OldPassword123!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := sessions.Create(ctx, auth.Session{ID: "s1", UserID: user.GetID(), Token: "tok"}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	_, token, err := as.Account.RequestPasswordReset(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if token == "" {
		t.Fatal("expected a raw reset token")
	}
	// Only the hash is persisted.
	if _, ok := vstore.byTok[token]; ok {
		t.Fatal("raw reset token must not be stored at rest")
	}
	if _, ok := vstore.byTok[hashTokenHex(token)]; !ok {
		t.Fatal("stored reset token should be the hash")
	}

	if _, err := as.Account.ConfirmPasswordReset(ctx, token, "NewPassword123!"); err != nil {
		t.Fatalf("confirm reset: %v", err)
	}
	ok, err := as.Account.VerifyPassword(ctx, user.GetID(), "NewPassword123!")
	if err != nil || !ok {
		t.Fatalf("new password should verify (ok=%v err=%v)", ok, err)
	}
	// Sessions are revoked on password change.
	remaining, err := sessions.GetByUserID(ctx, user.GetID(), 0, sessionPageSize)
	if err != nil {
		t.Fatalf("get sessions: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("sessions not revoked after reset: %d remain", len(remaining))
	}
	if !audit.has(AuditEventPasswordReset) {
		t.Error("expected password_reset audit event")
	}
	if !audit.has(AuditEventPasswordChanged) {
		t.Error("expected password_changed audit event")
	}
	// Single use: replaying the token must fail.
	if _, err := as.Account.ConfirmPasswordReset(ctx, token, "AnotherPassword123!"); err == nil {
		t.Fatal("reset token must be single-use")
	}
}

func TestPasswordResetUnknownEmail(t *testing.T) {
	as, _, _, _, _ := newSecurityTestAuth()
	_, _, err := as.Account.RequestPasswordReset(context.Background(), "nobody@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}

func TestPasswordResetRejectsWrongTokenType(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "OldPassword123!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	v, err := as.Verification.CreateVerification(ctx, "alice@example.com", VerificationTypeEmailVerification, time.Hour, nil)
	if err != nil {
		t.Fatalf("create verification: %v", err)
	}
	if _, err := as.Account.ConfirmPasswordReset(ctx, v.Token, "NewPassword123!"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestPasswordResetWeakPasswordKeepsToken(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "OldPassword123!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, token, err := as.Account.RequestPasswordReset(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if _, err := as.Account.ConfirmPasswordReset(ctx, token, "weak"); err == nil {
		t.Fatal("expected weak password rejection")
	}
	// The token must still be redeemable after a policy rejection.
	if _, err := as.Account.ConfirmPasswordReset(ctx, token, "NewPassword123!"); err != nil {
		t.Fatalf("token should survive a rejected password: %v", err)
	}
}

func TestPasswordResetExpiredToken(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "OldPassword123!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	v, err := as.Verification.CreateVerification(ctx, "alice@example.com", VerificationTypePasswordReset, -time.Minute, nil)
	if err != nil {
		t.Fatalf("create verification: %v", err)
	}
	if _, err := as.Account.ConfirmPasswordReset(ctx, v.Token, "NewPassword123!"); err == nil {
		t.Fatal("expected expired token rejection")
	}
}

func TestPasswordResetInvalidatesPreviousTokens(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "OldPassword123!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, first, err := as.Account.RequestPasswordReset(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	_, second, err := as.Account.RequestPasswordReset(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	if _, err := as.Account.ConfirmPasswordReset(ctx, first, "NewPassword123!"); err == nil {
		t.Fatal("superseded token must not work")
	}
	if _, err := as.Account.ConfirmPasswordReset(ctx, second, "NewPassword123!"); err != nil {
		t.Fatalf("latest token should work: %v", err)
	}
}
