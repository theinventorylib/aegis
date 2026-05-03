package core

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddlewareDefaults(t *testing.T) {
	handler := SecurityHeadersMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := rec.Result().Header
	if got := headers.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options=nosniff, got %q", got)
	}
	if got := headers.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options=DENY, got %q", got)
	}
	if got := headers.Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("expected Referrer-Policy default, got %q", got)
	}
	if got := headers.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS header on non-TLS request, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareSetsHSTSOnlyOverTLS(t *testing.T) {
	handler := SecurityHeadersMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Result().Header.Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("expected HSTS header on TLS request, got %q", got)
	}
}
