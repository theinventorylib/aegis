package sms

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ========== SMS HANDLERS ==========
//
// These handlers implement phone+password authentication and SMS OTP verification.

// Handlers encapsulates SMS plugin HTTP handlers.
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates SMS plugin handlers.
//
// Parameters:
//   - plugin: Initialized SMS plugin
//
// Returns:
//   - *Handlers: Handler instance ready for route registration
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// ========== PHONE+PASSWORD AUTHENTICATION ==========

// LoginWithPhoneHandler handles phone+password login.
//
// Authentication Flow:
//  1. Validate phone number format (E.164)
//  2. Retrieve user by phone number
//  3. Verify password with bcrypt
//  4. Create session
//  5. Set session cookie
//
// Endpoint:
//   - Method: POST
//   - Path: /sms/login
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "password": "SecurePassword123!"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Login successful",
//	  "data": {
//	    "user": {"id": "user_123", "phone": "+14155551234", ...}
//	  }
//	}
func (h *Handlers) LoginWithPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.accountService == nil {
		core.WriteJSON(w, http.StatusNotImplemented, &core.Response{
			Success: false,
			Error:   "Password authentication not configured",
		})
		return
	}

	var req LoginWithPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate phone number format
	if err := ValidatePhoneNumber(req.PhoneNumber); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get user by phone number
	user, err := h.plugin.GetUserByPhone(r.Context(), req.PhoneNumber)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password via core AuthService
	valid, err := h.plugin.accountService.VerifyPassword(r.Context(), user.ID, req.Password)
	if err != nil || !valid {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Create session
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie using CookieManager
		h.plugin.sessionService.GetCookieManager().SetSessionCookie(w, session.Token)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// RegisterWithPhoneHandler handles phone+password registration.
//
// Registration Flow:
//  1. Validate phone number format (E.164)
//  2. Check if phone already exists
//  3. Create user with hashed password
//  4. Create session (auto-login)
//  5. Set session cookie
//
// Endpoint:
//   - Method: POST
//   - Path: /sms/register
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "name": "John Doe",
//	  "phoneNumber": "+14155551234",
//	  "password": "SecurePassword123!",
//	  "avatar": "https://example.com/avatar.jpg"  // Optional
//	}
//
// Response (201 Created):
//
//	{
//	  "success": true,
//	  "message": "Registration successful",
//	  "data": {
//	    "user": {"id": "user_123", "phone": "+14155551234", "phoneVerified": false}
//	  }
//	}
func (h *Handlers) RegisterWithPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.accountService == nil {
		core.WriteJSON(w, http.StatusNotImplemented, &core.Response{
			Success: false,
			Error:   "Password authentication not configured",
		})
		return
	}

	var req RegisterWithPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate phone number format
	if err := ValidatePhoneNumber(req.PhoneNumber); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Name == nil || *req.Name == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Name is required",
		})
		return
	}

	// Create user with phone and password
	user, err := h.plugin.CreateUserWithPhoneAndPassword(r.Context(), *req.Name, req.PhoneNumber, req.Password)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Create session (auto-login)
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), &user.User, r.RemoteAddr, r.UserAgent())
		if err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie using CookieManager
		h.plugin.sessionService.GetCookieManager().SetSessionCookie(w, session.Token)
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Registration successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// ========== SMS OTP VERIFICATION ==========

// SendOTPHandler handles sending OTP via SMS.
//
// This endpoint is protected to prevent spam/abuse and SMS cost attacks.
// Only authenticated users can request OTP codes.
//
// Endpoint:
//   - Method: POST
//   - Path: /sms/send
//   - Auth: Required (session)
//
// Request Body:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "userId": "user_123",
//	  "purpose": "phone_verification"  // or "password_reset", "login_mfa"
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

	// Validate phone number format
	if err := ValidatePhoneNumber(req.PhoneNumber); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.plugin.SendOTP(r.Context(), req.PhoneNumber, req.UserID, req.Purpose); err != nil {
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

// VerifyOTPHandler handles verifying SMS OTP codes.
//
// This endpoint is public to allow users to verify their phone numbers
// without requiring prior authentication.
//
// Endpoint:
//   - Method: POST
//   - Path: /sms/verify
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "code": "123456",
//	  "purpose": "phone_verification"
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

	// Validate phone number format
	if err := ValidatePhoneNumber(req.PhoneNumber); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	valid, err := h.plugin.VerifyOTP(r.Context(), req.PhoneNumber, req.Code, req.Purpose)
	if err != nil || !valid {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid or expired OTP",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP verified successfully",
	})
}
