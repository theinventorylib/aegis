package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/google/uuid"
)

// IDStrategy defines how IDs should be generated
type IDStrategy string

// IDGeneratorFunc is a function type for custom ID generation
type IDGeneratorFunc func() string

const (
	// IDStrategySequence uses an in-memory sequence (default, simpler for most apps)
	IDStrategySequence IDStrategy = "sequence"
	// IDStrategyUUID uses UUID v4 for IDs (recommended for distributed systems)
	IDStrategyUUID IDStrategy = "uuid"
	// IDStrategyCustom uses a user-provided custom function
	IDStrategyCustom IDStrategy = "custom"
)

var (
	// Global ID strategy configuration (defaults to Sequence)
	currentIDStrategy = IDStrategySequence
	// Sequence counter for IDStrategySequence mode
	sequenceCounter uint64
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
// Default Strategy: Sequence (generates "1", "2", "3"...)
//
// Strategies:
//   - Sequence (default): Returns a sequential number like "1", "2", "3"
//     Best for: Single-instance applications, simpler debugging, smaller IDs
//   - UUID: Returns a UUID v4 like "550e8400-e29b-41d4-a716-446655440000"
//     Best for: Distributed systems, microservices, avoiding ID collisions
//   - Custom: Uses your own ID generation function
//     Best for: Special requirements (ULID, KSUID, nanoid, etc.)
//
// To use UUID: core.SetIDStrategy(core.IDStrategyUUID)
// To use custom: core.SetCustomIDGenerator(yourFunc)
//
// Note: For database-generated sequences, use database defaults instead (SERIAL in PostgreSQL, AUTO_INCREMENT in MySQL)
func GenerateID() string {
	switch currentIDStrategy {
	case IDStrategyUUID:
		return uuid.New().String()
	case IDStrategyCustom:
		if customIDGenerator != nil {
			return customIDGenerator()
		}
		// Fallback to sequence if custom generator is nil
		fallthrough
	case IDStrategySequence:
		fallthrough
	default:
		// Atomic increment for thread-safety
		id := atomic.AddUint64(&sequenceCounter, 1)
		return fmt.Sprintf("%d", id)
	}
}

// GenerateUUID always generates a UUID v4, regardless of strategy
// Use this when you specifically need a UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateSequenceID always generates a sequential ID, regardless of strategy
// Use this when you specifically need a sequence
func GenerateSequenceID() string {
	id := atomic.AddUint64(&sequenceCounter, 1)
	return fmt.Sprintf("%d", id)
}
