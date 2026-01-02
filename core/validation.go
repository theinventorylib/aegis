package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Email validation using RFC 5322 simplified pattern
var _ = regexp.MustCompile(EmailRegexPattern)

// ValidateEmail validates an email address format.
//
// Checks:
//   - Email is not empty
//   - Email matches RFC 5322 format (using ozzo-validation)
//
// Returns an error if validation fails. Leading/trailing whitespace is trimmed.
//
// Example:
//
//	if err := core.ValidateEmail(email); err != nil {
//		return fmt.Errorf("invalid email: %w", err)
//	}
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	if err := validation.Validate(email, validation.Required, is.Email); err != nil {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// ValidatePassword validates password strength based on a configurable policy.
//
// The validation checks are controlled by the PasswordPolicyConfig:
//   - MinLength: Minimum character count (default: 8)
//   - MaxLength: Maximum character count (default: 128, prevents DoS)
//   - RequireUpper: At least one uppercase letter A-Z
//   - RequireLower: At least one lowercase letter a-z
//   - RequireDigit: At least one numeric digit 0-9
//   - RequireSpecial: At least one special character (!@#$%^&*, etc.)
//
// If policy is nil, DefaultPasswordPolicyConfig is used (8+ chars, mixed case,
// digit required, special chars optional).
//
// Modern best practices (NIST/OWASP 2024):
//   - Enforce minimum length (8+ characters)
//   - Optionally require character diversity
//   - Check against breached password databases (not implemented here)
//   - Don't force regular password changes
//
// Parameters:
//   - password: The plaintext password to validate
//   - policy: The password policy to enforce (nil = use defaults)
//
// Example:
//
//	policy := &core.PasswordPolicyConfig{
//		MinLength:      12,
//		RequireSpecial: true,
//	}
//	if err := core.ValidatePassword(password, policy); err != nil {
//		return fmt.Errorf("weak password: %w", err)
//	}
func ValidatePassword(password string, policy *PasswordPolicyConfig) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	if policy == nil {
		policy = DefaultPasswordPolicyConfig()
	}

	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}

	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		return fmt.Errorf("password must be at most %d characters long", policy.MaxLength)
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= UppercaseStart && char <= UppercaseEnd:
			hasUpper = true
		case char >= LowercaseStart && char <= LowercaseEnd:
			hasLower = true
		case char >= DigitStart && char <= DigitEnd:
			hasDigit = true
		case (char >= SpecialRange1Start && char <= SpecialRange1End) ||
			(char >= SpecialRange2Start && char <= SpecialRange2End) ||
			(char >= SpecialRange3Start && char <= SpecialRange3End) ||
			(char >= SpecialRange4Start && char <= SpecialRange4End):
			hasSpecial = true
		}
	}

	if policy.RequireUpper && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if policy.RequireLower && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if policy.RequireDigit && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if policy.RequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// ValidatePasswordSimple validates password with basic length requirement only.
//
// This is a simplified validator that only checks minimum length, without
// requiring character diversity (uppercase, lowercase, digits, symbols).
//
// Use this when:
//   - Building low-security applications (internal tools, dev environments)
//   - Users find strict policies too frustrating
//   - You rely on other security measures (MFA, breach detection, etc.)
//
// For production systems with sensitive data, prefer ValidatePassword with
// a proper PasswordPolicyConfig.
//
// Parameters:
//   - password: The plaintext password to validate
//   - minLength: Minimum character count (0 = use default of 6)
//
// Example:
//
//	if err := core.ValidatePasswordSimple(password, 8); err != nil {
//		return err
//	}
func ValidatePasswordSimple(password string, minLength int) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	if minLength == 0 {
		minLength = 6 // default minimum
	}

	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters long", minLength)
	}

	return nil
}

// BindAndValidate decodes a JSON request body and validates it.
// T must implement a Validate() error method.
// This helper ensures consistent validation across all handlers.
//
// Example usage:
//
//	req, err := core.BindAndValidate[CreateOrganizationRequest](r)
//	if err != nil {
//	    core.WriteValidationError(w, err)
//	    return
//	}
func BindAndValidate[T interface{ Validate() error }](r *http.Request) (T, error) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := req.Validate(); err != nil {
		return req, err
	}
	return req, nil
}

// ValidateMiddleware creates a middleware that automatically validates request bodies.
// T must implement a Validate() error method.
// The validated request is passed to the handler, eliminating the need for manual validation.
//
// Example usage:
//
//	router.POST("/organizations", ValidateMiddleware(p.CreateOrganizationHandler))
//
//	func (p *Plugin) CreateOrganizationHandler(
//	    w http.ResponseWriter,
//	    r *http.Request,
//	    req CreateOrganizationRequest,  // Already validated!
//	) {
//	    // Use req directly - validation is guaranteed
//	}
func ValidateMiddleware[T interface{ Validate() error }](
	handler func(w http.ResponseWriter, r *http.Request, req T),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := BindAndValidate[T](r)
		if err != nil {
			GetValidationErrors(err)
			return
		}
		handler(w, r, req)
	}
}
