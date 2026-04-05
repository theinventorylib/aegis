package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Test constants for commonly used strings (goconst)
const (
	ratelimitTestIP    = "192.168.1.100"
	ratelimitTestEmail = "user@example.com"
)

// TC-RL-001: Rate Limiter Creation
func TestNewRateLimiter(t *testing.T) {
	// Given
	config := DefaultRateLimitConfig()

	// When
	limiter := NewRateLimiter(config, nil, nil)

	// Then
	if limiter == nil {
		t.Fatal("NewRateLimiter should return a non-nil limiter")
	}

	// Cleanup
	limiter.Stop()
}

// TC-RL-002: Rate Limiter with Default Config
func TestNewRateLimiter_DefaultConfig(t *testing.T) {
	// When - nil config should use defaults
	limiter := NewRateLimiter(nil, nil, nil)

	// Then
	if limiter == nil {
		t.Fatal("NewRateLimiter should return a non-nil limiter with nil config")
	}

	// Cleanup
	limiter.Stop()
}

// TC-RL-003: Rate Limit Allow - Within Limit
func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	// Given
	config := &RateLimitConfig{
		RequestsPerWindow: 3,
		WindowDuration:    1 * time.Minute,
		KeyPrefix:         "test:",
		ByIP:              true,
	}
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()

	ctx := context.Background()
	key := ratelimitTestIP

	// When - Attempt 3 times (should succeed)
	for i := 0; i < 3; i++ {
		allowed, remaining, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("Attempt %d should be allowed", i+1)
		}
		expectedRemaining := 3 - (i + 1)
		if remaining != expectedRemaining {
			t.Errorf("Expected remaining %d, got %d", expectedRemaining, remaining)
		}
	}
}

// TC-RL-004: Rate Limit Block - Exceeded Limit
func TestRateLimiter_Allow_ExceedLimit(t *testing.T) {
	// Given
	config := &RateLimitConfig{
		RequestsPerWindow: 3,
		WindowDuration:    1 * time.Minute,
		KeyPrefix:         "test:",
		ByIP:              true,
	}
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()

	ctx := context.Background()
	key := ratelimitTestIP

	// Exhaust the limit
	for i := 0; i < 3; i++ {
		_, _, _ = limiter.Allow(ctx, key)
	}

	// When - 4th attempt should be blocked
	allowed, remaining, err := limiter.Allow(ctx, key)

	// Then
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("4th attempt should be blocked")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", remaining)
	}
}

// TC-RL-005: Rate Limit Different Keys
func TestRateLimiter_Allow_DifferentKeys(t *testing.T) {
	// Given
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowDuration:    1 * time.Minute,
		KeyPrefix:         "test:",
		ByIP:              true,
	}
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()

	ctx := context.Background()
	key1 := "192.168.1.100"
	key2 := "192.168.1.101"

	// When - Use both keys
	_, _, _ = limiter.Allow(ctx, key1)
	_, _, _ = limiter.Allow(ctx, key1)

	// Then - key1 should be exhausted
	allowed1, _, _ := limiter.Allow(ctx, key1)
	if allowed1 {
		t.Error("key1 should be blocked")
	}

	// key2 should still have quota
	allowed2, _, _ := limiter.Allow(ctx, key2)
	if !allowed2 {
		t.Error("key2 should be allowed")
	}
}

// TC-RL-006: Rate Limit Window Reset (Short Window)
func TestRateLimiter_Allow_WindowReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limit window reset test in short mode")
	}

	// Given - Very short window for testing
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowDuration:    100 * time.Millisecond,
		KeyPrefix:         "test:",
		ByIP:              true,
	}
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()

	ctx := context.Background()
	key := ratelimitTestIP

	// Exhaust limit
	_, _, _ = limiter.Allow(ctx, key)
	_, _, _ = limiter.Allow(ctx, key)

	// Verify blocked
	allowed, _, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Error("Should be blocked after exhausting limit")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// When - Attempt again
	allowed, remaining, err := limiter.Allow(ctx, key)

	// Then - Should be allowed
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("Should be allowed after window reset")
	}
	if remaining != 1 {
		t.Errorf("Expected remaining 1, got %d", remaining)
	}
}

// TC-RL-007: Concurrent Rate Limiting
func TestRateLimiter_Allow_Concurrent(t *testing.T) {
	// Given
	config := &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowDuration:    1 * time.Minute,
		KeyPrefix:         "test:",
		ByIP:              true,
	}
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()

	ctx := context.Background()
	key := ratelimitTestIP

	// When - Make concurrent requests
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _, _ := limiter.Allow(ctx, key)
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Then - Should have approximately 100 allowed
	if allowedCount > 100 {
		t.Errorf("Expected at most 100 allowed, got %d", allowedCount)
	}
	if allowedCount < 90 { // Allow some variance due to race conditions
		t.Errorf("Expected at least 90 allowed, got %d", allowedCount)
	}
}

// TC-RL-008: Default Rate Limit Config
func TestDefaultRateLimitConfig(t *testing.T) {
	// When
	config := DefaultRateLimitConfig()

	// Then
	if config == nil {
		t.Fatal("DefaultRateLimitConfig should return non-nil config")
		return
	}

	if config.RequestsPerWindow != DefaultRateLimitRequests {
		t.Errorf("Expected %d requests, got %d", DefaultRateLimitRequests, config.RequestsPerWindow)
	}

	if config.WindowDuration != DefaultRateLimitWindow {
		t.Errorf("Expected %v window, got %v", DefaultRateLimitWindow, config.WindowDuration)
	}

	if config.KeyPrefix != DefaultRateLimitKeyPrefix {
		t.Errorf("Expected prefix %s, got %s", DefaultRateLimitKeyPrefix, config.KeyPrefix)
	}

	if !config.ByIP {
		t.Error("ByIP should be true by default")
	}

	if config.ByUser {
		t.Error("ByUser should be false by default")
	}
}

// TC-RL-009: Auth Rate Limit Config
func TestAuthRateLimitConfig(t *testing.T) {
	// When
	config := AuthRateLimitConfig()

	// Then
	if config == nil {
		t.Fatal("AuthRateLimitConfig should return non-nil config")
		return
	}

	if config.RequestsPerWindow != AuthRateLimitRequests {
		t.Errorf("Expected %d requests, got %d", AuthRateLimitRequests, config.RequestsPerWindow)
	}

	// Auth rate limit should be stricter than default
	defaultConfig := DefaultRateLimitConfig()
	if config.RequestsPerWindow >= defaultConfig.RequestsPerWindow {
		t.Error("Auth rate limit should be stricter than default")
	}
}

// TC-RL-010: Rate Limiter Stop
func TestRateLimiter_Stop(_ *testing.T) {
	// Given
	config := DefaultRateLimitConfig()
	limiter := NewRateLimiter(config, nil, nil)

	// When - Stop should not panic
	limiter.Stop()

	// Then - The cleanup goroutine should have been stopped
	// (We can't easily verify this externally, so we just check
	// that Stop() completed without panicking)
}

// TC-LAT-001: Login Attempt Tracker Creation
func TestNewLoginAttemptTracker(t *testing.T) {
	// Given
	config := DefaultLoginAttemptConfig()

	// When
	tracker := NewLoginAttemptTracker(config, nil)

	// Then
	if tracker == nil {
		t.Fatal("NewLoginAttemptTracker should return a non-nil tracker")
	}

	// Cleanup
	tracker.Stop()
}

// TC-LAT-002: Login Attempt Tracker Default Config
func TestNewLoginAttemptTracker_DefaultConfig(t *testing.T) {
	// When - nil config should use defaults
	tracker := NewLoginAttemptTracker(nil, nil)

	// Then
	if tracker == nil {
		t.Fatal("NewLoginAttemptTracker should return a non-nil tracker with nil config")
	}

	// Cleanup
	tracker.Stop()
}

// TC-LAT-003: Record Failed Attempt
func TestLoginAttemptTracker_RecordFailedAttempt(t *testing.T) {
	// Given
	config := &LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: 1 * time.Minute,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	identifier := ratelimitTestEmail

	// When - Record first attempt
	attempts, lockedOut, err := tracker.RecordFailedAttempt(ctx, identifier)

	// Then
	if err != nil {
		t.Fatalf("RecordFailedAttempt failed: %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if lockedOut {
		t.Error("Should not be locked out after first attempt")
	}
}

// TC-LAT-004: Lockout After Max Attempts
func TestLoginAttemptTracker_Lockout(t *testing.T) {
	// Given
	config := &LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: 1 * time.Minute,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	identifier := ratelimitTestEmail

	// When - Record max attempts
	for i := 0; i < 3; i++ {
		_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)
	}

	// Then - Should be locked out
	locked, _, err := tracker.IsLockedOut(ctx, identifier)
	if err != nil {
		t.Fatalf("IsLockedOut failed: %v", err)
	}
	if !locked {
		t.Error("Should be locked out after max attempts")
	}
}

// TC-LAT-005: Clear Attempts on Success
func TestLoginAttemptTracker_ClearAttempts(t *testing.T) {
	// Given
	config := &LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: 1 * time.Minute,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	identifier := ratelimitTestEmail

	// Record some attempts
	_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)
	_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)

	// When - Clear attempts
	err := tracker.ClearAttempts(ctx, identifier)

	// Then
	if err != nil {
		t.Fatalf("ClearAttempts failed: %v", err)
	}

	// Should not be locked out
	locked, _, _ := tracker.IsLockedOut(ctx, identifier)
	if locked {
		t.Error("Should not be locked out after clearing attempts")
	}

	// First new attempt should be counted as 1
	attempts, _, _ := tracker.RecordFailedAttempt(ctx, identifier)
	if attempts != 1 {
		t.Errorf("Expected 1 attempt after clear, got %d", attempts)
	}
}

// TC-LAT-006: Different Identifiers
func TestLoginAttemptTracker_DifferentIdentifiers(t *testing.T) {
	// Given
	config := &LoginAttemptConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Minute,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	user1 := "user1@example.com"
	user2 := "user2@example.com"

	// Lock out user1
	_, _, _ = tracker.RecordFailedAttempt(ctx, user1)
	_, _, _ = tracker.RecordFailedAttempt(ctx, user1)

	// Then - user1 should be locked, user2 should not
	locked1, _, _ := tracker.IsLockedOut(ctx, user1)
	locked2, _, _ := tracker.IsLockedOut(ctx, user2)

	if !locked1 {
		t.Error("user1 should be locked out")
	}
	if locked2 {
		t.Error("user2 should not be locked out")
	}
}

// TC-LAT-007: Lockout Duration
func TestLoginAttemptTracker_LockoutDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping lockout duration test in short mode")
	}

	// Given - Short lockout for testing
	config := &LoginAttemptConfig{
		MaxAttempts:     2,
		LockoutDuration: 100 * time.Millisecond,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	identifier := ratelimitTestEmail

	// Lock out
	_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)
	_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)

	// Verify locked
	locked, _, _ := tracker.IsLockedOut(ctx, identifier)
	if !locked {
		t.Error("Should be locked out")
	}

	// Wait for lockout to expire
	time.Sleep(150 * time.Millisecond)

	// When - Check again
	locked, _, _ = tracker.IsLockedOut(ctx, identifier)

	// Then - Should not be locked out anymore
	if locked {
		t.Error("Lockout should have expired")
	}
}

// TC-LAT-008: Default Login Attempt Config
func TestDefaultLoginAttemptConfig(t *testing.T) {
	// When
	config := DefaultLoginAttemptConfig()

	// Then
	if config == nil {
		t.Fatal("DefaultLoginAttemptConfig should return non-nil config")
		return
	}

	if config.MaxAttempts != DefaultMaxLoginAttempts {
		t.Errorf("Expected %d max attempts, got %d", DefaultMaxLoginAttempts, config.MaxAttempts)
	}

	if config.LockoutDuration != DefaultLoginLockoutDuration {
		t.Errorf("Expected %v lockout duration, got %v", DefaultLoginLockoutDuration, config.LockoutDuration)
	}

	if config.AttemptWindow != DefaultLoginAttemptWindow {
		t.Errorf("Expected %v attempt window, got %v", DefaultLoginAttemptWindow, config.AttemptWindow)
	}
}

// TC-LAT-009: Concurrent Login Attempts
func TestLoginAttemptTracker_Concurrent(t *testing.T) {
	// Given
	config := &LoginAttemptConfig{
		MaxAttempts:     50,
		LockoutDuration: 1 * time.Minute,
		AttemptWindow:   1 * time.Minute,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()

	ctx := context.Background()
	identifier := ratelimitTestEmail

	// When - Make concurrent attempts
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = tracker.RecordFailedAttempt(ctx, identifier)
		}()
	}
	wg.Wait()

	// Then - Should be locked out
	locked, _, _ := tracker.IsLockedOut(ctx, identifier)
	if !locked {
		t.Error("Should be locked out after concurrent attempts exceed max")
	}
}

// TC-LAT-010: Login Attempt Tracker Stop
func TestLoginAttemptTracker_Stop(_ *testing.T) {
	// Given
	config := DefaultLoginAttemptConfig()
	tracker := NewLoginAttemptTracker(config, nil)

	// When - Stop should not panic
	tracker.Stop()

	// Then - The cleanup goroutine should have been stopped
	// (We can't easily verify this externally, so we just check
	// that Stop() completed without panicking)
}
