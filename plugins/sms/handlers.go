package sms

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Handlers for SMS plugin
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates SMS plugin handlers
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// LoginWithPhoneRequest represents phone+password login
type LoginWithPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

// SendOTPRequest represents the request to send an OTP
type SendOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	UserID      string `json:"userId"`
	Purpose     string `json:"purpose"` // "phone_verification", "password_reset", "login_mfa"
}

// VerifyOTPRequest represents the request to verify an OTP
type VerifyOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Code        string `json:"code"`
	Purpose     string `json:"purpose"`
}

// LoginWithPhoneHandler handles phone+password login
func (h *Handlers) LoginWithPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.passwordPlugin == nil {
		http.Error(w, "Password authentication not configured", http.StatusNotImplemented)
		return
	}

	var req LoginWithPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get user by phone number
	user, err := h.plugin.GetUserByPhone(r.Context(), req.PhoneNumber)
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password via password plugin
	valid, err := h.plugin.passwordPlugin.VerifyPassword(r.Context(), user.ID, req.Password)
	if err != nil || !valid {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Create session
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			_ = core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    session.Token,
			Path:     "/",
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Secure:   true, // Should be configurable based on env
			SameSite: http.SameSiteLaxMode,
		})
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// SendOTPHandler handles sending OTP via SMS
func (h *Handlers) SendOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.plugin.SendOTP(r.Context(), req.PhoneNumber, req.UserID, req.Purpose); err != nil {
		_ = core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP sent successfully",
	})
}

// VerifyOTPHandler handles verifying OTP
func (h *Handlers) VerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	valid, err := h.plugin.VerifyOTP(r.Context(), req.PhoneNumber, req.Code, req.Purpose)
	if err != nil || !valid {
		_ = core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid or expired OTP",
		})
		return
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OTP verified successfully",
	})
}
