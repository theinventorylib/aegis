package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// TestLoginRejectsDisabledUser covers the #96 regression: a disabled account
// must not receive a session even when the password is correct.
func TestLoginRejectsDisabledUser(t *testing.T) {
	ctx := context.Background()
	as, _, _, audit := newSecurityTestAuth()

	user, err := as.User.CreateUser(ctx, auth.User{Name: "Disabled", Email: "disabled@example.com"}, "Password123!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	user.Disabled = true
	if err := as.userStore.Update(ctx, user); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	_, err = as.EmailPassword.Login(ctx, "disabled@example.com", "Password123!")
	if err == nil {
		t.Fatal("disabled user login must fail")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != AuthErrorCodeUserDisabled {
		t.Fatalf("got %v, want %s", err, AuthErrorCodeUserDisabled)
	}
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatal("disabled error should wrap ErrUserDisabled")
	}
	if !audit.has(AuditEventLoginFailed) {
		t.Fatal("expected login_failed audit event")
	}
}

// TestCreateSessionRejectsDisabledUser ensures every authentication method
// that funnels through CreateSession (OAuth, SMS, OTP) is protected, not only
// the password login path.
func TestCreateSessionRejectsDisabledUser(t *testing.T) {
	ctx := context.Background()
	as, _, _, _ := newSecurityTestAuth()

	user := &auth.User{ID: "u-disabled", Email: "disabled@example.com", Disabled: true}
	_, err := as.Session.CreateSession(ctx, user)
	if err == nil {
		t.Fatal("CreateSession must reject a disabled user")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != AuthErrorCodeUserDisabled {
		t.Fatalf("got %v, want %s", err, AuthErrorCodeUserDisabled)
	}
}

// TestValidateSessionRejectsDisabledUser locks in the session-validation gate.
func TestValidateSessionRejectsDisabledUser(t *testing.T) {
	ctx := context.Background()
	as, sessions, _, _ := newSecurityTestAuth()

	user := auth.User{ID: "u-disabled", Email: "disabled@example.com", Disabled: true}
	if _, err := as.userStore.Create(ctx, user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	rawToken := "raw-session-token"
	if err := sessions.Create(ctx, auth.Session{
		ID:        "s-disabled",
		UserID:    user.ID,
		Token:     hashTokenHex(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	_, _, err := as.Session.ValidateSession(ctx, rawToken)
	if err == nil {
		t.Fatal("session validation must reject a disabled user")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != AuthErrorCodeUserDisabled {
		t.Fatalf("got %v, want %s", err, AuthErrorCodeUserDisabled)
	}
}

// TestRefreshSessionRejectsDisabledUser checks that a refresh token issued
// before disabling cannot mint a fresh session.
func TestRefreshSessionRejectsDisabledUser(t *testing.T) {
	ctx := context.Background()
	as, sessions, _, _ := newSecurityTestAuth()

	user := auth.User{ID: "u-disabled", Email: "disabled@example.com", Disabled: true}
	if _, err := as.userStore.Create(ctx, user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	refresh := "raw-refresh-token"
	if err := sessions.Create(ctx, auth.Session{
		ID:           "s-disabled",
		UserID:       user.ID,
		Token:        hashTokenHex("raw-session-token"),
		RefreshToken: hashTokenHex(refresh),
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	_, err := as.Session.RefreshSession(ctx, refresh)
	if err == nil {
		t.Fatal("refresh for a disabled user must fail")
	}
	if _, getErr := sessions.GetByRefreshToken(ctx, hashTokenHex(refresh)); getErr == nil {
		t.Fatal("refresh failure should delete the old session")
	}
}

// TestLockoutErrorIncludesRetryAfter covers the discarded `remaining` value.
func TestLockoutErrorIncludesRetryAfter(t *testing.T) {
	ctx := context.Background()
	as, _, _, _ := newSecurityTestAuth()

	tracker := NewLoginAttemptTracker(&LoginAttemptConfig{
		MaxAttempts:     1,
		LockoutDuration: time.Minute,
		AttemptWindow:   time.Minute,
	}, nil)
	defer tracker.Stop()
	as.loginAttemptTracker = tracker

	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Locked", Email: "locked@example.com"}, "Password123!"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, _, err := tracker.RecordFailedAttempt(ctx, "locked@example.com"); err != nil {
		t.Fatalf("record failure: %v", err)
	}

	_, err := as.EmailPassword.Login(ctx, "locked@example.com", "Password123!")
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != AuthErrorCodeRateLimit {
		t.Fatalf("got %v, want %s", err, AuthErrorCodeRateLimit)
	}
	if authErr.RetryAfter <= 0 {
		t.Fatal("lockout error must carry RetryAfter")
	}
}
