package core

import (
	"context"
	"errors"
	"testing"

	"github.com/theinventorylib/aegis/auth"
)

func newTestUserService() *UserService {
	return newUserService(&mockUserStore{}, &fakeAccountStore{}, &mockSessionStore{},
		defaultPasswordHasherConfig(), DefaultAuthConfig(), &NoOpAuditLogger{}, nil, nil)
}

func TestCreateUserEnforcesPasswordPolicy(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, auth.User{Name: "A", Email: "a@example.com"}, "weak"); err == nil {
		t.Fatal("expected weak password to be rejected by the default policy")
	}
	if _, err := svc.CreateUser(ctx, auth.User{Name: "A", Email: "a@example.com"}, "Str0ngPassword"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}

func TestCreateUserValidatesEmail(t *testing.T) {
	svc := newTestUserService()
	if _, err := svc.CreateUser(context.Background(), auth.User{Name: "A", Email: "not-an-email"}, "Str0ngPassword"); err == nil {
		t.Fatal("expected invalid email to be rejected")
	}
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, auth.User{Name: "A", Email: "dup@example.com"}, "Str0ngPassword"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreateUser(ctx, auth.User{Name: "B", Email: "dup@example.com"}, "Str0ngPassword")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("got %v, want ErrEmailAlreadyExists", err)
	}
}

func TestCreateUserWithUsername(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	u, err := svc.CreateUserWithUsername(ctx, "Alice", "alice@example.com", "Alice", "Str0ngPassword")
	if err != nil {
		t.Fatalf("create with username: %v", err)
	}

	// Username is normalized (lowercased) and stored on the credentials account.
	acc, err := svc.accountStore.GetByProvider(ctx, PasswordProvider, "alice")
	if err != nil {
		t.Fatalf("username lookup: %v", err)
	}
	if acc.UserID != u.GetID() {
		t.Errorf("account.UserID = %q, want %q", acc.UserID, u.GetID())
	}

	// A second user cannot take the same username.
	_, err = svc.CreateUserWithUsername(ctx, "Bob", "bob@example.com", "alice", "Str0ngPassword")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("got %v, want ErrUsernameTaken", err)
	}
}

func newTestAuthService() *AuthService {
	users := &mockUserStore{}
	accounts := &fakeAccountStore{}
	sessions := &mockSessionStore{}
	cfg := DefaultAuthConfig()

	as := &AuthService{
		hashConfig:   defaultPasswordHasherConfig(),
		auditLogger:  &NoOpAuditLogger{},
		authConfig:   cfg,
		userStore:    users,
		accountStore: accounts,
		sessionStore: sessions,
	}
	as.User = newUserService(users, accounts, sessions, as.hashConfig, cfg, as.auditLogger, nil, nil)
	as.Account = newAccountService(accounts, sessions, as.hashConfig, cfg, as.auditLogger, nil, nil)
	as.Session = newSessionService(users, sessions, nil, as.auditLogger, nil)
	as.Verification = newVerificationService(nil, as.auditLogger)
	as.EmailPassword = NewEmailPasswordHandlers(as)
	return as
}

func TestLoginByUsernameOrEmail(t *testing.T) {
	as := newTestAuthService()
	ctx := context.Background()

	if _, err := as.EmailPassword.RegisterWithUsername(ctx, "Alice", "alice@example.com", "alice", "Str0ngPassword"); err != nil {
		t.Fatalf("register: %v", err)
	}

	for _, identifier := range []string{"alice@example.com", "alice"} {
		res, err := as.EmailPassword.Login(ctx, identifier, "Str0ngPassword")
		if err != nil {
			t.Fatalf("login with %q: %v", identifier, err)
		}
		if res.User.Email != "alice@example.com" {
			t.Errorf("login with %q returned user %q", identifier, res.User.Email)
		}
	}

	if _, err := as.EmailPassword.Login(ctx, "alice", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: got %v, want ErrInvalidCredentials", err)
	}
}

func TestRequireEmailVerificationGatesLogin(t *testing.T) {
	as := newTestAuthService()
	as.authConfig.RequireEmailVerification = true
	ctx := context.Background()

	res, err := as.EmailPassword.RegisterWithUsername(ctx, "Alice", "alice@example.com", "", "Str0ngPassword")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.Session != nil || !res.VerificationRequired {
		t.Fatal("expected no session and VerificationRequired when verification is on")
	}

	// Fail closed: no checker wired yet.
	if _, err := as.EmailPassword.Login(ctx, "alice@example.com", "Str0ngPassword"); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("got %v, want ErrEmailNotVerified", err)
	}

	// With a checker reporting verified, login proceeds.
	as.emailVerificationCheck = func(context.Context, auth.User) (bool, error) { return true, nil }
	if _, err := as.EmailPassword.Login(ctx, "alice@example.com", "Str0ngPassword"); err != nil {
		t.Fatalf("verified login: %v", err)
	}
}

func TestUpdateUserEmailRejectsDuplicateAndResetsVerification(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, auth.User{Name: "A", Email: "a@example.com"}, "Str0ngPassword"); err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateUser(ctx, auth.User{Name: "B", Email: "b@example.com"}, "Str0ngPassword")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.UpdateUserEmail(ctx, b.GetID(), "a@example.com"); !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("duplicate: got %v, want ErrEmailAlreadyExists", err)
	}
	if err := svc.UpdateUserEmail(ctx, b.GetID(), "nope"); err == nil {
		t.Fatal("expected invalid email rejection")
	}

	var gotID, gotEmail string
	svc.setEmailVerificationReset(func(_ context.Context, userID, email string) error {
		gotID, gotEmail = userID, email
		return nil
	})
	if err := svc.UpdateUserEmail(ctx, b.GetID(), "b2@example.com"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if gotID != b.GetID() || gotEmail != "b2@example.com" {
		t.Errorf("resetter called with (%q, %q)", gotID, gotEmail)
	}
}

func TestDeleteUserPurgesSessionCacheAndDeletes(t *testing.T) {
	sessions := &mockSessionStore{}
	svc := newUserService(&mockUserStore{}, &fakeAccountStore{}, sessions,
		defaultPasswordHasherConfig(), DefaultAuthConfig(), &NoOpAuditLogger{}, nil, nil)
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, auth.User{Name: "A", Email: "a@example.com"}, "Str0ngPassword"); err != nil {
		t.Fatal(err)
	}
	u, err := svc.GetUserByEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create(ctx, auth.Session{ID: "s1", UserID: u.GetID(), Token: "t"}); err != nil {
		t.Fatal(err)
	}

	purged := false
	svc.setSessionCachePurger(func(_ context.Context, userID string) error {
		purged = true
		return nil
	})
	if err := svc.DeleteUser(ctx, u.GetID()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !purged {
		t.Error("session cache purger was not called")
	}
	rows, _ := sessions.GetByUserID(ctx, u.GetID(), 0, 10)
	if len(rows) != 0 {
		t.Errorf("sessions not deleted: %d remain", len(rows))
	}
	if _, err := svc.GetUserByID(ctx, u.GetID()); err == nil {
		t.Error("user still present after delete")
	}
}

func TestUnverifiedLoginDoesNotLockOut(t *testing.T) {
	as := newTestAuthService()
	as.authConfig.RequireEmailVerification = true
	as.emailVerificationCheck = func(context.Context, auth.User) (bool, error) { return false, nil }
	as.loginAttemptTracker = NewLoginAttemptTracker(DefaultLoginAttemptConfig(), nil)
	ctx := context.Background()

	if _, err := as.EmailPassword.RegisterWithUsername(ctx, "Alice", "alice@example.com", "", "Str0ngPassword"); err != nil {
		t.Fatalf("register: %v", err)
	}

	// More attempts than the default lockout threshold.
	for i := range 10 {
		if _, err := as.EmailPassword.Login(ctx, "alice@example.com", "Str0ngPassword"); !errors.Is(err, ErrEmailNotVerified) {
			t.Fatalf("attempt %d: got %v, want ErrEmailNotVerified", i, err)
		}
	}

	locked, _, err := as.loginAttemptTracker.IsLockedOut(ctx, "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("an unverified-email rejection must not feed the lockout counter")
	}
}
