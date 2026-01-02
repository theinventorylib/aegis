package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// IDStrategy defines the algorithm used for generating unique identifiers.
//
// Aegis supports multiple ID generation strategies to accommodate different
// use cases and preferences.
type IDStrategy string

// IDGeneratorFunc is a function type for custom ID generation.
// Implement this to use your own ID generation algorithm (KSUIDs, nanoid, Snowflake, etc.).
type IDGeneratorFunc func() string

const (
	// IDStrategyULID uses ULID (Universally Unique Lexicographically Sortable Identifier).
	// This is the DEFAULT strategy.
	//
	// Benefits:
	//   - Sortable: IDs are ordered by creation time
	//   - Compact: 26 characters (vs 36 for UUID)
	//   - No configuration needed: Works immediately
	//   - Collision resistant: 80 bits of randomness
	//   - Database friendly: Efficient indexing due to sortability
	//
	// Format: 01ARZ3NDEKTSV4RRFFQ69G5FAV (26 characters)
	// Structure: 10-byte timestamp + 16-byte randomness
	//
	// Best for: Most use cases, especially when you need sortable IDs
	IDStrategyULID IDStrategy = "ulid"

	// IDStrategyUUID uses UUID v4 (random UUIDs).
	//
	// Benefits:
	//   - Standard format: Widely recognized and supported
	//   - Maximum randomness: 122 bits of entropy
	//   - Collision resistant: Extremely low probability of collisions
	//
	// Drawbacks:
	//   - Not sortable: IDs are random, not time-ordered
	//   - Longer: 36 characters with hyphens
	//   - Database indexing: Less efficient than ULIDs
	//
	// Format: 550e8400-e29b-41d4-a716-446655440000 (36 characters)
	//
	// Best for: When you need standard UUID format or maximum randomness
	IDStrategyUUID IDStrategy = "uuid"

	// IDStrategyCustom uses a user-provided custom ID generation function.
	//
	// Use this when you need:
	//   - KSUID (K-Sortable Unique Identifier)
	//   - Snowflake IDs (Twitter's distributed ID generation)
	//   - Nanoid (shorter, URL-safe IDs)
	//   - Database-generated IDs (SERIAL, AUTO_INCREMENT)
	//   - Custom formatting or business logic
	//
	// Set the custom generator with:
	//   core.SetCustomIDGenerator(func() string { return myIDGenerator() })
	//
	// Best for: Specialized requirements or existing ID systems
	IDStrategyCustom IDStrategy = "custom"
)

var (
	// currentIDStrategy holds the active ID generation strategy (default: ULID)
	currentIDStrategy = IDStrategyULID

	// ulidEntropy is the cryptographically secure random source for ULID generation.
	// Uses monotonic mode to ensure IDs are strictly increasing even within the same
	// millisecond (prevents sorting issues in high-throughput scenarios).
	ulidEntropy = ulid.Monotonic(rand.Reader, 0)

	// customIDGenerator holds the user-provided ID generation function (nil by default)
	customIDGenerator IDGeneratorFunc
)

// SetIDStrategy sets the global ID generation strategy for the application.
//
// This should be called during application initialization, before any IDs are generated.
// Changing the strategy after IDs have been generated may cause inconsistent ID formats.
//
// Example:
//
//	// Switch to UUID v4
//	core.SetIDStrategy(core.IDStrategyUUID)
//
//	// Use ULID (default)
//	core.SetIDStrategy(core.IDStrategyULID)
func SetIDStrategy(strategy IDStrategy) {
	currentIDStrategy = strategy
}

// SetCustomIDGenerator sets a custom ID generation function.
//
// This automatically switches the ID strategy to IDStrategyCustom. The provided
// function will be called every time GenerateID() is invoked.
//
// Your custom generator should:
//   - Return unique IDs (collision-free)
//   - Be thread-safe if called concurrently
//   - Generate IDs quickly (called frequently)
//
// Example (using KSUID):
//
//	import "github.com/segmentio/ksuid"
//	core.SetCustomIDGenerator(func() string {
//		return ksuid.New().String()
//	})
//
// Example (using database sequences - NOT recommended for distributed systems):
//
//	var counter uint64
//	core.SetCustomIDGenerator(func() string {
//		return fmt.Sprintf("%d", atomic.AddUint64(&counter, 1))
//	})
func SetCustomIDGenerator(generator IDGeneratorFunc) {
	customIDGenerator = generator
	currentIDStrategy = IDStrategyCustom
}

// GetIDStrategy returns the currently active ID generation strategy.
//
// Useful for logging or debugging to verify which strategy is in use.
func GetIDStrategy() IDStrategy {
	return currentIDStrategy
}

// GenerateOTPCode generates a random numeric OTP (One-Time Password) code.
//
// The code uses cryptographically secure randomness (crypto/rand) and includes
// leading zeros to ensure the specified length.
//
// Parameters:
//   - length: Number of digits (typically 4-8)
//
// Common lengths:
//   - 6 digits: Standard for most 2FA systems (Google Authenticator, etc.)
//   - 4 digits: Short codes for SMS (balance security vs user convenience)
//   - 8 digits: High-security scenarios
//
// Returns a numeric string with leading zeros if necessary.
//
// Example:
//
//	// Generate 6-digit code
//	code, _ := core.GenerateOTPCode(6) // "042816", "912345", etc.
//
//	// Generate 4-digit code for SMS
//	code, _ := core.GenerateOTPCode(4) // "0042", "9123", etc.
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

	// Format with leading zeros
	format := fmt.Sprintf("%%0%dd", length)
	code := fmt.Sprintf(format, n)

	return code, nil
}

// GenerateID generates a unique identifier using the configured ID strategy.
//
// This is the primary ID generation function used throughout Aegis for:
//   - User IDs
//   - Session IDs
//   - Account IDs
//   - Verification token IDs
//
// Default Strategy: ULID
//   - Format: "01ARZ3NDEKTSV4RRFFQ69G5FAV" (26 characters)
//   - Sortable by creation time
//   - Database-friendly indexing
//   - No configuration required
//
// Strategy Selection:
//
//   - ULID (default): Best for most use cases
//
//   - Sortable IDs improve database performance
//
//   - Compact format (26 chars vs 36 for UUID)
//
//   - Built-in timestamp makes debugging easier
//
//   - UUID: When you need standard UUID format
//
//   - Format: "550e8400-e29b-41d4-a716-446655440000"
//
//   - Maximum randomness (122 bits)
//
//   - Not sortable (random ordering)
//
//   - Custom: For specialized requirements
//
//   - Implement IDGeneratorFunc
//
//   - Examples: KSUID, Snowflake, nanoid, database sequences
//
// Usage Examples:
//
//	// Default (ULID)
//	userID := core.GenerateID() // "01ARZ3NDEKTSV4RRFFQ69G5FAV"
//
//	// Switch to UUID
//	core.SetIDStrategy(core.IDStrategyUUID)
//	userID := core.GenerateID() // "550e8400-e29b-41d4-a716-446655440000"
//
//	// Use custom generator
//	core.SetCustomIDGenerator(func() string { return ksuid.New().String() })
//	userID := core.GenerateID() // Custom format
//
// Note: For database-generated IDs (SERIAL, AUTO_INCREMENT), configure your
// database schema to generate IDs and don't call this function.
func GenerateID() string {
	switch currentIDStrategy {
	case IDStrategyULID:
		// Generate ULID with monotonic entropy for sortability
		// Even if multiple IDs are generated in the same millisecond, they'll be unique and sorted
		return ulid.MustNew(ulid.Timestamp(time.Now()), ulidEntropy).String()
	case IDStrategyUUID:
		return uuid.New().String()
	case IDStrategyCustom:
		if customIDGenerator != nil {
			return customIDGenerator()
		}
		// Fallback to ULID if custom generator is nil
		return ulid.MustNew(ulid.Timestamp(time.Now()), ulidEntropy).String()
	default:
		// Default to ULID
		return ulid.MustNew(ulid.Timestamp(time.Now()), ulidEntropy).String()
	}
}

// // GenerateUUID always generates a UUID v4, regardless of strategy
// // Use this when you specifically need a UUID
// func GenerateUUID() string {
// 	return uuid.New().String()
// }
// // GenerateSequenceID always generates a sequential ID, regardless of strategy
// // Use this when you specifically need a sequence
// func GenerateSequenceID() string {
// 	id := atomic.AddUint64(&sequenceCounter, 1)
// 	return fmt.Sprintf("%d", id)
// }
