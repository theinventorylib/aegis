package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/oklog/ulid/v2"
	"uuid"
)

// IDStrategy defines the algorithm used for generating unique identifiers.
type IDStrategy string

// iDGeneratorFunc is a function type for custom ID generation.
// Implement this to use your own ID generation algorithm (KSUIDs, nanoid, Snowflake, etc.).
type iDGeneratorFunc func() string

const (
	// IDStrategyULID uses ULID (Universally Unique Lexicographically Sortable
	// Identifier) — the default. Sortable by creation time, compact (26
	// characters), database-index friendly.
	IDStrategyULID IDStrategy = "ulid"

	// IDStrategyUUID uses UUID v4 (random UUIDs). Standard format, maximum
	// randomness, but not sortable.
	IDStrategyUUID IDStrategy = "uuid"

	// IDStrategyCustom uses a user-provided custom ID generation function
	// (KSUID, Snowflake, nanoid, database sequences, …). Set the generator
	// with core.SetCustomIDGenerator.
	IDStrategyCustom IDStrategy = "custom"
)

// idConfig holds the ID generation configuration
type idConfig struct {
	strategy  IDStrategy
	entropy   io.Reader
	generator iDGeneratorFunc
}

// defaultIDConfig is the package-level ID generation config (default: ULID).
// entropy uses crypto/rand.Reader directly: it is concurrency-safe and avoids
// the shared mutable state inside ulid.Monotonic.
var defaultIDConfig = &idConfig{
	strategy:  IDStrategyULID,
	entropy:   rand.Reader,
	generator: nil,
}

// SetIDStrategy sets the global ID generation strategy for the application.
//
// Call during application initialization, before any IDs are generated;
// changing it later produces inconsistent ID formats.
func SetIDStrategy(strategy IDStrategy) {
	defaultIDConfig.strategy = strategy
}

// SetCustomIDGenerator sets a custom ID generation function and switches the
// strategy to IDStrategyCustom. The generator must return unique IDs and be
// thread-safe if called concurrently.
func SetCustomIDGenerator(generator func() string) {
	defaultIDConfig.generator = generator
	defaultIDConfig.strategy = IDStrategyCustom
}

// GetIDStrategy returns the currently active ID generation strategy.
func GetIDStrategy() IDStrategy {
	return defaultIDConfig.strategy
}

// GenerateOTPCode generates a random numeric OTP code of the given length
// using cryptographically secure randomness, with leading zeros preserved.
//
//	code, _ := core.GenerateOTPCode(6) // "042816", "912345", …
func GenerateOTPCode(length int) (string, error) {
	if length <= 0 {
		return "", ValidationError{Field: "length", Message: "OTP length must be positive"}
	}

	// Calculate the maximum value for the OTP (e.g., 999999 for 6 digits)
	maxValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	maxValue.Sub(maxValue, big.NewInt(1))

	n, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate OTP", err)
	}

	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n), nil
}

// GenerateID generates a unique identifier using the configured ID strategy.
// Defaults to ULID (26 chars, sortable by creation time); switch with
// SetIDStrategy / SetCustomIDGenerator before generating IDs.
func GenerateID() string {
	switch defaultIDConfig.strategy {
	case IDStrategyULID:
		// ULID is sortable by creation time. IDs generated in the same
		// millisecond are unique but not ordered among themselves.
		return ulid.MustNew(ulid.Timestamp(time.Now()), defaultIDConfig.entropy).String()
	case IDStrategyUUID:
		return uuid.New().String()
	case IDStrategyCustom:
		if defaultIDConfig.generator != nil {
			return defaultIDConfig.generator()
		}
		// Fallback to ULID if custom generator is nil
		return ulid.MustNew(ulid.Timestamp(time.Now()), defaultIDConfig.entropy).String()
	default:
		// Default to ULID
		return ulid.MustNew(ulid.Timestamp(time.Now()), defaultIDConfig.entropy).String()
	}
}

// ClampIntToInt32 converts an int to int32, clamping to the valid int32
// range. Prevents overflow on platforms where int is larger than 32 bits and
// guards against untrusted values.
func ClampIntToInt32(n int) int32 {
	if n <= 0 {
		return 0
	}
	if n > math.MaxInt32 {
		return int32(math.MaxInt32)
	}
	return int32(n)
}

// redactForLog returns a masked version of a user-provided identifier
// suitable for logs, preserving only a non-sensitive hint:
//
//	"alice@example.com" -> "a***@example.com"
//	"+1234567890"       -> "+1******90"
func redactForLog(s string) string {
	if s == "" {
		return ""
	}
	// Email-like: keep first char and domain
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			local := s[:i]
			domain := s[i+1:]
			if len(local) <= 1 {
				return "***@" + domain
			}
			return fmt.Sprintf("%c***@%s", local[0], domain)
		}
	}

	// Phone-like: keep country/first char and last two chars
	if len(s) >= 4 {
		return fmt.Sprintf("%s***%s", s[:1], s[len(s)-2:])
	}

	// Fallback: keep first and last char
	if len(s) == 2 {
		return s
	}
	if len(s) == 1 {
		return "*"
	}
	return fmt.Sprintf("%c***%c", s[0], s[len(s)-1])
}

// hashShort returns a short hash (first 8 hex chars of SHA-256) of the input,
// suitable for non-reversible identification in logs.
func hashShort(s string) string {
	if s == "" {
		return ""
	}
	h := sha256.Sum256([]byte(s))
	// return first 8 hex chars for brevity
	return fmt.Sprintf("%x", h)[:8]
}

// hashTokenHex returns the SHA-256 hex digest of a token, suitable for
// at-rest persistence and cache keying. The output is 64 lowercase hex
// characters; an empty input yields an empty string. SHA-256 is appropriate
// here because the input is high-entropy random bytes (not a low-entropy
// password) — no per-token salt or cost factor is required for collision
// resistance or pre-image protection at this entropy level.
func hashTokenHex(token string) string {
	if token == "" {
		return ""
	}
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// isHashedToken reports whether s looks like a SHA-256 hex digest produced
// by hashTokenHex. Used by migration helpers to skip already-hashed rows
// for idempotency.
func isHashedToken(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
