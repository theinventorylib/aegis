package core

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddlewareIssuesCookieOnSafeRequests(t *testing.T) {
	mw := CSRFMiddleware(testCSRFConfig())
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/form", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	cookie := findCookie(rec.Result().Cookies(), "aegis_csrf")
	if cookie == nil {
		t.Fatal("expected CSRF cookie to be issued")
	}
	if cookie.HttpOnly {
		t.Fatal("expected CSRF cookie to be readable by client code")
	}
	if !verifyCSRFToken(DeriveSecret(testCSRFConfig().MasterSecret, "csrf-token", DefaultSecretLength), cookie.Value) {
		t.Fatal("expected issued CSRF cookie to contain a valid signed token")
	}
}

func TestCSRFMiddlewareAcceptsValidUnsafeRequest(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRFMiddleware(cfg)

	getHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getRec := httptest.NewRecorder()
	getHandler.ServeHTTP(getRec, getReq)

	cookie := findCookie(getRec.Result().Cookies(), cfg.CookieName)
	if cookie == nil {
		t.Fatal("expected CSRF cookie to be issued")
	}

	called := false
	postHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	postReq := httptest.NewRequest(http.MethodPost, "/form", nil)
	postReq.AddCookie(cookie)
	postReq.Header.Set(cfg.HeaderName, cookie.Value)
	postReq.TLS = &tls.ConnectionState{}
	postRec := httptest.NewRecorder()
	postHandler.ServeHTTP(postRec, postReq)

	if !called {
		t.Fatal("expected next handler to be called for valid CSRF token")
	}
	if postRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, postRec.Code)
	}
}

func TestCSRFMiddlewareSkipsBearerRequests(t *testing.T) {
	mw := CSRFMiddleware(testCSRFConfig())
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected bearer-auth request to bypass CSRF enforcement")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected bearer-auth request to skip CSRF cookie issuance")
	}
}

func TestCSRFMiddlewareStopsWhenCookieIssuanceFails(t *testing.T) {
	originalGenerator := csrfTokenGenerator
	csrfTokenGenerator = func([]byte) (string, error) {
		return "", errors.New("boom")
	}
	defer func() { csrfTokenGenerator = originalGenerator }()

	mw := CSRFMiddleware(testCSRFConfig())
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/form", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("expected next handler to be skipped after CSRF token issuance failure")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func testCSRFConfig() CSRFConfig {
	cfg := DefaultCSRFConfig()
	cfg.MasterSecret = []byte("0123456789abcdef0123456789abcdef")
	cfg.Secure = false
	return cfg
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
