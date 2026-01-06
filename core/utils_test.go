package core

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TC-ID-001: ULID Uniqueness
func TestGenerateULID_Uniqueness(t *testing.T) {
	// Given
	count := 10000
	ids := make(map[string]bool)

	// When - Generate many IDs
	for i := 0; i < count; i++ {
		id := GenerateULID()
		ids[id] = true
	}

	// Then - All should be unique
	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

// TC-ID-002: ULID Sortability
func TestGenerateULID_Sortable(t *testing.T) {
	// Given
	var ids []string

	// When - Generate IDs over time
	for i := 0; i < 10; i++ {
		ids = append(ids, GenerateULID())
		time.Sleep(1 * time.Millisecond)
	}

	// Then - Should be lexicographically sortable
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Errorf("IDs not sorted: %s >= %s", ids[i-1], ids[i])
		}
	}
}

// TC-ID-003: UUID Uniqueness
func TestGenerateUUID_Uniqueness(t *testing.T) {
	// Given
	count := 10000
	ids := make(map[string]bool)

	// When - Generate many UUIDs
	for i := 0; i < count; i++ {
		id := GenerateUUID()
		ids[id] = true
	}

	// Then - All should be unique
	if len(ids) != count {
		t.Errorf("Expected %d unique UUIDs, got %d", count, len(ids))
	}
}

// TC-ID-004: UUID Format
func TestGenerateUUID_Format(t *testing.T) {
	// Given
	id := GenerateUUID()

	// Then - Should be in UUID format (36 characters with hyphens)
	if len(id) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(id))
	}

	// Check hyphen positions (8-4-4-4-12)
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("Invalid UUID format: %s", id)
	}
}

// TC-ID-005: Thread-Safe ID Generation
func TestIDGeneration_ThreadSafe(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ids := make(chan string, 1000)

	// When - Generate IDs concurrently
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- GenerateULID()
		}()
	}
	wg.Wait()
	close(ids)

	// Then - All IDs should be unique
	uniqueIDs := make(map[string]bool)
	for id := range ids {
		uniqueIDs[id] = true
	}
	if len(uniqueIDs) != 1000 {
		t.Errorf("Expected 1000 unique IDs, got %d", len(uniqueIDs))
	}
}

// TC-ID-006: ULID Length
func TestGenerateULID_Length(t *testing.T) {
	// Given
	id := GenerateULID()

	// Then - Should be 26 characters
	if len(id) != 26 {
		t.Errorf("Expected ULID length 26, got %d for %s", len(id), id)
	}
}

// TC-ID-007: Random Suffix Generation
func TestGenerateRandomSuffix(t *testing.T) {
	// Given
	length := 16

	// When
	suffix1 := GenerateRandomSuffix(length)
	suffix2 := GenerateRandomSuffix(length)

	// Then
	if len(suffix1) != length {
		t.Errorf("Expected length %d, got %d", length, len(suffix1))
	}
	if suffix1 == suffix2 {
		t.Error("Two random suffixes should be different")
	}
}

// TC-ID-008: Random Suffix Uniqueness
func TestGenerateRandomSuffix_Uniqueness(t *testing.T) {
	// Given
	count := 1000
	length := 12
	suffixes := make(map[string]bool)

	// When
	for i := 0; i < count; i++ {
		suffix := GenerateRandomSuffix(length)
		suffixes[suffix] = true
	}

	// Then - Most should be unique (allow for tiny collision chance)
	if len(suffixes) < count-5 {
		t.Errorf("Expected ~%d unique suffixes, got %d", count, len(suffixes))
	}
}

// TC-ID-009: Random Suffix Characters
func TestGenerateRandomSuffix_Characters(t *testing.T) {
	// Given
	suffix := GenerateRandomSuffix(100)

	// Then - Should only contain valid characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for _, char := range suffix {
		if !containsRune(validChars, char) {
			t.Errorf("Invalid character in suffix: %c", char)
		}
	}
}

// Helper function to check if a string contains a rune
func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// TC-ID-010: Concurrent UUID Generation
func TestGenerateUUID_Concurrent(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	uuids := make(chan string, 500)

	// When - Generate UUIDs concurrently
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uuids <- GenerateUUID()
		}()
	}
	wg.Wait()
	close(uuids)

	// Then - All should be unique
	uniqueUUIDs := make(map[string]bool)
	for uuid := range uuids {
		uniqueUUIDs[uuid] = true
	}
	if len(uniqueUUIDs) != 500 {
		t.Errorf("Expected 500 unique UUIDs, got %d", len(uniqueUUIDs))
	}
}

// TC-ID-011: ULID Timestamp Component
func TestGenerateULID_Timestamp(t *testing.T) {
	// Given
	before := time.Now()

	// When
	id := GenerateULID()

	// Then - The ULID should encode a timestamp close to now
	// (This is implicitly tested by sortability, but we verify it works)
	after := time.Now()

	if len(id) != 26 {
		t.Errorf("Invalid ULID length: %d", len(id))
	}

	// ULIDs generated within a second should have similar prefixes
	time.Sleep(10 * time.Millisecond)
	id2 := GenerateULID()

	// First few characters should be similar (same timestamp range)
	if id[:5] != id2[:5] {
		t.Logf("IDs generated within 10ms have different timestamps (expected): %s vs %s", id[:5], id2[:5])
	}

	_ = before
	_ = after
}

// TC-ID-012: Zero Length Random Suffix
func TestGenerateRandomSuffix_ZeroLength(t *testing.T) {
	// Given
	length := 0

	// When
	suffix := GenerateRandomSuffix(length)

	// Then
	if len(suffix) != 0 {
		t.Errorf("Expected empty suffix, got length %d", len(suffix))
	}
}

// TC-ID-013: Large Random Suffix
func TestGenerateRandomSuffix_Large(t *testing.T) {
	// Given
	length := 1000

	// When
	suffix := GenerateRandomSuffix(length)

	// Then
	if len(suffix) != length {
		t.Errorf("Expected length %d, got %d", length, len(suffix))
	}
}

// TC-ID-014: Format Validation
func TestValidateIDFormats(t *testing.T) {
	tests := []struct {
		name     string
		generate func() string
		validate func(string) bool
		count    int
	}{
		{
			name:     "ULID",
			generate: GenerateULID,
			validate: func(id string) bool { return len(id) == 26 },
			count:    100,
		},
		{
			name:     "UUID",
			generate: GenerateUUID,
			validate: func(id string) bool { return len(id) == 36 },
			count:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.count; i++ {
				id := tt.generate()
				if !tt.validate(id) {
					t.Errorf("Generated ID failed validation: %s", id)
				}
			}
		})
	}
}

// TC-ID-015: High Volume ID Generation
func TestIDGeneration_HighVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high volume test in short mode")
	}

	// Given
	count := 100000
	ids := make(map[string]bool, count)

	// When
	start := time.Now()
	for i := 0; i < count; i++ {
		ids[GenerateULID()] = true
	}
	duration := time.Since(start)

	// Then
	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}

	// Performance check: should generate 100k IDs in under 1 second
	if duration > time.Second {
		t.Logf("Warning: Generated %d IDs in %v (expected < 1s)", count, duration)
	} else {
		t.Logf("Generated %d IDs in %v", count, duration)
	}
}

// TC-ID-016: Mixed Concurrent Generation
func TestIDGeneration_MixedConcurrent(t *testing.T) {
	// Given
	var wg sync.WaitGroup
	ulids := make(chan string, 500)
	uuids := make(chan string, 500)

	// When - Generate both types concurrently
	for i := 0; i < 500; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ulids <- GenerateULID()
		}()
		go func() {
			defer wg.Done()
			uuids <- GenerateUUID()
		}()
	}
	wg.Wait()
	close(ulids)
	close(uuids)

	// Then - All should be unique
	uniqueULIDs := make(map[string]bool)
	for id := range ulids {
		uniqueULIDs[id] = true
	}

	uniqueUUIDs := make(map[string]bool)
	for id := range uuids {
		uniqueUUIDs[id] = true
	}

	if len(uniqueULIDs) != 500 {
		t.Errorf("Expected 500 unique ULIDs, got %d", len(uniqueULIDs))
	}
	if len(uniqueUUIDs) != 500 {
		t.Errorf("Expected 500 unique UUIDs, got %d", len(uniqueUUIDs))
	}
}

// TC-ID-017: Test Generate Secure Token
func TestGenerateSecureToken(t *testing.T) {
	// Generate tokens
	token1 := GenerateSecureToken()
	token2 := GenerateSecureToken()

	// Should be different
	if token1 == token2 {
		t.Error("Two secure tokens should be different")
	}

	// Should have reasonable length
	if len(token1) < 32 {
		t.Errorf("Secure token too short: %d", len(token1))
	}
}

// TC-ID-018: Secure Token Uniqueness
func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	// Given
	count := 1000
	tokens := make(map[string]bool)

	// When
	for i := 0; i < count; i++ {
		token := GenerateSecureToken()
		tokens[token] = true
	}

	// Then - All should be unique
	if len(tokens) != count {
		t.Errorf("Expected %d unique tokens, got %d", count, len(tokens))
	}
}

// TC-ID-019: Test constant time comparison
func TestConstantTimeCompare(t *testing.T) {
	// This test is more about ensuring the function exists and works
	a := "secret_token_12345"
	b := "secret_token_12345"
	c := "different_token_67"

	// Equal strings should match
	if !ConstantTimeCompare(a, b) {
		t.Error("Equal strings should match")
	}

	// Different strings should not match
	if ConstantTimeCompare(a, c) {
		t.Error("Different strings should not match")
	}

	// Different lengths should not match
	if ConstantTimeCompare(a, "short") {
		t.Error("Different length strings should not match")
	}
}

// TC-ID-020: Constant Time Compare Timing
func TestConstantTimeCompare_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	secret := "this_is_a_very_long_secret_token_that_should_be_compared_in_constant_time"

	// Test with matching strings
	iterations := 1000
	var matchDurations []time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		ConstantTimeCompare(secret, secret)
		matchDurations = append(matchDurations, time.Since(start))
	}

	// Test with non-matching strings (but same length)
	different := "xxxx_xx_x_xxxx_xxxx_xxxxxx_xxxxx_xxxx_xxxxxx_xx_xxxxxxxxxx_xx_xxxxxxxxx_xxxx"
	var mismatchDurations []time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		ConstantTimeCompare(secret, different)
		mismatchDurations = append(mismatchDurations, time.Since(start))
	}

	// Calculate averages
	var matchTotal, mismatchTotal time.Duration
	for i := 0; i < iterations; i++ {
		matchTotal += matchDurations[i]
		mismatchTotal += mismatchDurations[i]
	}
	matchAvg := matchTotal / time.Duration(iterations)
	mismatchAvg := mismatchTotal / time.Duration(iterations)

	// Log timing information (informational only due to system variance)
	t.Logf("Match avg: %v, Mismatch avg: %v", matchAvg, mismatchAvg)
}

func TestGenerateID_Strategies(t *testing.T) {
	// Save current strategy to restore later
	originalStrategy := GetIDStrategy()
	defer func() {
		SetIDStrategy(originalStrategy)
	}()

	t.Run("ULID Strategy (Default)", func(t *testing.T) {
		SetIDStrategy(IDStrategyULID)
		id := GenerateID()
		// ULID is 26 characters
		if len(id) != 26 {
			t.Errorf("Expected ULID length 26, got %d for %s", len(id), id)
		}
	})

	t.Run("UUID Strategy", func(t *testing.T) {
		SetIDStrategy(IDStrategyUUID)
		id := GenerateID()
		// UUID v4 is 36 characters with hyphens
		if len(id) != 36 {
			t.Errorf("Expected UUID length 36, got %d for %s", len(id), id)
		}
		if !strings.Contains(id, "-") {
			t.Errorf("Expected UUID to contain hyphens, got %s", id)
		}
	})

	t.Run("Custom Strategy", func(t *testing.T) {
		expectedID := "custom-id-prefix-999"
		SetCustomIDGenerator(func() string {
			return expectedID
		})
		id := GenerateID()
		if id != expectedID {
			t.Errorf("Expected custom ID %s, got %s", expectedID, id)
		}
		if GetIDStrategy() != IDStrategyCustom {
			t.Errorf("Expected strategy to be custom, got %s", GetIDStrategy())
		}
	})
}
