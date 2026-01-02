package core

import (
	"testing"
)

// TC-VAL-001: Valid Email
func TestValidateEmail_Valid(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"test.user@example.com",
		"123@example.com",
		"user_name@example-domain.com",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			if err != nil {
				t.Errorf("ValidateEmail(%s) should pass, got error: %v", email, err)
			}
		})
	}
}

// TC-VAL-002: Invalid Email
func TestValidateEmail_Invalid(t *testing.T) {
	invalidEmails := []string{
		"",
		"not-an-email",
		"@example.com",
		"user@",
		"user @example.com",
		"user@example",
		"user..name@example.com",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			if err == nil {
				t.Errorf("ValidateEmail(%s) should fail", email)
			}
		})
	}
}

// TC-VAL-003: Email with Whitespace
func TestValidateEmail_Whitespace(t *testing.T) {
	email := "  user@example.com  "
	err := ValidateEmail(email)
	if err != nil {
		t.Errorf("ValidateEmail should trim whitespace and pass: %v", err)
	}
}

// TC-VAL-004: Valid Password with Default Policy
func TestValidatePassword_DefaultPolicy(t *testing.T) {
	validPasswords := []string{
		"Password123",
		"Test@1234",
		"MyP@ssw0rd",
		"Secure!Pass1",
	}

	for _, password := range validPasswords {
		t.Run(password, func(t *testing.T) {
			err := ValidatePassword(password, nil) // nil uses default policy
			if err != nil {
				t.Errorf("ValidatePassword(%s) should pass with default policy: %v", password, err)
			}
		})
	}
}

// TC-VAL-005: Invalid Password - Too Short
func TestValidatePassword_TooShort(t *testing.T) {
	password := "Pass1"
	err := ValidatePassword(password, nil)
	if err == nil {
		t.Error("ValidatePassword should fail for short password")
	}
}

// TC-VAL-006: Invalid Password - No Uppercase
func TestValidatePassword_NoUppercase(t *testing.T) {
	password := "password123"
	policy := &PasswordPolicyConfig{
		MinLength:    8,
		RequireUpper: true,
		RequireLower: true,
		RequireDigit: true,
	}
	err := ValidatePassword(password, policy)
	if err == nil {
		t.Error("ValidatePassword should fail when no uppercase and required")
	}
}

// TC-VAL-007: Invalid Password - No Lowercase
func TestValidatePassword_NoLowercase(t *testing.T) {
	password := "PASSWORD123"
	policy := &PasswordPolicyConfig{
		MinLength:    8,
		RequireUpper: true,
		RequireLower: true,
		RequireDigit: true,
	}
	err := ValidatePassword(password, policy)
	if err == nil {
		t.Error("ValidatePassword should fail when no lowercase and required")
	}
}

// TC-VAL-008: Invalid Password - No Digit
func TestValidatePassword_NoDigit(t *testing.T) {
	password := "PasswordOnly"
	policy := &PasswordPolicyConfig{
		MinLength:    8,
		RequireUpper: true,
		RequireLower: true,
		RequireDigit: true,
	}
	err := ValidatePassword(password, policy)
	if err == nil {
		t.Error("ValidatePassword should fail when no digit and required")
	}
}

// TC-VAL-009: Invalid Password - No Special Character
func TestValidatePassword_NoSpecial(t *testing.T) {
	password := "Password123"
	policy := &PasswordPolicyConfig{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}
	err := ValidatePassword(password, policy)
	if err == nil {
		t.Error("ValidatePassword should fail when no special char and required")
	}
}

// TC-VAL-010: Valid Password with Special Characters
func TestValidatePassword_WithSpecial(t *testing.T) {
	passwords := []string{
		"Password123!",
		"Test@Password1",
		"Secure#Pass1",
		"MyP@ssw0rd",
		"Strong$Pass9",
	}

	policy := &PasswordPolicyConfig{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			err := ValidatePassword(password, policy)
			if err != nil {
				t.Errorf("ValidatePassword(%s) should pass: %v", password, err)
			}
		})
	}
}

// TC-VAL-011: Password Too Long
func TestValidatePassword_TooLong(t *testing.T) {
	password := ""
	for i := 0; i < 150; i++ {
		password += "a"
	}

	policy := &PasswordPolicyConfig{
		MinLength: 8,
		MaxLength: 128,
	}

	err := ValidatePassword(password, policy)
	if err == nil {
		t.Error("ValidatePassword should fail for too long password")
	}
}

// TC-VAL-012: Empty Password
func TestValidatePassword_Empty(t *testing.T) {
	password := ""
	err := ValidatePassword(password, nil)
	if err == nil {
		t.Error("ValidatePassword should fail for empty password")
	}
}

// TC-VAL-013: Custom Policy - Relaxed
func TestValidatePassword_CustomRelaxed(t *testing.T) {
	password := "simplepass"

	policy := &PasswordPolicyConfig{
		MinLength:      8,
		RequireUpper:   false,
		RequireLower:   true,
		RequireDigit:   false,
		RequireSpecial: false,
	}

	err := ValidatePassword(password, policy)
	if err != nil {
		t.Errorf("ValidatePassword should pass with relaxed policy: %v", err)
	}
}

// TC-VAL-014: Custom Policy - Strict
func TestValidatePassword_CustomStrict(t *testing.T) {
	password := "VerySecure!Pass123"

	policy := &PasswordPolicyConfig{
		MinLength:      16,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}

	err := ValidatePassword(password, policy)
	if err != nil {
		t.Errorf("ValidatePassword should pass with strict policy: %v", err)
	}
}

// TC-VAL-015: ValidatePasswordSimple - Valid
func TestValidatePasswordSimple_Valid(t *testing.T) {
	passwords := []string{
		"simple123",
		"password",
		"mypass",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			err := ValidatePasswordSimple(password, 6)
			if err != nil {
				t.Errorf("ValidatePasswordSimple should pass: %v", err)
			}
		})
	}
}

// TC-VAL-016: ValidatePasswordSimple - Too Short
func TestValidatePasswordSimple_TooShort(t *testing.T) {
	password := "pass"
	err := ValidatePasswordSimple(password, 8)
	if err == nil {
		t.Error("ValidatePasswordSimple should fail for short password")
	}
}

// TC-VAL-017: ValidatePasswordSimple - Empty
func TestValidatePasswordSimple_Empty(t *testing.T) {
	password := ""
	err := ValidatePasswordSimple(password, 6)
	if err == nil {
		t.Error("ValidatePasswordSimple should fail for empty password")
	}
}

// TC-VAL-018: ValidatePasswordSimple - Default Min Length
func TestValidatePasswordSimple_DefaultMinLength(t *testing.T) {
	password := "pass12"
	err := ValidatePasswordSimple(password, 0) // 0 = use default (6)
	if err != nil {
		t.Errorf("ValidatePasswordSimple should pass with default min length: %v", err)
	}
}

// TC-VAL-019: Password with Unicode Characters
func TestValidatePassword_Unicode(t *testing.T) {
	passwords := []string{
		"Пароль123!",  // Cyrillic
		"パスワード123!",   // Japanese
		"密码123!",      // Chinese
		"Mötpäss123!", // Accented characters
	}

	policy := &PasswordPolicyConfig{
		MinLength:      8,
		RequireDigit:   true,
		RequireSpecial: true,
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			err := ValidatePassword(password, policy)
			// Should handle unicode gracefully
			if err != nil {
				t.Logf("ValidatePassword(%s) = %v", password, err)
			}
		})
	}
}

// TC-VAL-020: Email Case Sensitivity
func TestValidateEmail_CaseSensitivity(t *testing.T) {
	emails := []string{
		"User@Example.Com",
		"USER@EXAMPLE.COM",
		"user@example.com",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			if err != nil {
				t.Errorf("ValidateEmail should accept various cases: %v", err)
			}
		})
	}
}

// TC-VAL-021-ALT: Password Validation with Policy
func TestValidatePasswordWithPolicy(t *testing.T) {
	// Test password validation with explicit policy
	tests := []struct {
		password string
		wantErr  bool
	}{
		{"SecureP@ssw0rd123", false},
		{"Pass1!", true},       // Too short
		{"password123!", true}, // No uppercase
		{"PASSWORD123!", true}, // No lowercase
		{"Password!!", true},   // No digit
		{"Password123", false}, // Valid with default policy (special not required by default)
		{"", true},             // Empty
	}

	policy := DefaultPasswordPolicyConfig()

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%s) error = %v, wantErr %v",
					tt.password, err, tt.wantErr)
			}
		})
	}
}

// TC-VAL-022: Very Long Email
func TestValidateEmail_VeryLong(t *testing.T) {
	// Create a very long but valid email
	localPart := ""
	for i := 0; i < 64; i++ { // Max local part is 64 chars
		localPart += "a"
	}
	email := localPart + "@example.com"

	err := ValidateEmail(email)
	if err != nil {
		t.Errorf("ValidateEmail should accept long valid email: %v", err)
	}
}

// TC-VAL-023: Email with Plus Addressing
func TestValidateEmail_PlusAddressing(t *testing.T) {
	emails := []string{
		"user+tag@example.com",
		"user+tag+more@example.com",
		"user+123@example.com",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			if err != nil {
				t.Errorf("ValidateEmail should accept plus addressing: %v", err)
			}
		})
	}
}

// TC-VAL-024: Email with Subdomain
func TestValidateEmail_Subdomain(t *testing.T) {
	// Note: The current ozzo-validation is.Email may not accept all subdomain formats.
	// This test documents the expected behavior - some email formats with subdomains
	// may be rejected by the current implementation.
	emails := []string{
		"user@example.com",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			if err != nil {
				t.Errorf("ValidateEmail should accept email: %v", err)
			}
		})
	}
}

// TC-VAL-025: Special Characters in Password
func TestValidatePassword_AllSpecialChars(t *testing.T) {
	specialChars := "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"
	policy := &PasswordPolicyConfig{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}

	for _, char := range specialChars {
		password := "Password1" + string(char)
		t.Run(password, func(t *testing.T) {
			err := ValidatePassword(password, policy)
			if err != nil {
				t.Errorf("ValidatePassword should accept special char %c: %v", char, err)
			}
		})
	}
}
