package totp

import (
	"errors"
	"net/http"

	"github.com/theinventorylib/aegis/v2/core"
	totptypes "github.com/theinventorylib/aegis/v2/plugins/totp/types"
)

// Handlers encapsulates TOTP plugin HTTP handlers.
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates TOTP plugin handlers.
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// currentUserID returns the authenticated user ID or writes a 401 and returns "".
func currentUserID(w http.ResponseWriter, r *http.Request) string {
	userID := core.GetUserID(r.Context())
	if userID == "" {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{Success: false, Error: "Not authenticated"})
	}
	return userID
}

// SetupHandler starts enrollment.
func (h *Handlers) SetupHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	res, err := h.plugin.Setup(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrAlreadyEnabled) {
			core.WriteJSON(w, http.StatusConflict, &core.Response{Success: false, Error: err.Error()})
			return
		}
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to start setup"})
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Data: res})
}

// EnableHandler confirms a code and activates TOTP.
func (h *Handlers) EnableHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	var req totptypes.EnableRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}
	codes, err := h.plugin.Enable(r.Context(), userID, req.Code)
	switch {
	case errors.Is(err, ErrInvalidCode), errors.Is(err, ErrNotSetup):
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	case errors.Is(err, ErrAlreadyEnabled):
		core.WriteJSON(w, http.StatusConflict, &core.Response{Success: false, Error: err.Error()})
		return
	case err != nil:
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to enable two-factor"})
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    totptypes.EnableResponse{RecoveryCodes: codes},
	})
}

// DisableHandler removes the credential.
func (h *Handlers) DisableHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	if err := h.plugin.Disable(r.Context(), userID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to disable two-factor"})
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Two-factor disabled"})
}

// VerifyHandler steps up the current session with a TOTP or recovery code.
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	session := core.GetSession(r.Context())
	if session == nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{Success: false, Error: "Not authenticated"})
		return
	}
	var req totptypes.VerifyRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}

	var ok bool
	var err error
	if req.Recovery {
		ok, err = h.plugin.VerifyRecoveryCode(r.Context(), userID, req.Code)
	} else {
		ok, err = h.plugin.Verify(r.Context(), userID, req.Code)
	}
	if err != nil && !errors.Is(err, ErrNotEnabled) {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to verify code"})
		return
	}
	if !ok {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "invalid code"})
		return
	}
	if err := h.plugin.MarkSessionVerified(r.Context(), session.ID, userID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to mark session verified"})
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Data: map[string]bool{"verified": true}})
}

// StatusHandler reports enrollment and current-session state.
func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	enabled, err := h.plugin.IsEnabled(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to read status"})
		return
	}
	verified := false
	if session := core.GetSession(r.Context()); session != nil {
		verified = h.plugin.IsSessionVerified(r.Context(), session.ID, userID)
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    totptypes.StatusResponse{Enabled: enabled, Verified: verified},
	})
}

// RegenerateRecoveryCodesHandler replaces the recovery codes after confirming
// a TOTP code.
func (h *Handlers) RegenerateRecoveryCodesHandler(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(w, r)
	if userID == "" {
		return
	}
	var req totptypes.RegenerateRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: "Invalid request"})
		return
	}
	codes, err := h.plugin.RegenerateRecoveryCodes(r.Context(), userID, req.Code)
	switch {
	case errors.Is(err, ErrInvalidCode):
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	case err != nil:
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{Success: false, Error: "failed to regenerate codes"})
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    totptypes.EnableResponse{RecoveryCodes: codes},
	})
}
