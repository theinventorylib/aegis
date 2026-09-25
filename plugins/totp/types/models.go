// Package types defines the domain models and store contract for the TOTP plugin.
package types

// Credential is a user's TOTP credential state.
//
// Secret is empty until enrollment starts. Enabled is only true after a code
// generated from the pending secret has been confirmed, so a half-finished
// setup can never lock a user out.
type Credential struct {
	UserID  string
	Secret  string
	Enabled bool
}

// ========== HTTP DTOs ==========

// SetupResponse is returned by the enrollment endpoint. The secret is shown
// once; TOTP is inactive until Enable confirms a code.
type SetupResponse struct {
	Secret string `json:"secret"`
	URL    string `json:"otpauth_url"`
}

// EnableRequest confirms the pending secret.
type EnableRequest struct {
	Code string `json:"code"`
}

// EnableResponse carries the single-use recovery codes. Also used by the
// regenerate endpoint.
type EnableResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

// VerifyRequest steps up the current session with a TOTP or recovery code.
type VerifyRequest struct {
	Code     string `json:"code"`
	Recovery bool   `json:"recovery"`
}

// DisableRequest confirms the second factor before removing it.
type DisableRequest struct {
	Code string `json:"code"`
}

// StatusResponse reports enrollment and current-session state.
type StatusResponse struct {
	Enabled  bool `json:"enabled"`
	Verified bool `json:"verified"`
}

// RegenerateRequest asks for fresh recovery codes.
type RegenerateRequest struct {
	Code string `json:"code"`
}
