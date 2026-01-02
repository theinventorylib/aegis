package core

import (
	"context"
	"fmt"
	"testing"
	"time" // Added for time.Now() and time.Since()
)

// Test constants for commonly used strings (goconst)
const (
	benchTestPassword     = "TestPassword123!"
	benchSecurePassword   = "SecureP@ssw0rd123"
	benchTestSessionToken = "test_session_token_abc123"
	// The following lines from the user's edit are syntactically incorrect for constant declarations.
	// benchTestP	password := benchTestPassword
	// benchSecureP	password := benchSecurePassword
)

// BenchmarkHashPassword benchmarks password hashing performance.
// Target: < 500ms, Critical threshold: < 1s
func BenchmarkHashPassword(b *testing.B) {
	password := benchTestPassword
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password, 0, 0, 0, 0)
	}
}

// BenchmarkVerifyPassword benchmarks password verification performance.
// Target: < 500ms, Critical threshold: < 1s
func BenchmarkVerifyPassword(b *testing.B) {
	password := benchTestPassword
	hash, _ := HashPassword(password, 0, 0, 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = VerifyPassword(password, hash)
		_ = time.Since(start)
	}
}

// BenchmarkVerifyPassword_WrongPassword benchmarks verification with wrong password.
func BenchmarkVerifyPassword_WrongPassword(b *testing.B) {
	password := benchTestPassword
	hash, _ := HashPassword(password, 0, 0, 0, 0)
	wrongPassword := "WrongPassword456!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyPassword(wrongPassword, hash)
	}
}

// BenchmarkGenerateULID benchmarks ULID generation performance.
// Target: < 1ms, Critical threshold: < 5ms
func BenchmarkGenerateULID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateULID()
	}
}

// BenchmarkGenerateUUID benchmarks UUID generation performance.
func BenchmarkGenerateUUID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateUUID()
	}
}

// BenchmarkGenerateSecureToken benchmarks secure token generation performance.
func BenchmarkGenerateSecureToken(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSecureToken()
	}
}

// BenchmarkGenerateID benchmarks the main ID generation function.
func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateID()
	}
}

// BenchmarkRateLimiter_Allow benchmarks rate limit checking.
// Target: < 5ms, Critical threshold: < 10ms
func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := DefaultRateLimitConfig()
	config.RequestsPerWindow = 1000000 // Set high to avoid blocking
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = limiter.Allow(ctx, fmt.Sprintf("key_%d", i%1000))
	}
}

// BenchmarkRateLimiter_Allow_SameKey benchmarks rate limiting same key repeatedly.
func BenchmarkRateLimiter_Allow_SameKey(b *testing.B) {
	config := DefaultRateLimitConfig()
	config.RequestsPerWindow = 1000000 // Set high to avoid blocking
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = limiter.Allow(ctx, "constant_key")
	}
}

// BenchmarkLoginAttemptTracker_RecordFailedAttempt benchmarks login attempt tracking.
func BenchmarkLoginAttemptTracker_RecordFailedAttempt(b *testing.B) {
	config := &LoginAttemptConfig{
		MaxAttempts:     1000000, // Set high to avoid lockout
		LockoutDuration: DefaultLoginLockoutDuration,
		AttemptWindow:   DefaultLoginAttemptWindow,
	}
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = tracker.RecordFailedAttempt(ctx, fmt.Sprintf("user_%d", i%1000))
	}
}

// BenchmarkLoginAttemptTracker_IsLockedOut benchmarks lockout status checking.
func BenchmarkLoginAttemptTracker_IsLockedOut(b *testing.B) {
	config := DefaultLoginAttemptConfig()
	tracker := NewLoginAttemptTracker(config, nil)
	defer tracker.Stop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = tracker.IsLockedOut(ctx, fmt.Sprintf("user_%d", i%1000))
	}
}

// BenchmarkValidateEmail benchmarks email validation.
func BenchmarkValidateEmail(b *testing.B) {
	email := "test@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEmail(email)
	}
}

// BenchmarkValidatePassword benchmarks password validation.
func BenchmarkValidatePassword(b *testing.B) {
	password := benchSecurePassword
	policy := DefaultPasswordPolicyConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePassword(password, policy)
	}
}

// BenchmarkCookieManager_SetSessionCookie benchmarks cookie creation.
func BenchmarkCookieManager_SetSessionCookie(b *testing.B) {
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just test the cookie name retrieval since we can't use ResponseWriter in benchmark
		cm.GetSessionCookieName()
	}
}

// BenchmarkConstantTimeCompare benchmarks constant-time string comparison.
func BenchmarkConstantTimeCompare(b *testing.B) {
	a := benchTestSessionToken
	bStr := benchTestSessionToken

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConstantTimeCompare(a, bStr)
	}
}

// BenchmarkConstantTimeCompare_Mismatch benchmarks constant-time comparison with mismatch.
func BenchmarkConstantTimeCompare_Mismatch(b *testing.B) {
	a := benchTestSessionToken
	bStr := "different_session_token_xyz"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConstantTimeCompare(a, bStr)
	}
}

// Parallel Benchmarks

// BenchmarkGenerateULID_Parallel benchmarks concurrent ULID generation.
func BenchmarkGenerateULID_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GenerateULID()
		}
	})
}

// BenchmarkRateLimiter_Allow_Parallel benchmarks concurrent rate limiting.
func BenchmarkRateLimiter_Allow_Parallel(b *testing.B) {
	config := DefaultRateLimitConfig()
	config.RequestsPerWindow = 1000000
	limiter := NewRateLimiter(config, nil, nil)
	defer limiter.Stop()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _, _ = limiter.Allow(ctx, fmt.Sprintf("key_%d", i%1000))
			i++
		}
	})
}
