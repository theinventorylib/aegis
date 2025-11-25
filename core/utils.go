package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// IDStrategy defines how IDs should be generated
type IDStrategy string

// IDGeneratorFunc is a function type for custom ID generation
type IDGeneratorFunc func() string

const (
	// IDStrategyULID uses ULID for IDs (default, best for most use cases)
	// ULIDs are sortable, time-based, and work across restarts
	IDStrategyULID IDStrategy = "ulid"
	// IDStrategyUUID uses UUID v4 for IDs (random, not sortable)
	IDStrategyUUID IDStrategy = "uuid"
	// IDStrategyCustom uses a user-provided custom function
	IDStrategyCustom IDStrategy = "custom"
)

var (
	// Global ID strategy configuration (defaults to ULID)
	currentIDStrategy = IDStrategyULID
	// Entropy source for ULID generation (crypto/rand for security)
	ulidEntropy = ulid.Monotonic(rand.Reader, 0)
	// Custom ID generator function (nil by default)
	customIDGenerator IDGeneratorFunc
)

// SetIDStrategy sets the global ID generation strategy
// This should be called during application initialization
func SetIDStrategy(strategy IDStrategy) {
	currentIDStrategy = strategy
}

// SetCustomIDGenerator sets a custom ID generation function
// This automatically switches the strategy to IDStrategyCustom
func SetCustomIDGenerator(generator IDGeneratorFunc) {
	customIDGenerator = generator
	currentIDStrategy = IDStrategyCustom
}

// GetIDStrategy returns the current ID generation strategy
func GetIDStrategy() IDStrategy {
	return currentIDStrategy
}

// GenerateOTPCode generates a random numeric OTP code of the specified length
func GenerateOTPCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("OTP length must be positive")
	}

	// Calculate the maximum value for the OTP (e.g., 999999 for 6 digits)
	maxValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	maxValue.Sub(maxValue, big.NewInt(1))

	n, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Format with leading zeros
	format := fmt.Sprintf("%%0%dd", length)
	code := fmt.Sprintf(format, n)

	return code, nil
}

// GenerateID generates a unique identifier based on the configured strategy
//
// Default Strategy: ULID (generates sortable, time-based IDs like "01ARZ3NDEKTSV4RRFFQ69G5FAV")
//
// Strategies:
//   - ULID (default): Returns a sortable, time-based ID (26 characters)
//     Best for: Most use cases - sortable, works after restarts, distributed systems
//   - UUID: Returns a UUID v4 like "550e8400-e29b-41d4-a716-446655440000"
//     Best for: When you need standard UUID format
//   - Sequence: Returns sequential numbers like "1", "2", "3" (WARNING: resets on restart)
//     Best for: Testing only - NOT safe for production use
//   - Custom: Uses your own ID generation function
//     Best for: Special requirements (KSUID, nanoid, etc.)
//
// Examples:
//   - Use UUID: core.SetIDStrategy(core.IDStrategyUUID)
//   - Use Sequence (testing only): core.SetIDStrategy(core.IDStrategySequence)
//   - Use Custom: core.SetCustomIDGenerator(yourFunc)
//
// Note: For database-generated IDs, use database defaults instead
// (SERIAL in PostgreSQL, AUTO_INCREMENT in MySQL) and don't call GenerateID()
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
