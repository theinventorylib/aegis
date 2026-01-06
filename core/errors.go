package core

import (
	"errors"
	"fmt"
)

// Common authentication errors used throughout the framework.
// These sentinel errors can be compared using errors.Is() and provide
// consistent error handling across the application.
var (
	// ErrUserNotFound indicates the requested user does not exist
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidCredentials indicates incorrect username/password combination
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserDisabled indicates the user account has been deactivated
	ErrUserDisabled = errors.New("user account is disabled")

	// ErrEmailNotVerified indicates email verification is required
	ErrEmailNotVerified = errors.New("email not verified")

	// ErrInvalidToken indicates a malformed or invalid token
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indicates the token has exceeded its lifetime
	ErrTokenExpired = errors.New("token expired")

	// ErrSessionNotFound indicates the session does not exist
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidSession indicates a malformed or corrupted session
	ErrInvalidSession = errors.New("invalid session")

	// ErrSessionExpired indicates the session has exceeded its lifetime
	ErrSessionExpired = errors.New("session expired")

	// ErrRateLimitExceeded indicates too many requests from this client
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidRequest indicates malformed request data
	ErrInvalidRequest = errors.New("invalid request")

	// ErrUnauthorized indicates authentication is required
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the authenticated user lacks permissions
	ErrForbidden = errors.New("forbidden")

	// ErrInternalServer indicates an unexpected server error
	ErrInternalServer = errors.New("internal server error")

	// ErrDatabaseConnection indicates database connectivity issues
	ErrDatabaseConnection = errors.New("database connection error")

	// ErrRedisConnection indicates Redis connectivity issues
	ErrRedisConnection = errors.New("redis connection error")
)

// ValidationError represents a validation error for a specific field.
// Used when input validation fails for user-provided data.
type ValidationError struct {
	// Field is the name of the field that failed validation
	Field string

	// Message is a human-readable description of what's wrong
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors.
// Useful when validating entire request bodies and returning all
// errors at once rather than failing on the first issue.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors occurred", len(e))
}

// Errors returns all validation errors as a slice for iteration.
func (e ValidationErrors) Errors() []ValidationError {
	return e
}

// AuthError represents an authentication-specific error with additional context.
// It wraps an optional cause error and includes a machine-readable code for
// API responses.
type AuthError struct {
	// Code is a machine-readable error code (e.g., "INVALID_CREDENTIALS")
	Code string

	// Message is a human-readable error description
	Message string

	// Cause is the underlying error that triggered this auth error (optional)
	Cause error
}

func (e AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("auth error [%s]: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("auth error [%s]: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error for error chain unwrapping.
func (e AuthError) Unwrap() error {
	return e.Cause
}

// Predefined auth error codes for API responses.
// These codes provide stable identifiers that clients can programmatically
// handle without parsing error messages.
const (
	// AuthErrorCodeInvalidCredentials indicates wrong username/password
	// #nosec G101
	AuthErrorCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// AuthErrorCodeUserNotFound indicates user does not exist
	AuthErrorCodeUserNotFound = "USER_NOT_FOUND"

	// AuthErrorCodeUserDisabled indicates account is deactivated
	AuthErrorCodeUserDisabled = "USER_DISABLED"

	// AuthErrorCodeAccountNotFound indicates no account for this provider
	AuthErrorCodeAccountNotFound = "ACCOUNT_NOT_FOUND"

	// AuthErrorCodeTokenInvalid indicates malformed token
	AuthErrorCodeTokenInvalid = "TOKEN_INVALID"

	// AuthErrorCodeTokenExpired indicates token lifetime exceeded
	AuthErrorCodeTokenExpired = "TOKEN_EXPIRED"

	// AuthErrorCodeSessionInvalid indicates invalid session
	AuthErrorCodeSessionInvalid = "SESSION_INVALID"

	// AuthErrorCodeRateLimit indicates too many requests
	AuthErrorCodeRateLimit = "RATE_LIMIT"

	// AuthErrorCodeUnauthorized indicates authentication required
	AuthErrorCodeUnauthorized = "UNAUTHORIZED"

	// AuthErrorCodeInternal indicates an unexpected server error
	AuthErrorCodeInternal = "INTERNAL_ERROR"
)

// NewAuthError creates a new AuthError with the given code and message.
func NewAuthError(code, message string) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
	}
}

// NewAuthErrorWithCause creates a new AuthError wrapping an underlying cause.
func NewAuthErrorWithCause(code, message string, cause error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

// IsAuthError checks if an error is an auth error
func IsAuthError(err error) bool {
	var authErr AuthError
	return errors.As(err, &authErr)
}

// GetValidationErrors extracts all validation errors from an error
func GetValidationErrors(err error) ValidationErrors {
	var validationErrs ValidationErrors
	if errors.As(err, &validationErrs) {
		return validationErrs
	}
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return ValidationErrors{validationErr}
	}
	return nil
}
