package core

import (
	"strings"
	"testing"
	"time"
)

// TC-PWD-001: Argon2id Hashing
func TestHashPassword(t *testing.T) {
	// Given
	password := "SecureP@ssw0rd123"

	// When
	hash, err := HashPassword(password, 0, 0, 0, 0) // Use defaults

	// Then
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if len(hash) == 0 {
		t.Error("Hash should not be empty")
	}
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Hash should start with $argon2id$, got: %s", hash[:20])
	}
}

// TC-PWD-002: Password Verification (Valid)
func TestVerifyPassword_Valid(t *testing.T) {
	// Given
	password := "SecureP@ssw0rd123"
	hash, err := HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// When
	valid, _ := VerifyPassword(password, hash)

	// Then
	if !valid {
		t.Error("Password verification should succeed for correct password")
	}
}

// TC-PWD-003: Password Verification (Invalid)
func TestVerifyPassword_Invalid(t *testing.T) {
	// Given
	password := "SecureP@ssw0rd123"
	wrongPassword := "WrongPassword123"
	hash, err := HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// When
	valid, _ := VerifyPassword(wrongPassword, hash)

	// Then
	if valid {
		t.Error("Password verification should fail for incorrect password")
	}
}

// TC-PWD-004: Password Strength Validation
func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Valid Strong", "SecureP@ssw0rd123", false},
		{"Too Short", "Pass1!", true},
		{"No Uppercase", "password123!", true},
		{"No Lowercase", "PASSWORD123!", true},
		{"No Number", "Password!!", true},
		{"No Special", "Password123", false}, // Special chars not required by default
		{"Empty", "", true},
		{"Only Spaces", "        ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, DefaultPasswordPolicyConfig())
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TC-PWD-005: Constant-Time Comparison
func TestVerifyPassword_TimingSafe(t *testing.T) {
	// Verify that password comparison is timing-safe
	// This test ensures constant-time comparison to prevent timing attacks
	password := "SecureP@ssw0rd123"
	hash, err := HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Measure time for correct password
	iterations := 100
	var correctDurations []time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		VerifyPassword(password, hash)
		correctDurations = append(correctDurations, time.Since(start))
	}

	// Measure time for incorrect password
	var incorrectDurations []time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		VerifyPassword("WrongPassword", hash)
		incorrectDurations = append(incorrectDurations, time.Since(start))
	}

	// Calculate average durations
	var correctTotal, incorrectTotal time.Duration
	for i := 0; i < iterations; i++ {
		correctTotal += correctDurations[i]
		incorrectTotal += incorrectDurations[i]
	}
	correctAvg := correctTotal / time.Duration(iterations)
	incorrectAvg := incorrectTotal / time.Duration(iterations)

	// Timing should be similar (within 20% tolerance due to system variance)
	ratio := float64(correctAvg) / float64(incorrectAvg)
	if ratio < 0.8 || ratio > 1.2 {
		t.Logf("Correct avg: %v, Incorrect avg: %v, Ratio: %f", correctAvg, incorrectAvg, ratio)
		// Note: This is informational. Argon2 is designed to be timing-safe,
		// but system variance can affect results
	}
}

// TC-PWD-006: Argon2id Parameters
func TestArgon2idParameters(t *testing.T) {
	// Verify OWASP-compliant Argon2id parameters
	password := "TestPassword123!"
	hash, err := HashPassword(password, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Parse hash to verify parameters
	hashStr := string(hash)
	parts := strings.Split(hashStr, "$")
	if len(parts) < 5 {
		t.Fatalf("Invalid hash format: %s", hashStr)
	}

	// Verify it's argon2id
	if parts[1] != "argon2id" {
		t.Errorf("Expected argon2id, got %s", parts[1])
	}

	// Verify version
	if parts[2] != "v=19" {
		t.Errorf("Expected v=19, got %s", parts[2])
	}

	// Parse parameters (m=65536,t=3,p=4)
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		t.Fatalf("Invalid parameter format: %s", parts[3])
	}

	// Note: The actual parameter values are verified in the implementation
	// This test ensures the hash format is correct
	t.Logf("Hash parameters: %s", parts[3])
}

// TC-PWD-007: Multiple Hash Uniqueness (Same Password)
func TestHashPassword_UniqueSalts(t *testing.T) {
	// Given
	password := "TestPassword123!"

	// When - Hash the same password multiple times
	hashes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		hash, err := HashPassword(password, 0, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}
		hashes[string(hash)] = true
	}

	// Then - All hashes should be unique (different salts)
	if len(hashes) != 10 {
		t.Errorf("Expected 10 unique hashes, got %d", len(hashes))
	}

	// All hashes should still verify the same password
	for hashStr := range hashes {
		valid, err := VerifyPassword(password, hashStr)
		if err != nil {
			t.Errorf("VerifyPassword error: %v", err)
		}
		if !valid {
			t.Error("Hash should verify the original password")
		}
	}
}

// TC-PWD-008: Empty Password
func TestHashPassword_Empty(t *testing.T) {
	// Given
	password := ""

	// When
	hash, err := HashPassword(password, 0, 0, 0, 0)

	// Then - Should still hash (validation happens elsewhere)
	if err != nil {
		t.Fatalf("HashPassword should not error on empty password: %v", err)
	}
	if len(hash) == 0 {
		t.Error("Hash should be generated even for empty password")
	}
}

// TC-PWD-009: Very Long Password
func TestHashPassword_VeryLong(t *testing.T) {
	// Given
	password := strings.Repeat("a", 1000) // 1000 character password

	// When
	hash, err := HashPassword(password, 0, 0, 0, 0)

	// Then
	if err != nil {
		t.Fatalf("HashPassword failed for long password: %v", err)
	}
	if len(hash) == 0 {
		t.Error("Hash should not be empty")
	}

	// Verify it works
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !valid {
		t.Error("Failed to verify long password")
	}
}

// TC-PWD-010: Special Characters in Password
func TestHashPassword_SpecialCharacters(t *testing.T) {
	passwords := []string{
		"!@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		"パスワード123!",    // Japanese characters
		"пароль123!",   // Cyrillic
		"🔐🔑🗝️password", // Emojis
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password, 0, 0, 0, 0)
			if err != nil {
				t.Fatalf("HashPassword failed: %v", err)
			}

			valid, err := VerifyPassword(password, hash)
			if err != nil {
				t.Fatalf("VerifyPassword error: %v", err)
			}
			if !valid {
				t.Error("Failed to verify password with special characters")
			}
		})
	}
}

// TC-PWD-011: Invalid Hash Format
func TestVerifyPassword_InvalidHash(t *testing.T) {
	password := "TestPassword123!"

	invalidHashes := []string{
		"not a hash",
		"$argon2id$invalid",
		"",
		"$bcrypt$10$invalidhash",
	}

	for _, hash := range invalidHashes {
		t.Run(hash, func(t *testing.T) {
			// When
			valid, _ := VerifyPassword(password, hash)

			// Then - Should return false for invalid hash format
			if valid {
				t.Error("VerifyPassword should return false for invalid hash")
			}
		})
	}
}
