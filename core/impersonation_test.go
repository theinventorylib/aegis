package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

func newImpersonationTestService(t *testing.T) (*SessionService, *mockSessionStore, *mockUserStore) {
	t.Helper()
	users := &mockUserStore{}
	sessions := &mockSessionStore{}
	return newSessionService(users, sessions, DefaultSessionConfig(), nil, nil), sessions, users
}

func seedUser(t *testing.T, users *mockUserStore, id string) auth.User {
	t.Helper()
	user := auth.User{ID: id, Email: id + "@example.com", Name: id}
	if _, err := users.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func TestImpersonationSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	service, store, users := newImpersonationTestService(t)
	admin := seedUser(t, users, "admin")
	target := seedUser(t, users, "target")

	session, err := service.CreateImpersonationSession(ctx, &admin, &target, "org-1", time.Minute)
	if err != nil {
		t.Fatalf("create impersonation: %v", err)
	}
	if session.RefreshToken != "" {
		t.Fatal("impersonation session must not receive a refresh token")
	}
	if session.Impersonation == nil || session.Impersonation.ActorUserID != admin.ID || session.Impersonation.OrganizationID != "org-1" {
		t.Fatalf("unexpected impersonation record: %+v", session.Impersonation)
	}
	if _, err := store.GetImpersonation(ctx, session.ID); err != nil {
		t.Fatalf("impersonation record not persisted: %v", err)
	}

	validated, user, err := service.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if user.ID != target.ID {
		t.Fatalf("validated user = %s, want %s", user.ID, target.ID)
	}
	if validated.Impersonation == nil || validated.Impersonation.ActorUserID != admin.ID {
		t.Fatal("validated session should carry the impersonation marker")
	}

	if _, err := service.RefreshSession(ctx, ""); err == nil {
		t.Fatal("impersonation session must not be refreshable")
	}
}

func TestImpersonationSessionExpires(t *testing.T) {
	ctx := context.Background()
	service, store, users := newImpersonationTestService(t)
	admin := seedUser(t, users, "admin")
	target := seedUser(t, users, "target")

	session, err := service.CreateImpersonationSession(ctx, &admin, &target, "org-1", time.Minute)
	if err != nil {
		t.Fatalf("create impersonation: %v", err)
	}
	imp, _ := store.GetImpersonation(ctx, session.ID)
	imp.ExpiresAt = time.Now().Add(-time.Second)
	if err := store.CreateImpersonation(ctx, imp); err != nil {
		t.Fatal(err)
	}

	_, _, err = service.ValidateSession(ctx, session.Token)
	if err == nil {
		t.Fatal("expired impersonation session must fail validation")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != AuthErrorCodeSessionInvalid {
		t.Fatalf("got %v, want %s", err, AuthErrorCodeSessionInvalid)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatal("expired impersonation session should be deleted")
	}
}

func TestImpersonationRevokedByActor(t *testing.T) {
	ctx := context.Background()
	service, store, users := newImpersonationTestService(t)
	admin := seedUser(t, users, "admin")
	target := seedUser(t, users, "target")

	session, err := service.CreateImpersonationSession(ctx, &admin, &target, "org-1", time.Minute)
	if err != nil {
		t.Fatalf("create impersonation: %v", err)
	}
	if err := service.RevokeImpersonationsByActor(ctx, admin.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatal("revoked impersonation session should be deleted")
	}
	if _, _, err := service.ValidateSession(ctx, session.Token); err == nil {
		t.Fatal("revoked impersonation session must fail validation")
	}
}

func TestImpersonationRefusesDisabledTargetAndSelf(t *testing.T) {
	ctx := context.Background()
	service, _, users := newImpersonationTestService(t)
	admin := seedUser(t, users, "admin")
	disabled := seedUser(t, users, "disabled")
	disabled.Disabled = true
	if err := users.Update(ctx, disabled); err != nil {
		t.Fatal(err)
	}

	if _, err := service.CreateImpersonationSession(ctx, &admin, &disabled, "org-1", time.Minute); err == nil {
		t.Fatal("disabled target must be rejected")
	}
	if _, err := service.CreateImpersonationSession(ctx, &admin, &admin, "org-1", time.Minute); err == nil {
		t.Fatal("self-impersonation must be rejected")
	}
}

func TestImpersonationCannotNest(t *testing.T) {
	ctx := context.Background()
	service, _, users := newImpersonationTestService(t)
	admin := seedUser(t, users, "admin")
	target := seedUser(t, users, "target")
	other := seedUser(t, users, "other")

	first, err := service.CreateImpersonationSession(ctx, &admin, &target, "org-1", time.Minute)
	if err != nil {
		t.Fatalf("create first impersonation: %v", err)
	}
	nestedCtx := WithSession(ctx, first)
	if _, err := service.CreateImpersonationSession(nestedCtx, &target, &other, "org-1", time.Minute); err == nil {
		t.Fatal("an impersonation session must not start another impersonation")
	}
}
