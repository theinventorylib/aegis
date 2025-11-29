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

// LoginWithPhoneHandler handles phone+password login
func (h *Handlers) LoginWithPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.authService == nil {
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
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password via core AuthService
	valid, err := h.plugin.authService.VerifyPassword(r.Context(), user.ID, req.Password)
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

		// Set session cookie using central helper to respect configured cookie settings
		if h.plugin.sessionService != nil {
			cfg := h.plugin.sessionService.GetConfig()
			core.SetSessionCookie(w, session.Token, cfg)
		}
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// RegisterWithPhoneHandler handles phone+password registration
func (h *Handlers) RegisterWithPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.authService == nil {
		http.Error(w, "Password authentication not configured", http.StatusNotImplemented)
		return
	}

	var req RegisterWithPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create user with phone and password
	user, err := h.plugin.CreateUserWithPhoneAndPassword(r.Context(), req.PhoneNumber, req.Password)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Create session (auto-login)
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie
		cfg := h.plugin.sessionService.GetConfig()
		core.SetSessionCookie(w, session.Token, cfg)
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Registration successful",
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

// VerifyOTPHandler handles verifying OTP
func (h *Handlers) VerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
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
