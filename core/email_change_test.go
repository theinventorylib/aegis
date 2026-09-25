package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

func TestEmailChangeEndToEnd(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, audit := newSecurityTestAuth()

	user, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "old@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	var resetFor, marked string
	as.User.setEmailVerificationReset(func(_ context.Context, _, email string) error {
		resetFor = email
		return nil
	})
	as.User.setEmailVerifiedMarker(func(_ context.Context, _, email string) error {
		marked = email
		return nil
	})

	token, err := as.User.RequestEmailChange(ctx, user.GetID(), "new@example.com")
	if err != nil {
		t.Fatalf("request change: %v", err)
	}
	if token == "" {
		t.Fatal("expected a raw change token")
	}

	updated, err := as.User.ConfirmEmailChange(ctx, user.GetID(), token)
	if err != nil {
		t.Fatalf("confirm change: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("email = %q, want new@example.com", updated.Email)
	}
	if resetFor != "new@example.com" {
		t.Errorf("verification resetter saw %q, want new@example.com", resetFor)
	}
	if marked != "new@example.com" {
		t.Errorf("verified marker saw %q, want new@example.com", marked)
	}
	if !audit.has(AuditEventEmailChanged) {
		t.Error("expected email_changed audit event")
	}
	// Single use: replaying the token must fail.
	if _, err := as.User.ConfirmEmailChange(ctx, user.GetID(), token); err == nil {
		t.Fatal("email-change token must be single-use")
	}
}

func TestEmailChangeRejectsTakenEmail(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	alice, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if _, err := as.User.CreateUser(ctx, auth.User{Name: "Bob", Email: "bob@example.com"}, "Str0ngPassword1!"); err != nil {
		t.Fatalf("create bob: %v", err)
	}
	_, err = as.User.RequestEmailChange(ctx, alice.GetID(), "bob@example.com")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("got %v, want ErrEmailAlreadyExists", err)
	}
}

func TestEmailChangeRejectsSameEmail(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	user, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := as.User.RequestEmailChange(ctx, user.GetID(), "alice@example.com"); err == nil {
		t.Fatal("expected same-email rejection")
	}
}

func TestEmailChangeRejectsWrongTokenType(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	user, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "alice@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	v, err := as.Verification.CreateVerification(ctx, "new@example.com", VerificationTypePasswordReset, time.Hour, nil)
	if err != nil {
		t.Fatalf("create verification: %v", err)
	}
	if _, err := as.User.ConfirmEmailChange(ctx, user.GetID(), v.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestEmailChangeMarkerFailureStillApplies(t *testing.T) {
	ctx := context.Background()
	as, _, _, _, _ := newSecurityTestAuth()
	user, err := as.User.CreateUser(ctx, auth.User{Name: "Alice", Email: "old@example.com"}, "Str0ngPassword1!")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	as.User.setEmailVerifiedMarker(func(_ context.Context, _, _ string) error {
		return errors.New("plugin unavailable")
	})
	token, err := as.User.RequestEmailChange(ctx, user.GetID(), "new@example.com")
	if err != nil {
		t.Fatalf("request change: %v", err)
	}
	if _, err := as.User.ConfirmEmailChange(ctx, user.GetID(), token); err != nil {
		t.Fatalf("marker failure must not fail the change: %v", err)
	}
}
