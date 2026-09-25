package emailotp

import (
	"errors"
	"net/http"

	"github.com/theinventorylib/aegis/v2/core"
	emailotptypes "github.com/theinventorylib/aegis/v2/plugins/emailotp/types"
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
	var req emailotptypes.SendOTPRequest
	if err := core.ReadJSON(r, &req); err != nil {
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

// SendVerificationOTPHandler sends an email-verification OTP to an address
// without requiring authentication, so newly registered but unverified users
// can prove ownership. Endpoint is public; apply rate limiting at the router
// or gateway to prevent abuse.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/send-verification
//   - Auth: Public
func (h *Handlers) SendVerificationOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}

	email := core.SanitizeEmail(req.Email)
	if err := ValidateEmail(email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// Throttle by email so the unauthenticated endpoint cannot be used to spam
	// an address. Uses the application's configured rate limiter when present.
	if rl := h.plugin.aegis.GetRateLimiter(); rl != nil {
		allowed, _, err := rl.Allow(r.Context(), "email-verification:"+email)
		if err != nil && h.plugin.logger != nil {
			h.plugin.logger.Error("email-otp: rate limiter error", "error", err)
		}
		if !allowed {
			core.WriteJSON(w, http.StatusTooManyRequests, &core.Response{Success: false, Error: "Too many requests"})
			return
		}
	}

	if err := h.plugin.SendOTP(r.Context(), email, "email_verification"); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: err.Error()})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Verification code sent"})
}

// VerifyOTPHandler handles verifying OTP codes.
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
	var req emailotptypes.VerifyOTPRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize inputs
	req.Email = core.SanitizeEmail(req.Email)
	req.Code = core.SanitizeString(req.Code, nil)
	if req.Purpose == "" {
		req.Purpose = core.VerificationTypeEmailVerification
	}

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	valid, err := h.plugin.VerifyOTP(r.Context(), req.Email, req.Purpose, req.Code)
	if err != nil || !valid {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid or expired OTP",
		})
		return
	}

	// Only an email-verification code proves ownership of the address; a
	// password-reset or MFA code must not flip the verified flag.
	if req.Purpose == core.VerificationTypeEmailVerification {
		if err := h.plugin.MarkEmailVerified(r.Context(), req.Email); err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to update email verification status",
			})
			return
		}
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP verified successfully",
	})
}

// ========== PASSWORD RESET ==========

// ForgotPasswordHandler starts the password-reset flow. The response is the
// same whether or not the address has an account, so the endpoint cannot be
// used to enumerate users; delivery failures are logged, not surfaced.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/forgot-password
//   - Auth: Public (rate-limited per email when a rate limiter is configured)
func (h *Handlers) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req emailotptypes.ForgotPasswordRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}

	email := core.SanitizeEmail(req.Email)
	if err := ValidateEmail(email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// Throttle by email so the public endpoint cannot be used to spam an
	// address. Uses the application's configured rate limiter when present.
	if h.plugin.aegis != nil {
		if rl := h.plugin.aegis.GetRateLimiter(); rl != nil {
			allowed, _, err := rl.Allow(r.Context(), "password-reset:"+email)
			if err != nil && h.plugin.logger != nil {
				h.plugin.logger.Error("email-otp: rate limiter error", "error", err)
			}
			if !allowed {
				core.WriteJSON(w, http.StatusTooManyRequests, &core.Response{Success: false, Error: "Too many requests"})
				return
			}
		}
	}

	if err := h.plugin.RequestPasswordReset(r.Context(), email); err != nil && h.plugin.logger != nil {
		// An error can only happen for an address that has an account, so
		// surfacing it would leak account existence.
		h.plugin.logger.Error("email-otp: password reset request failed", "error", err)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "If that email has an account, a reset link has been sent",
	})
}

// ResetPasswordHandler completes a password reset.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/reset-password
//   - Auth: Public
func (h *Handlers) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req emailotptypes.ResetPasswordRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "token and new_password are required"})
		return
	}

	if err := h.plugin.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Password reset successfully"})
}

// ========== EMAIL CHANGE ==========

// RequestEmailChangeHandler starts the verified email-change flow for the
// authenticated user and mails a confirmation token to the new address.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/email-change
//   - Auth: Required (session)
func (h *Handlers) RequestEmailChangeHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetUserID(r.Context())
	if userID == "" {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{Success: false, Error: "Not authenticated"})
		return
	}

	var req emailotptypes.RequestEmailChangeRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}
	newEmail := core.SanitizeEmail(req.NewEmail)
	if err := ValidateEmail(newEmail); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if err := h.plugin.RequestEmailChange(r.Context(), userID, newEmail); err != nil {
		switch {
		case errors.Is(err, core.ErrEmailAlreadyExists):
			core.WriteJSON(w, http.StatusConflict, &core.Response{Success: false, Error: "that email is already in use"})
		case core.GetValidationErrors(err) != nil:
			core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		default:
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to start email change"})
		}
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Confirmation sent to the new address"})
}

// ConfirmEmailChangeHandler completes an email change with the token mailed to
// the new address.
//
// Endpoint:
//   - Method: POST
//   - Path: /email-otp/email-change/confirm
//   - Auth: Required (session)
func (h *Handlers) ConfirmEmailChangeHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetUserID(r.Context())
	if userID == "" {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{Success: false, Error: "Not authenticated"})
		return
	}

	var req emailotptypes.ConfirmEmailChangeRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}
	if req.Token == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "token is required"})
		return
	}

	user, err := h.plugin.ConfirmEmailChange(r.Context(), userID, req.Token)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Email updated",
		Data:    map[string]string{"email": user.Email},
	})
}
