package core

// Verification token types used by the account-security flows. These are the
// values persisted in verification.type: keep them stable, because tokens
// issued under one spelling cannot be validated under another.
const (
	// VerificationTypeEmailVerification is used by email-verification OTPs.
	VerificationTypeEmailVerification = "email_verification"

	// VerificationTypePasswordReset is used by password-reset tokens.
	VerificationTypePasswordReset = "password_reset"

	// VerificationTypeEmailChange is used by email-change confirmation tokens.
	VerificationTypeEmailChange = "email_change"
)
