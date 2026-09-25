// Package totp adds TOTP (RFC 6238) two-factor authentication to Aegis.
//
// The plugin extends the user table with a pending/enabled credential and keeps
// per-session verification in its own table, so a session that has not cleared
// the second factor can be restricted with RequireVerification without touching
// the core session schema.
//
// Flow:
//  1. POST /totp/setup          → returns a secret + otpauth URL (not yet enabled)
//  2. POST /totp/enable         → confirms a code, enables TOTP, returns recovery codes
//  3. POST /totp/verify         → step-up: marks the current session verified
//  4. POST /totp/disable        → removes the credential and session verifications
//  5. GET  /totp/status         → enabled + current-session verification state
//
// Recovery codes are stored as one-time core verification tokens, so they are
// hashed at rest and consumed on use without a plugin-owned table.
package totp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 mandates HMAC-SHA1 for authenticator compatibility
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
	"github.com/theinventorylib/aegis/v2/config"
	"github.com/theinventorylib/aegis/v2/core"
	iversion "github.com/theinventorylib/aegis/v2/internal/version"
	"github.com/theinventorylib/aegis/v2/plugins"
	"github.com/theinventorylib/aegis/v2/plugins/openapi"
	totpdefaultstore "github.com/theinventorylib/aegis/v2/plugins/totp/default_store"
	totptypes "github.com/theinventorylib/aegis/v2/plugins/totp/types"
	"github.com/theinventorylib/aegis/v2/router"
)

const (
	// verificationTypeRecovery is the verification type used for recovery codes.
	verificationTypeRecovery = "totp_recovery"

	// recoveryExpiry is how long generated recovery codes stay valid. They are
	// one-time tokens, so a long window is acceptable; regeneration
	// invalidates outstanding codes.
	recoveryExpiry = 10 * 365 * 24 * time.Hour

	// defaultIssuer labels credentials in authenticator apps.
	defaultIssuer = "Aegis"
)

// Sentinel errors returned by the programmatic API.
var (
	ErrAlreadyEnabled = errors.New("two-factor is already enabled")
	ErrNotSetup       = errors.New("two-factor setup has not been started")
	ErrNotEnabled     = errors.New("two-factor is not enabled")
	ErrInvalidCode    = errors.New("invalid code")
)

// verificationAPI is the subset of core.VerificationService the plugin uses for
// recovery codes. Defined here so tests can substitute a fake.
type verificationAPI interface {
	CreateVerification(ctx context.Context, identifier, vType string, expiry time.Duration, customToken *string) (auth.Verification, error)
	InvalidateVerification(ctx context.Context, identifier, vType string) error
	ValidateVerificationFor(ctx context.Context, identifier, vType, token string) (auth.Verification, error)
}

// Config holds TOTP plugin configuration.
type Config struct {
	// Issuer labels credentials in authenticator apps (default: "Aegis").
	Issuer string
	// Digits is the code length (default: 6).
	Digits int
	// Period is the time step (default: 30s).
	Period time.Duration
	// Skew is how many steps before/after now are accepted (default: 1).
	Skew int
	// SecretBytes is the entropy of generated secrets (default: 20).
	SecretBytes int
	// RecoveryCodes is how many single-use recovery codes are generated on
	// enable (default: 10). Set it to a negative value to disable recovery
	// codes; 0 means "use the default", so an unset field cannot silently
	// drop recovery codes.
	RecoveryCodes int
	// SessionTTL is how long a session stays verified after a successful code
	// (default: 12h).
	SessionTTL time.Duration
}

// Plugin provides TOTP two-factor authentication.
type Plugin struct {
	issuer         string
	digits         int
	period         time.Duration
	skew           int
	secretBytes    int
	recoveryCodes  int
	sessionTTL     time.Duration
	store          totptypes.Store
	verification   verificationAPI
	sessionService *core.SessionService
	logger         config.Logger
	aegis          plugins.Aegis
	dialect        plugins.Dialect
}

// New creates a new TOTP plugin instance.
//
// Parameters:
//   - cfg: Plugin configuration (can be nil for defaults)
//   - store: Custom Store implementation (can be nil, will use DefaultTOTPStore)
//   - dialect: Database dialect (optional, defaults to PostgreSQL)
func New(cfg *Config, store totptypes.Store, dialect ...plugins.Dialect) *Plugin {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Issuer == "" {
		cfg.Issuer = defaultIssuer
	}
	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period == 0 {
		cfg.Period = 30 * time.Second
	}
	if cfg.Skew == 0 {
		cfg.Skew = 1
	}
	if cfg.SecretBytes == 0 {
		cfg.SecretBytes = 20
	}
	if cfg.RecoveryCodes == 0 {
		cfg.RecoveryCodes = 10
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 12 * time.Hour
	}

	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}

	return &Plugin{
		issuer:        cfg.Issuer,
		digits:        cfg.Digits,
		period:        cfg.Period,
		skew:          cfg.Skew,
		secretBytes:   cfg.SecretBytes,
		recoveryCodes: cfg.RecoveryCodes,
		sessionTTL:    cfg.SessionTTL,
		store:         store,
		dialect:       d,
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "totp" }

// Version returns the plugin version for compatibility tracking.
func (p *Plugin) Version() string { return iversion.Version }

// Description returns a human-readable description for logging.
func (p *Plugin) Description() string {
	return "TOTP two-factor authentication plugin (RFC 6238)"
}

// Init initializes the plugin.
func (p *Plugin) Init(ctx context.Context, a plugins.Aegis) error {
	authService := a.GetAuthService()
	p.sessionService = authService.Session
	p.verification = authService.Verification
	p.logger = a.GetLogger()
	p.aegis = a

	if p.store == nil {
		s, err := totpdefaultstore.NewDefaultTOTPStore(a.DB(), p.dialect)
		if err != nil {
			return fmt.Errorf("totp: failed to initialize store: %w", err)
		}
		p.store = s
	}

	d := string(p.dialect)
	requirements := []plugins.SchemaRequirement{
		plugins.ValidateTableExistsForDialect(d, "user"),
		plugins.ValidateTableExistsForDialect(d, "session"),
		plugins.ValidateColumnExistsForDialect(d, "user", "totp_secret"),
		plugins.ValidateColumnExistsForDialect(d, "user", "totp_enabled"),
		plugins.ValidateTableExistsForDialect(d, "totp_session"),
	}
	if err := a.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return fmt.Errorf("totp plugin: schema validation failed: %w", err)
	}

	return nil
}

// Setup generates a new secret and stores it in the pending (disabled) state.
// Returns the secret and an otpauth:// URL for authenticator apps. Call Enable
// with a code from the authenticator to activate it.
//
// Re-running Setup replaces a pending secret; it refuses while TOTP is enabled
// so a stolen session cannot silently re-enroll the account.
func (p *Plugin) Setup(ctx context.Context, userID string) (totptypes.SetupResponse, error) {
	cred, err := p.store.GetCredential(ctx, userID)
	if err != nil {
		return totptypes.SetupResponse{}, err
	}
	if cred.Enabled {
		return totptypes.SetupResponse{}, ErrAlreadyEnabled
	}

	secret, err := generateSecret(p.secretBytes)
	if err != nil {
		return totptypes.SetupResponse{}, err
	}
	if err := p.store.SetCredential(ctx, userID, &secret, false); err != nil {
		return totptypes.SetupResponse{}, err
	}

	return totptypes.SetupResponse{
		Secret: secret,
		URL:    otpauthURL(p.issuer, userID, secret, p.digits, p.period),
	}, nil
}

// Enable confirms a code from the pending secret and activates TOTP. It returns
// the freshly generated single-use recovery codes.
func (p *Plugin) Enable(ctx context.Context, userID, code string) ([]string, error) {
	cred, err := p.store.GetCredential(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cred.Enabled {
		return nil, ErrAlreadyEnabled
	}
	if cred.Secret == "" {
		return nil, ErrNotSetup
	}
	if !p.validCode(cred.Secret, code, time.Now()) {
		return nil, ErrInvalidCode
	}

	// Generate recovery codes before flipping the flag: if generation fails
	// the account stays disabled and the caller can retry cleanly.
	codes, err := p.generateRecoveryCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := p.store.SetCredential(ctx, userID, &cred.Secret, true); err != nil {
		return nil, err
	}
	return codes, nil
}

// Disable clears the credential, every session verification and any outstanding
// recovery codes. A valid TOTP code is required so a hijacked session cannot
// silently remove the second factor. Disabling an account that has no credential
// is a no-op, so callers stay idempotent.
func (p *Plugin) Disable(ctx context.Context, userID, code string) error {
	ok, err := p.Verify(ctx, userID, code)
	if err != nil {
		if errors.Is(err, ErrNotEnabled) {
			return nil
		}
		return err
	}
	if !ok {
		return ErrInvalidCode
	}

	if err := p.store.SetCredential(ctx, userID, nil, false); err != nil {
		return err
	}
	if err := p.store.DeleteUserSessionVerifications(ctx, userID); err != nil {
		return err
	}
	if p.verification != nil {
		if err := p.verification.InvalidateVerification(ctx, userID, verificationTypeRecovery); err != nil && p.logger != nil {
			p.logger.Error("totp: failed to invalidate recovery codes", "user_id", userID, "error", err)
		}
	}
	return nil
}

// Verify reports whether code is a valid TOTP code for the user.
func (p *Plugin) Verify(ctx context.Context, userID, code string) (bool, error) {
	cred, err := p.store.GetCredential(ctx, userID)
	if err != nil {
		return false, err
	}
	if !cred.Enabled || cred.Secret == "" {
		return false, ErrNotEnabled
	}
	return p.validCode(cred.Secret, code, time.Now()), nil
}

// VerifyRecoveryCode consumes a single-use recovery code. Invalid, expired and
// already-used codes all report false.
func (p *Plugin) VerifyRecoveryCode(ctx context.Context, userID, code string) (bool, error) {
	if p.verification == nil || strings.TrimSpace(code) == "" {
		return false, nil
	}
	if _, err := p.verification.ValidateVerificationFor(ctx, userID, verificationTypeRecovery, code); err != nil {
		// Invalid, expired and already-used codes are all failed
		// verifications, not operational errors.
		return false, nil //nolint:nilerr
	}
	return true, nil
}

// RegenerateRecoveryCodes replaces the user's recovery codes. A valid TOTP code
// is required so a hijacked session cannot mint new recovery codes.
func (p *Plugin) RegenerateRecoveryCodes(ctx context.Context, userID, code string) ([]string, error) {
	ok, err := p.Verify(ctx, userID, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCode
	}
	return p.generateRecoveryCodes(ctx, userID)
}

// IsEnabled reports whether TOTP is active for the user.
func (p *Plugin) IsEnabled(ctx context.Context, userID string) (bool, error) {
	cred, err := p.store.GetCredential(ctx, userID)
	if err != nil {
		return false, err
	}
	return cred.Enabled, nil
}

// MarkSessionVerified records that sessionID cleared the second factor.
func (p *Plugin) MarkSessionVerified(ctx context.Context, sessionID, userID string) error {
	return p.store.MarkSessionVerified(ctx, sessionID, userID, time.Now())
}

// IsSessionVerified reports whether sessionID cleared the second factor within
// the configured session TTL.
func (p *Plugin) IsSessionVerified(ctx context.Context, sessionID, userID string) bool {
	if p.store == nil {
		return false
	}
	ok, err := p.store.IsSessionVerified(ctx, sessionID, userID, time.Now().Add(-p.sessionTTL))
	return err == nil && ok
}

// RequireVerification blocks requests for users with TOTP enabled until the
// current session has cleared the second factor. Unauthenticated requests pass
// through, so this composes with RequireAuthMiddleware (mount it after auth).
func (p *Plugin) RequireVerification(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := core.GetUserID(r.Context())
		session := core.GetSession(r.Context())
		if userID == "" || session == nil {
			next.ServeHTTP(w, r)
			return
		}
		enabled, err := p.IsEnabled(r.Context(), userID)
		if err != nil {
			// Fail closed: a credential lookup error must not silently drop
			// the second factor.
			if p.logger != nil {
				p.logger.Error("totp: failed to read credential", "user_id", userID, "error", err)
			}
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "two-factor check failed",
			})
			return
		}
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}
		if p.IsSessionVerified(r.Context(), session.ID, userID) {
			next.ServeHTTP(w, r)
			return
		}
		core.WriteJSON(w, http.StatusForbidden, &core.Response{
			Success: false,
			Error:   "two-factor verification required",
			Data:    map[string]string{"code": "totp_required"},
		})
	})
}

// generateRecoveryCodes invalidates outstanding codes and mints a fresh set as
// hashed, single-use verification tokens.
func (p *Plugin) generateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	if p.verification == nil || p.recoveryCodes <= 0 {
		return nil, nil
	}
	if err := p.verification.InvalidateVerification(ctx, userID, verificationTypeRecovery); err != nil {
		return nil, err
	}
	codes := make([]string, 0, p.recoveryCodes)
	for range p.recoveryCodes {
		code, err := generateSecret(5)
		if err != nil {
			return nil, err
		}
		if _, err := p.verification.CreateVerification(ctx, userID, verificationTypeRecovery, recoveryExpiry, &code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// validCode checks code against the secret with the configured time-step skew.
func (p *Plugin) validCode(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for offset := -p.skew; offset <= p.skew; offset++ {
		unix := now.Add(time.Duration(offset) * p.period).Unix()
		if unix < 0 {
			continue
		}
		want, err := hotp(secret, uint64(unix/int64(p.period.Seconds())), p.digits) //nolint:gosec // unix >= 0 checked above
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// generateSecret returns a base32 (no padding) random secret of n bytes.
func generateSecret(n int) (string, error) {
	if n <= 0 {
		n = 20
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// hotp computes an RFC 4226 code from a base32 secret and counter.
func hotp(secret string, counter uint64, digits int) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", ErrInvalidCode
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	mod := uint32(1)
	for range digits {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, value%mod), nil
}

// otpauthURL builds the otpauth:// URI authenticator apps understand.
func otpauthURL(issuer, account, secret string, digits int, period time.Duration) string {
	label := url.PathEscape(issuer + ":" + account)
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", strconv.Itoa(digits))
	params.Set("period", strconv.Itoa(int(period.Seconds())))
	return "otpauth://totp/" + label + "?" + params.Encode()
}

// MountRoutes registers HTTP routes for the TOTP plugin.
func (p *Plugin) MountRoutes(r router.Router, prefix string) {
	handlers := NewHandlers(p)
	totpGroup := r.Group(prefix, "TOTP")
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	totpGroup.POST("/setup", requireAuth(http.HandlerFunc(handlers.SetupHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/setup",
		Summary:     "Start TOTP enrollment",
		Description: "Generate a pending TOTP secret and otpauth:// URL. TOTP is not active until /enable confirms a code.",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[totptypes.SetupResponse]("Secret and otpauth URL"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			409: openapi.RefResponse("Two-factor is already enabled", "Error"),
		},
	})

	totpGroup.POST("/enable", requireAuth(http.HandlerFunc(handlers.EnableHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/enable",
		Summary:     "Enable TOTP",
		Description: "Confirm a code from the pending secret and activate two-factor authentication. Returns single-use recovery codes.",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Body:        openapi.BodyOf[totptypes.EnableRequest](),
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[totptypes.EnableResponse]("TOTP enabled; recovery codes returned once"),
			400: openapi.RefResponse("No pending setup or invalid code", "Error"),
			409: openapi.RefResponse("Two-factor is already enabled", "Error"),
		},
	})

	totpGroup.POST("/disable", requireAuth(http.HandlerFunc(handlers.DisableHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/disable",
		Summary:     "Disable TOTP",
		Description: "Remove the TOTP credential, all session verifications and outstanding recovery codes. Requires a valid TOTP code so a hijacked session cannot remove the second factor.",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Body:        openapi.BodyOf[totptypes.DisableRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("TOTP disabled", "Success"),
			400: openapi.RefResponse("Invalid code", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	totpGroup.POST("/verify", requireAuth(http.HandlerFunc(handlers.VerifyHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/verify",
		Summary:     "Verify a TOTP code",
		Description: "Step-up the current session with a TOTP code (or a single-use recovery code when recovery=true). Sessions stay verified for the configured TTL.",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Body:        openapi.BodyOf[totptypes.VerifyRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("Code accepted; session verified", "Success"),
			400: openapi.RefResponse("Invalid code", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	totpGroup.POST("/recovery-codes", requireAuth(http.HandlerFunc(handlers.RegenerateRecoveryCodesHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/recovery-codes",
		Summary:     "Regenerate recovery codes",
		Description: "Replace the recovery codes. Requires a valid TOTP code so a hijacked session cannot mint new ones.",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Body:        openapi.BodyOf[totptypes.RegenerateRequest](),
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[totptypes.EnableResponse]("Fresh recovery codes"),
			400: openapi.RefResponse("Invalid code", "Error"),
		},
	})

	totpGroup.GET("/status", requireAuth(http.HandlerFunc(handlers.StatusHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/status",
		Summary:     "TOTP status",
		Description: "Report whether TOTP is enabled and whether the current session has cleared it",
		Tags:        []string{"TOTP"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[totptypes.StatusResponse]("TOTP status"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})
}

// Dependencies returns external package dependencies.
func (p *Plugin) Dependencies() []plugins.Dependency { return []plugins.Dependency{} }

// RequiresTables returns the core tables this plugin reads from.
func (p *Plugin) RequiresTables() []string {
	return []string{"user", "session"}
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string { return []string{"totp"} }

// GetMigrations returns the plugin migrations.
func (p *Plugin) GetMigrations() []plugins.Migration {
	migs, err := GetMigrations(p.dialect)
	if err != nil {
		return []plugins.Migration{}
	}
	return migs
}

// EnrichUser adds the TOTP-enabled flag to authenticated user responses.
func (p *Plugin) EnrichUser(ctx context.Context, user *core.EnrichedUser) error {
	if user == nil || user.User == nil {
		return nil
	}
	cred, err := p.store.GetCredential(ctx, user.ID)
	if err != nil {
		return err
	}
	user.Set("totpEnabled", cred.Enabled)
	return nil
}

// Ensure Plugin implements UserEnricher
var _ plugins.UserEnricher = (*Plugin)(nil)

// Ensure Plugin implements Plugin
var _ plugins.Plugin = (*Plugin)(nil)
