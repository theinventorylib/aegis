package core

import (
	"github.com/theinventorylib/aegis/auth"
)

// Compile-time guard for the v1 public API.
//
// The v1.6 identifiers below are kept as deprecated shims on the v1 line (see
// deprecated.go). Referencing them here means removing one breaks the build
// before it can ship as an accidental breaking change. This file exists only on
// the v1 branch; main (v2) deliberately drops these symbols.
type compatReq struct{}

func (compatReq) Validate() error { return nil }

var (
	// Constructors.
	_ = NewSessionService
	_ = NewAccountService
	_ = NewUserService
	_ = NewVerificationService
	_ = NewPluginData

	// Validation / errors.
	_ = ValidatePassword
	_ = ValidatePasswordSimple
	_ = BindAndValidate[compatReq]
	_ = ValidateMiddleware[compatReq]
	_ = WrapError
	_ = IsValidationError

	// Context.
	_ = MustGetUser
	_ = MustGetEnrichedUser
	_ = IsContextInitialized
	_ = NewAegisContext
	_ AegisContext

	// Config.
	_ = AuthRateLimitConfig
	_ = DefaultPasswordHasherConfig
	_ = DefaultPasswordPolicyConfig
	_ = SetCustomIDGenerator
	_ = EmailRegexPattern
	_ = (*AuthService).GetAuthConfig

	// Sanitizers / utilities.
	_ = SanitizeFilename
	_ = SanitizeHTML
	_ = SanitizeSQL
	_ = SanitizeSQLIdentifier
	_ = StripTags
	_ = NormalizeWhitespace
	_ = RedactForLog
	_ = HashShort
	_ = HashTokenHex
	_ = IsHashedToken
	_ = BoolPtr
	_ = SanitizationConfig{NormalizeWhitespace: true}

	// Audit adapter.
	_ = NewLoggerAuditLogger
	_ LoggerAuditLogger

	// Types.
	_ IDGeneratorFunc   = func() string { return "" }
	_ AccountModel      = (*auth.Account)(nil)
	_ VerificationModel = (*auth.Verification)(nil)
)
