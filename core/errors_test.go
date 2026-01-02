package core

import (
	"errors"
	"testing"
)

// TC-ERR-001: AuthError Creation
func TestAuthError_Creation(t *testing.T) {
	// Given
	code := "INVALID_CREDENTIALS"
	message := "Invalid email or password"

	// When
	err := NewAuthError(code, message)

	// Then
	if err.Code != code {
		t.Errorf("Expected code %s, got %s", code, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %s, got %s", message, err.Message)
	}
	// Error() returns formatted string with code
	expectedError := "auth error [INVALID_CREDENTIALS]: Invalid email or password"
	if err.Error() != expectedError {
		t.Errorf("Expected Error() to return %s, got %s", expectedError, err.Error())
	}
}

// TC-ERR-002: Error Wrapping
func TestAuthError_Wrapping(t *testing.T) {
	// Given
	originalErr := errors.New("database connection failed")

	// When
	wrapped := NewAuthErrorWithCause("INTERNAL_ERROR", "Internal server error", originalErr)

	// Then
	if !errors.Is(wrapped, originalErr) {
		t.Error("errors.Is should recognize wrapped error")
	}

	var authErr *AuthError
	if !errors.As(wrapped, &authErr) {
		t.Error("errors.As should extract AuthError")
	}

	if authErr.Cause != originalErr {
		t.Error("Cause should be set to original error")
	}
}

// TC-ERR-003: Error Unwrapping
func TestAuthError_Unwrap(t *testing.T) {
	// Given
	originalErr := errors.New("underlying error")
	authErr := NewAuthErrorWithCause("TEST_ERROR", "Test error", originalErr)

	// When
	unwrapped := errors.Unwrap(authErr)

	// Then
	if unwrapped != originalErr {
		t.Error("Unwrap should return the original error")
	}
}

// TC-ERR-004: Sentinel Errors
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrUserNotFound", ErrUserNotFound},
		{"ErrInvalidCredentials", ErrInvalidCredentials},
		{"ErrUserDisabled", ErrUserDisabled},
		{"ErrEmailNotVerified", ErrEmailNotVerified},
		{"ErrInvalidToken", ErrInvalidToken},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrSessionNotFound", ErrSessionNotFound},
		{"ErrInvalidSession", ErrInvalidSession},
		{"ErrSessionExpired", ErrSessionExpired},
		{"ErrRateLimitExceeded", ErrRateLimitExceeded},
		{"ErrInvalidRequest", ErrInvalidRequest},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrForbidden", ErrForbidden},
		{"ErrInternalServer", ErrInternalServer},
		{"ErrDatabaseConnection", ErrDatabaseConnection},
		{"ErrRedisConnection", ErrRedisConnection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have an error message", tt.name)
			}
		})
	}
}

// TC-ERR-005: ValidationError
func TestValidationError(t *testing.T) {
	// Given
	field := "email"
	message := "Email is required"

	// When
	err := ValidationError{
		Field:   field,
		Message: message,
	}

	// Then
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}
	// Should contain both field and message
	if !contains(errorMsg, field) {
		t.Errorf("Error message should contain field name: %s", errorMsg)
	}
	if !contains(errorMsg, message) {
		t.Errorf("Error message should contain message: %s", errorMsg)
	}
}

// TC-ERR-006: ValidationErrors Multiple
func TestValidationErrors_Multiple(t *testing.T) {
	// Given
	errs := ValidationErrors{
		{Field: "email", Message: "Email is required"},
		{Field: "password", Message: "Password must be at least 8 characters"},
	}

	// When
	errorMsg := errs.Error()

	// Then
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}

	// Should indicate multiple errors
	errors := errs.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}

// TC-ERR-007: ValidationErrors Single
func TestValidationErrors_Single(t *testing.T) {
	// Given
	errs := ValidationErrors{
		{Field: "email", Message: "Email is required"},
	}

	// When
	errorMsg := errs.Error()

	// Then
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}

	errors := errs.Errors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
}

// TC-ERR-008: ValidationErrors Empty
func TestValidationErrors_Empty(t *testing.T) {
	// Given
	errs := ValidationErrors{}

	// When
	errorMsg := errs.Error()

	// Then
	if errorMsg == "" {
		t.Error("Error message should not be empty even for no errors")
	}

	errors := errs.Errors()
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

// TC-ERR-009: Error Is Comparison
func TestError_IsComparison(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "Same sentinel error",
			err:      ErrUserNotFound,
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "Different sentinel error",
			err:      ErrUserNotFound,
			target:   ErrInvalidCredentials,
			expected: false,
		},
		{
			name:     "Wrapped sentinel error",
			err:      NewAuthErrorWithCause("NOT_FOUND", "User not found", ErrUserNotFound),
			target:   ErrUserNotFound,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TC-ERR-010: Error As Type Assertion
func TestError_AsTypeAssertion(t *testing.T) {
	// Given
	authErr := NewAuthError("TEST_CODE", "Test message")

	// When
	var target *AuthError
	success := errors.As(authErr, &target)

	// Then
	if !success {
		t.Error("errors.As should succeed for AuthError")
	}
	if target.Code != "TEST_CODE" {
		t.Errorf("Expected code TEST_CODE, got %s", target.Code)
	}
}

// TC-ERR-011: Error As Failed
func TestError_AsFailed(t *testing.T) {
	// Given
	standardErr := errors.New("standard error")

	// When
	var target *AuthError
	success := errors.As(standardErr, &target)

	// Then
	if success {
		t.Error("errors.As should fail for non-AuthError")
	}
}

// TC-ERR-012: Nil Cause
func TestAuthError_NilCause(t *testing.T) {
	// Given
	authErr := NewAuthError("TEST_ERROR", "Test message")

	// When
	unwrapped := errors.Unwrap(authErr)

	// Then
	if unwrapped != nil {
		t.Error("Unwrap should return nil when cause is nil")
	}
}

// TC-ERR-013: Error Chain
func TestAuthError_Chain(t *testing.T) {
	// Given
	rootErr := errors.New("root cause")
	midErr := NewAuthErrorWithCause("MID_ERROR", "Middle error", rootErr)
	topErr := NewAuthErrorWithCause("TOP_ERROR", "Top error", midErr)

	// When & Then - Should be able to unwrap through chain
	if !errors.Is(topErr, rootErr) {
		t.Error("Should find root error in chain")
	}
	if !errors.Is(topErr, midErr) {
		t.Error("Should find middle error in chain")
	}
}

// TC-ERR-014: Error Code Retrieval
func TestAuthError_CodeRetrieval(t *testing.T) {
	tests := []struct {
		name string
		err  *AuthError
		code string
	}{
		{
			name: "Simple code",
			err:  NewAuthError("SIMPLE", "Simple error"),
			code: "SIMPLE",
		},
		{
			name: "Code with underscores",
			err:  NewAuthError("INVALID_CREDENTIALS", "Invalid credentials"),
			code: "INVALID_CREDENTIALS",
		},
		{
			name: "Empty code",
			err:  NewAuthError("", "No code error"),
			code: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, tt.err.Code)
			}
		})
	}
}

// TC-ERR-015: Error Message Retrieval
func TestAuthError_MessageRetrieval(t *testing.T) {
	// Given
	message := "This is a test error message"
	err := NewAuthError("TEST", message)

	// When - Check Message field, not Error() which is formatted
	retrieved := err.Message

	// Then
	if retrieved != message {
		t.Errorf("Expected message %s, got %s", message, retrieved)
	}
}

// TC-ERR-016: Cause Preservation
func TestAuthError_CausePreservation(t *testing.T) {
	// Given
	cause := errors.New("original cause")
	err := NewAuthErrorWithCause("TEST", "Test error", cause)

	// When
	retrievedCause := err.Cause

	// Then
	if retrievedCause != cause {
		t.Error("Cause should be preserved")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
