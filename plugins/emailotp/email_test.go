package emailotp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theinventorylib/aegis/v2/core"
	"github.com/theinventorylib/aegis/v2/plugins"
	emailotptypes "github.com/theinventorylib/aegis/v2/plugins/emailotp/types"
)

// fakeStore is a minimal emailotptypes.Store for handler tests.
type fakeStore struct {
	users    map[string]*emailotptypes.User
	verified map[string]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]*emailotptypes.User{}, verified: map[string]bool{}}
}

func (f *fakeStore) CreateUser(_ context.Context, user emailotptypes.User) (*emailotptypes.User, error) {
	f.users[user.ID] = &user
	return &user, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (*emailotptypes.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, core.ErrUserNotFound
}

func (f *fakeStore) UpdateUserEmail(_ context.Context, userID, email string, verified bool) error {
	if u, ok := f.users[userID]; ok {
		u.Email = email
		f.verified[userID] = verified
	}
	return nil
}

// fakeProvider records sends and answers VerifyOTP with a fixed verdict.
type fakeProvider struct {
	sent   []string
	verify bool
}

func (p *fakeProvider) SendOTP(to, code string) error {
	p.sent = append(p.sent, to+":"+code)
	return nil
}

func (p *fakeProvider) SendEmail(to, subject, body string) error {
	p.sent = append(p.sent, to+":"+subject+":"+body)
	return nil
}

func (p *fakeProvider) VerifyOTP(_, _ string) (bool, error) {
	return p.verify, nil
}

func newTestHandlers() (*Handlers, *fakeStore, *fakeProvider) {
	store := newFakeStore()
	store.users["u1"] = &emailotptypes.User{Email: "user@example.com"}
	store.users["u1"].ID = "u1"
	provider := &fakeProvider{verify: true}
	p := New(&Config{Provider: provider}, store, plugins.DialectPostgres)
	return NewHandlers(p), store, provider
}

func postVerify(h *Handlers, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/email-otp/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.VerifyOTPHandler(rec, req)
	return rec
}

// A password-reset (or MFA) code must not flip the email-verified flag; only
// an email_verification code proves ownership of the address.
func TestVerifyOTPOnlyMarksEmailVerifiedForVerificationPurpose(t *testing.T) {
	h, store, _ := newTestHandlers()

	rec := postVerify(h, `{"email":"user@example.com","code":"123456","purpose":"password_reset"}`)
	if rec.Code != 200 {
		t.Fatalf("password_reset verify status = %d, body %s", rec.Code, rec.Body.String())
	}
	if store.verified["u1"] {
		t.Fatal("password_reset code must not mark the email verified")
	}

	rec = postVerify(h, `{"email":"user@example.com","code":"123456","purpose":"email_verification"}`)
	if rec.Code != 200 {
		t.Fatalf("email_verification verify status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !store.verified["u1"] {
		t.Fatal("email_verification code must mark the email verified")
	}
}

func TestPasswordResetMailUsesURLTemplate(t *testing.T) {
	p := New(&Config{PasswordResetURL: "https://app.example.com/reset?token={token}"}, nil, plugins.DialectPostgres)
	body := p.passwordResetMail("abc123")
	if !strings.Contains(body, "https://app.example.com/reset?token=abc123") {
		t.Fatalf("body does not contain rendered link: %q", body)
	}
}

func TestEmailChangeMailFallsBackToCode(t *testing.T) {
	p := New(&Config{}, nil, plugins.DialectPostgres)
	body := p.emailChangeMail("abc123")
	if !strings.Contains(body, "abc123") {
		t.Fatalf("body does not contain the code: %q", body)
	}
}
