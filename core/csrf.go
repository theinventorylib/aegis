package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"
)

// CSRF protection in Aegis follows the **signed double-submit cookie**
// pattern (OWASP Cheat Sheet recommendation, 2024):
//
//  1. On first non-Bearer request, the middleware sets a CSRF cookie
//     containing a freshly-generated random nonce plus an HMAC of that
//     nonce keyed by a secret derived from the master secret via
//     DeriveSecret(master, "csrf-token", 32). The cookie is scoped with
//     SameSite=Lax (or whatever the caller configures) and marked
//     Secure / HttpOnly=false so client JS can read it and echo it
//     back in a header.
//
//  2. On any state-changing request (POST/PUT/PATCH/DELETE) the middleware
//     reads the same cookie and compares it against a value the client
//     supplies via the configured header (default "X-CSRF-Token") using
//     constant-time comparison.
//
//  3. Bearer-token requests (Authorization: Bearer …) and safe methods
//     (GET/HEAD/OPTIONS/TRACE) are exempted, matching how the rest of
//     Aegis treats stateless API calls.
//
// The HMAC binds the nonce to the server's secret, so an attacker that
// can forge a cookie value cannot forge a valid signed token without
// also knowing the master secret. Verification is constant-time
// (hmac.Equal) and the per-request comparison between the cookie and
// the header is also constant-time.

// CSRFConfig configures the CSRF protection middleware.
type CSRFConfig struct {
	// MasterSecret is the application's master secret. The middleware
	// derives a CSRF-specific HMAC key from it via DeriveSecret. Must
	// be at least 32 bytes; shorter secrets cause CSRFMiddleware to
	// panic at construction time, which fails fast at startup.
	MasterSecret []byte

	// CookieName is the cookie that carries the signed token.
	// Default: "aegis_csrf".
	CookieName string

	// HeaderName is the request header the middleware reads on unsafe
	// methods. Default: "X-CSRF-Token".
	HeaderName string

	// CookiePath restricts the CSRF cookie to a path. Default: "/".
	CookiePath string

	// CookieDomain optionally scopes the cookie to a domain.
	CookieDomain string

	// SameSite controls the cookie's SameSite attribute. Default:
	// http.SameSiteLaxMode, which is the recommended balance for
	// browser apps that perform top-level navigation.
	SameSite http.SameSite

	// Secure marks the cookie Secure (HTTPS-only). Default: true.
	// Set to false ONLY for local development.
	Secure bool

	// MaxAge is the cookie lifetime. Default: 12h.
	MaxAge time.Duration

	// TrustedOrigins, when non-empty, additionally requires the request
	// Origin (or Referer, fallback) to match one of these absolute
	// origins (e.g. "https://app.example.com"). This is defense in
	// depth on top of the double-submit check.
	TrustedOrigins []string

	// SkipPaths is an optional list of exact paths to bypass entirely.
	// Use sparingly; the standard exemptions (safe methods + Bearer)
	// already cover most legitimate cases.
	SkipPaths []string
}

// DefaultCSRFConfig returns a CSRFConfig populated with safe defaults.
// MasterSecret must still be set by the caller.
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		CookieName: "aegis_csrf",
		HeaderName: "X-CSRF-Token",
		CookiePath: "/",
		SameSite:   http.SameSiteLaxMode,
		Secure:     true,
		MaxAge:     12 * time.Hour,
	}
}

// csrfNonceLen is the length of the random nonce that backs each token.
// 32 bytes (256 bits) provides ample collision resistance.
const csrfNonceLen = 32

var csrfTokenGenerator = generateCSRFToken

// CSRFMiddleware returns middleware that enforces CSRF protection
// using the signed double-submit cookie pattern. See the package-level
// comment for the algorithm and threat model.
//
// Behavior:
//   - GET/HEAD/OPTIONS/TRACE: pass through; ensure a valid token
//     cookie is set (issuing one if missing or invalid).
//   - Authorization: Bearer …: pass through unchanged. Bearer auth is
//     not vulnerable to CSRF because browsers do not auto-attach
//     custom Authorization headers cross-site.
//   - Other methods: require a valid cookie AND a matching header
//     value. Mismatches return 403.
//
// Panics if cfg.MasterSecret is shorter than 32 bytes.
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	if len(cfg.MasterSecret) < 32 {
		panic("aegis: CSRFMiddleware requires a MasterSecret of at least 32 bytes")
	}
	applyCSRFDefaults(&cfg)

	signingKey := DeriveSecret(cfg.MasterSecret, "csrf-token", DefaultSecretLength)

	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			// Bearer-token requests are CSRF-immune: browsers will not
			// auto-send an Authorization header on a forged cross-site
			// request. Skip enforcement (and skip cookie issuance —
			// pure-API clients don't need it).
			if isBearerRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			cookieVal, hasCookie := readCSRFCookie(r, cfg.CookieName)
			cookieValid := hasCookie && verifyCSRFToken(signingKey, cookieVal)

			if isSafeMethod(r.Method) {
				if !cookieValid {
					if err := issueCSRFCookie(w, cfg, signingKey); err != nil {
						WriteJSONError(w, http.StatusInternalServerError, "failed to issue CSRF token")
						return
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			// Unsafe method: enforce.
			if !cookieValid {
				WriteJSONError(w, http.StatusForbidden, "CSRF cookie missing or invalid")
				return
			}

			headerVal := r.Header.Get(cfg.HeaderName)
			if headerVal == "" {
				// Some frameworks place the token in a form field
				// named the same as the header; accept that as a
				// fallback for traditional <form> submissions.
				r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
				if val := r.PostFormValue(cfg.HeaderName); val != "" {
					headerVal = val
				}
			}
			if headerVal == "" || !constantTimeEqualString(headerVal, cookieVal) {
				WriteJSONError(w, http.StatusForbidden, "CSRF token missing or mismatched")
				return
			}

			if len(cfg.TrustedOrigins) > 0 && !originAllowed(r, cfg.TrustedOrigins) {
				WriteJSONError(w, http.StatusForbidden, "CSRF origin not allowed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func applyCSRFDefaults(cfg *CSRFConfig) {
	if cfg.CookieName == "" {
		cfg.CookieName = "aegis_csrf"
	}
	if cfg.HeaderName == "" {
		cfg.HeaderName = "X-CSRF-Token"
	}
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/"
	}
	if cfg.SameSite == 0 {
		cfg.SameSite = http.SameSiteLaxMode
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 12 * time.Hour
	}
}

// generateCSRFToken creates a fresh signed token: base64(nonce || hmac).
func generateCSRFToken(signingKey []byte) (string, error) {
	nonce := make([]byte, csrfNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(nonce)
	sig := mac.Sum(nil)
	combined := make([]byte, 0, len(nonce)+len(sig))
	combined = append(combined, nonce...)
	combined = append(combined, sig...)
	return base64.RawURLEncoding.EncodeToString(combined), nil
}

// verifyCSRFToken checks that token decodes to nonce||hmac and that the
// HMAC matches under signingKey. Constant-time.
func verifyCSRFToken(signingKey []byte, token string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	if len(raw) != csrfNonceLen+sha256.Size {
		return false
	}
	nonce := raw[:csrfNonceLen]
	sig := raw[csrfNonceLen:]
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(nonce)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

func issueCSRFCookie(w http.ResponseWriter, cfg CSRFConfig, signingKey []byte) error {
	token, err := csrfTokenGenerator(signingKey)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure/SameSite/HttpOnly are caller-controlled via CSRFConfig
		Name:     cfg.CookieName,
		Value:    token,
		Path:     cfg.CookiePath,
		Domain:   cfg.CookieDomain,
		MaxAge:   int(cfg.MaxAge.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: false, // client JS must read this to echo it back
		SameSite: cfg.SameSite,
	})
	return nil
}

func readCSRFCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func isBearerRequest(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ")
}

func originAllowed(r *http.Request, allowed []string) bool {
	candidate := r.Header.Get("Origin")
	if candidate == "" {
		// Fall back to Referer's origin component.
		ref := r.Header.Get("Referer")
		if ref == "" {
			// Without either header we cannot make a positive
			// assertion; refuse rather than guess.
			return false
		}
		// Strip path: keep scheme://host[:port]
		if i := strings.Index(ref, "://"); i != -1 {
			rest := ref[i+3:]
			if j := strings.IndexByte(rest, '/'); j != -1 {
				candidate = ref[:i+3+j]
			} else {
				candidate = ref
			}
		}
	}
	for _, a := range allowed {
		if candidate == a {
			return true
		}
	}
	return false
}

// constantTimeEqualString compares two strings of arbitrary length in
// constant time relative to the longer input. Wraps subtle.ConstantTimeCompare
// without leaking length via early-return.
func constantTimeEqualString(a, b string) bool {
	if len(a) != len(b) {
		// Still consume both to avoid a length-only timing oracle.
		// hmac.Equal-style: hash both sides to fixed length first.
		ha := sha256.Sum256([]byte(a))
		hb := sha256.Sum256([]byte(b))
		return hmac.Equal(ha[:], hb[:]) // never true here, but constant-time
	}
	return hmac.Equal([]byte(a), []byte(b))
}
