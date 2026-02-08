package emailotp

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ========== EMAIL OTP HANDLERS ==========
//
// These handlers implement email+password authentication and OTP verification.

// Handlers encapsulates Email OTP plugin HTTP handlers.
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates Email OTP plugin handlers.
//
// Parameters:
//   - plugin: Initialized Email OTP plugin
//
// Returns:
//   - *Handlers: Handler instance ready for route registration
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// ========== OTP VERIFICATION ==========

// SendOTPHandler handles sending OTP via email.
//
// This endpoint is protected to prevent spam/abuse.
// Only authenticated users can request OTP codes.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/send
//   - Auth: Required (session)
//
// Request Body:
//
//	{
//	  "email": "user@example.com",
//	  "userId": "user_123",
//	  "purpose": "email_verification"  // or "password_reset", "login_mfa"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "OTP sent successfully"
//	}
func (h *Handlers) SendOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize inputs
	req.Email = core.SanitizeEmail(req.Email)
	req.Purpose = core.SanitizeString(req.Purpose, nil)

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.plugin.SendOTP(r.Context(), req.Email, req.Purpose); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP sent successfully",
	})
}

// VerifyOTPHandler handles verifying OTP codes.
//
// This endpoint is public to allow users to verify their email addresses
// without requiring prior authentication.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/verify
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "email": "user@example.com",
//	  "code": "123456",
//	  "purpose": "email_verification"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "OTP verified successfully"
//	}
//
// Response (400 Bad Request):
//
//	{
//	  "success": false,
//	  "error": "Invalid or expired OTP"
//	}
func (h *Handlers) VerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize inputs
	req.Email = core.SanitizeEmail(req.Email)
	req.Code = core.SanitizeString(req.Code, nil)

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	valid, err := h.plugin.VerifyOTP(r.Context(), req.Email, req.Code)
	if err != nil || !valid {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid or expired OTP",
		})
		return
	}

	// Mark email as verified after successful OTP validation
	if err := h.plugin.MarkEmailVerified(r.Context(), req.Email); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to update email verification status",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP verified successfully",
	})
}
