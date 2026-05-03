package oauth

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/theinventorylib/aegis/core"
)

// StateStore manages OAuth state cookies for CSRF protection during OAuth flows.
//
// OAuth CSRF Attack Without State:
//  1. Attacker initiates OAuth flow → gets callback URL with auth code
//  2. Attacker tricks victim into visiting callback URL
//  3. Victim's browser exchanges code → victim's account linked to attacker's provider
//  4. Attacker can now access victim's account via OAuth
//
// CSRF Protection With State:
//  1. Plugin generates random state token before redirect
//  2. State stored in signed cookie (attacker can't forge)
//  3. Provider includes state in callback URL
//  4. Plugin validates callback state matches cookie
//  5. If mismatch → reject (CSRF attempt)
//
// State Cookie Contents:
//   - CSRF state token (32 random bytes, base64-encoded)
//   - Provider name ("google", "github", etc.)
//   - Marshaled OAuth session (for resuming flow)
//   - HMAC signature (prevents tampering)
//
// Security Features:
//   - HMAC-SHA256 signing with derived secret (prevents cookie tampering)
//   - Gzip compression (session data can be large)
//   - Short expiration (15 minutes default)
//   - HTTPOnly, Secure, SameSite settings from CookieManager
//
// Integration:
// This store integrates with Aegis's core CookieManager for consistent cookie
// settings (domain, secure flag, SameSite policy) across all cookies.
type StateStore struct {
	cookieManager *core.CookieManager // Handles cookie settings (Secure, HTTPOnly, SameSite)
	secret        []byte              // HMAC signing key (derived from master secret)
	maxAge        time.Duration       // State cookie expiration (15 minutes default)
	mu            sync.RWMutex        // Protects maxAge (rarely changed)
}

// StateStoreConfig holds configuration for creating a StateStore.
//
// Example:
//
//	cfg := &oauth.StateStoreConfig{
//	    SessionConfig: aegis.GetSessionConfig(), // Domain, Secure, HTTPOnly, SameSite
//	    Secret: aegis.DeriveSecret("aegis:oauth-state"), // 32+ bytes
//	    MaxAge: 15 * 60, // 15 minutes
//	}
//	store := oauth.NewStateStore(cfg)
type StateStoreConfig struct {
	// SessionConfig contains cookie settings (Domain, Secure, HTTPOnly, SameSite)
	SessionConfig *core.SessionConfig
	// Secret is the key used for signing cookies (should be at least 32 bytes)
	Secret []byte
	// MaxAge overrides the default OAuth state cookie max age in seconds (default: 15 minutes)
	MaxAge int
}

// StateCookieName is the cookie name used for OAuth state storage.
// This cookie is separate from the session cookie and is only used during
// the OAuth flow (created on BeginAuth, validated on callback, then deleted).
const StateCookieName = "_aegis_oauth_state"

// NewStateStore creates a new StateStore for managing OAuth state with CSRF protection.
//
// The secret should be at least 32 bytes for cryptographic security. Aegis derives
// this from the master secret using the purpose "aegis:oauth-state".
//
// Parameters:
//   - cfg: Configuration with session settings, secret, and max age
//
// Returns:
//   - *StateStore: Initialized store ready for use
//
// Example:
//
//	secret := aegis.DeriveSecret("aegis:oauth-state")
//	store := oauth.NewStateStore(&oauth.StateStoreConfig{
//	    SessionConfig: aegis.GetSessionConfig(),
//	    Secret: secret,
//	    MaxAge: 15 * 60, // 15 minutes
//	})
//
// NewStateStore creates a new StateStore for managing OAuth state with CSRF protection.
//
// The secret is REQUIRED and must be at least 16 bytes. Aegis derives this from
// the master secret via DeriveSecret("aegis:oauth-state") so it is deterministic
// across restarts and shared across instances.
//
// Returns an error if cfg.Secret is empty or shorter than 16 bytes — accepting
// unsigned cookies would let an attacker forge state cookies and bypass the
// OAuth CSRF check entirely.
//
// Parameters:
//   - cfg: Configuration with session settings, secret, and max age
//
// Returns:
//   - *StateStore: Initialized store ready for use
//   - error: Configuration error (missing/short secret)
func NewStateStore(cfg *StateStoreConfig) (*StateStore, error) {
	if cfg == nil {
		cfg = &StateStoreConfig{}
	}

	if len(cfg.Secret) < 16 {
		return nil, fmt.Errorf("oauth state store: secret must be at least 16 bytes (got %d); derive one from your Aegis master secret", len(cfg.Secret))
	}

	sessionCfg := cfg.SessionConfig
	if sessionCfg == nil {
		sessionCfg = core.DefaultSessionConfig()
	}

	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 15 * 60 // 15 minutes default for OAuth state
	}

	return &StateStore{
		cookieManager: core.NewCookieManager(sessionCfg),
		secret:        cfg.Secret,
		maxAge:        time.Duration(maxAge) * time.Second,
	}, nil
}

// StateData holds the OAuth state data stored during the OAuth flow.
//
// This data is serialized, compressed, signed, and stored in a cookie during
// BeginAuth, then retrieved and validated during the callback.
//
// Cookie Format (with signature):
//
//	<hmac-signature>.<base64(state|provider|base64(gzip(sessionData)))>
//
// Example:
//
//	"a8f3d..." (HMAC) + "." + "YWJjMTIzfGdvb2dsZXxINHNJQUFBLi4u" (payload)
type StateData struct {
	State       string    // CSRF state token (32 random bytes, base64)
	Provider    string    // OAuth provider name ("google", "github", etc.)
	SessionData string    // Marshaled goth.Session (provider-specific OAuth state)
	IssuedAt    time.Time // When the state was issued (server-side expiry source)
}

// GenerateState generates a cryptographically secure random state string.
//
// The state is a 32-byte random value base64-encoded for use as a CSRF token.
// This is the value included in the OAuth authorization URL and validated in
// the callback.
//
// Returns:
//   - string: Base64-encoded random state (44 characters)
//   - error: Crypto random generation error (extremely rare)
func (s *StateStore) GenerateState() (string, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(nonceBytes), nil
}

// StoreState stores OAuth state data in a signed, compressed cookie.
//
// This method is called at the start of the OAuth flow (BeginAuth) to save the
// state data for validation during the callback. The cookie is HTTPOnly and
// Secure (if configured) to prevent JavaScript access and MITM attacks.
//
// Processing:
//  1. Compress SessionData with gzip (can be large ~1-2KB)
//  2. Concatenate: state|provider|compressedSession
//  3. Sign with HMAC-SHA256 if secret is set
//  4. Base64-encode payload
//  5. Store in cookie with configured expiration (15 minutes default)
//
// Parameters:
//   - w: HTTP response writer for setting cookie
//   - data: State data to store
//
// Returns:
//   - error: Compression or encoding error
func (s *StateStore) StoreState(w http.ResponseWriter, data *StateData) error {
	// Encode the state data
	encoded, err := s.encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// Use CookieManager to set the cookie with consistent settings
	s.cookieManager.SetCookie(w, StateCookieName, encoded, s.maxAge)
	return nil
}

// GetState retrieves and validates the OAuth state data from the cookie.
//
// This method is called during the OAuth callback to retrieve the stored state
// data for validation. It verifies the HMAC signature to ensure the cookie
// wasn't tampered with.
//
// Processing:
//  1. Read cookie value
//  2. Base64-decode payload
//  3. Verify HMAC signature (if secret is set)
//  4. Parse: state|provider|compressedSession
//  5. Decompress SessionData with gzip
//
// Parameters:
//   - r: HTTP request with state cookie
//
// Returns:
//   - *StateData: Decoded state data
//   - error: Cookie not found, invalid signature, or decoding error
func (s *StateStore) GetState(r *http.Request) (*StateData, error) {
	value, err := s.cookieManager.GetCookie(r, StateCookieName)
	if err != nil {
		return nil, fmt.Errorf("oauth state cookie not found: %w", err)
	}

	data, err := s.decode(value)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth state cookie: %w", err)
	}

	return data, nil
}

// ClearState clears the OAuth state cookie.
//
// This method is called after successful callback validation to delete the
// one-time-use state cookie. It sets MaxAge=-1 to instruct the browser to
// delete the cookie immediately.
//
// Parameters:
//   - w: HTTP response writer for clearing cookie
func (s *StateStore) ClearState(w http.ResponseWriter) {
	// Use CookieManager's custom cookie to clear with MaxAge -1
	s.cookieManager.SetCustomCookie(w, core.CookieOptions{
		Name:   StateCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// ValidateState checks if the callback state matches the stored state.
//
// This is the critical CSRF protection check. It ensures the OAuth callback
// originated from a legitimate authorization flow initiated by this server.
//
// Validation:
//  1. Retrieve state data from cookie
//  2. Compare stored.State with callbackState from query parameter
//  3. If mismatch → return error (possible CSRF attack)
//
// Parameters:
//   - r: HTTP request with state cookie
//   - callbackState: State parameter from OAuth callback URL
//
// Returns:
//   - *StateData: Validated state data (if state matches)
//   - error: State mismatch (CSRF) or cookie retrieval error
func (s *StateStore) ValidateState(r *http.Request, callbackState string) (*StateData, error) {
	stored, err := s.GetState(r)
	if err != nil {
		return nil, err
	}

	if !hmac.Equal([]byte(stored.State), []byte(callbackState)) {
		return nil, fmt.Errorf("state mismatch: possible CSRF attack")
	}

	return stored, nil
}

// encode serializes and signs the state data
func (s *StateStore) encode(data *StateData) (string, error) {
	// Format: state|provider|issuedAtUnix|sessionData
	// We compress the session data since it can be large
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(data.SessionData)); err != nil {
		return "", fmt.Errorf("failed to compress: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip: %w", err)
	}

	compressedSession := base64.URLEncoding.EncodeToString(buf.Bytes())
	issuedAt := time.Now().Unix()
	payload := fmt.Sprintf("%s|%s|%d|%s", data.State, data.Provider, issuedAt, compressedSession)

	// Sign the payload. NewStateStore guarantees a non-empty secret, so
	// we never emit an unsigned cookie that an attacker could forge.
	signature := s.sign(payload)
	return signature + "." + base64.URLEncoding.EncodeToString([]byte(payload)), nil
}

// decode verifies signature and deserializes state data.
//
// The signature is mandatory: NewStateStore rejects empty secrets, so any
// cookie reaching this code path must carry a valid HMAC. Cookies without
// a "<sig>.<payload>" structure or with an invalid signature are rejected.
func (s *StateStore) decode(value string) (*StateData, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cookie format")
	}
	signature, encodedPayload := parts[0], parts[1]

	payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}
	payload := string(payloadBytes)

	expectedSig := s.sign(payload)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid cookie signature")
	}

	// Parse payload: state|provider|issuedAtUnix|compressedSession
	parts = strings.SplitN(payload, "|", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid payload format")
	}

	// Validate issued-at and enforce server-side expiry. Even though the
	// browser cookie has its own MaxAge, we cannot trust the client to
	// honor it — a stolen cookie could be replayed long after expiry.
	issuedAtUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid issued-at: %w", err)
	}
	issuedAt := time.Unix(issuedAtUnix, 0)
	s.mu.RLock()
	maxAge := s.maxAge
	s.mu.RUnlock()
	now := time.Now()
	if now.Sub(issuedAt) > maxAge {
		return nil, fmt.Errorf("oauth state cookie expired")
	}
	// Reject tokens dated in the future (allow small clock skew).
	if issuedAt.After(now.Add(1 * time.Minute)) {
		return nil, fmt.Errorf("oauth state cookie issued in the future")
	}

	// Decompress session data
	compressedSession, err := base64.URLEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(compressedSession))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { err := gz.Close(); _ = err }()

	sessionData, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress session: %w", err)
	}

	return &StateData{
		State:       parts[0],
		Provider:    parts[1],
		IssuedAt:    issuedAt,
		SessionData: string(sessionData),
	}, nil
}

// sign creates an HMAC signature for the given data
func (s *StateStore) sign(data string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// GetMaxAge returns the max age for state cookies
func (s *StateStore) GetMaxAge() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxAge
}

// SetMaxAge sets the max age for state cookies
func (s *StateStore) SetMaxAge(age time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxAge = age
}
