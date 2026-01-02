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

// ========== EMAIL+PASSWORD AUTHENTICATION ==========

// LoginWithEmailHandler handles email+password login.
//
// Authentication Flow:
//  1. Validate email format
//  2. Retrieve user by email
//  3. Verify password with bcrypt
//  4. Create session
//  5. Set session cookie
//
// Endpoint:
//   - Method: POST
//   - Path: /emailotp/login
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "email": "user@example.com",
//	  "password": "SecurePassword123!"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Login successful",
//	  "data": {
//	    "user": {"id": "user_123", "email": "user@example.com", ...}
//	  }
//	}
func (h *Handlers) LoginWithEmailHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.accountService == nil {
		core.WriteJSON(w, http.StatusNotImplemented, &core.Response{
			Success: false,
			Error:   "Password authentication not configured",
		})
		return
	}

	var req LoginWithEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get user by email
	user, err := h.plugin.GetUserByEmail(r.Context(), req.Email)
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

// RegisterWithEmailHandler handles email+password registration.
//
// Registration Flow:
//  1. Validate email format
//  2. Check if email already exists
//  3. Create user with hashed password
//  4. Create session (auto-login)
//  5. Set session cookie
//
// Endpoint:
//   - Method: POST
//   - Path: /emailotp/register
//   - Auth: Public
//
// Request Body:
//
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
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
//	    "user": {"id": "user_123", "email": "john@example.com", "emailVerified": false}
//	  }
//	}
func (h *Handlers) RegisterWithEmailHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.accountService == nil {
		core.WriteJSON(w, http.StatusNotImplemented, &core.Response{
			Success: false,
			Error:   "Password authentication not configured",
		})
		return
	}

	var req RegisterWithEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
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

	// Create user with email and password
	user, err := h.plugin.CreateUserWithEmailAndPassword(r.Context(), *req.Name, req.Email, req.Password)
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

// ========== OTP VERIFICATION ==========

// SendOTPHandler handles sending OTP via email.
//
// This endpoint is protected to prevent spam/abuse.
// Only authenticated users can request OTP codes.
//
// Endpoint:
//   - Method: POST
//   - Path: /emailotp/send
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

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.plugin.SendOTP(r.Context(), req.Email, req.UserID, req.Purpose); err != nil {
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
//   - Path: /emailotp/verify
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

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	valid, err := h.plugin.VerifyOTP(r.Context(), req.Email, req.Code, req.Purpose)
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
