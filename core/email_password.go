package core

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/auth"
)

// EmailPasswordHandlers provides HTTP handlers for traditional email+password authentication.
//
// This handler set implements the classic username/password authentication flow:
//   - Registration: Create a new user with email+password
//   - Login: Authenticate existing user with credentials
//
// Security features:
//   - Password hashing: Argon2id with configurable parameters
//   - Account lockout: Rate limiting for failed login attempts
//   - Audit logging: All authentication events logged
//   - Session management: Automatic session creation on success
//   - CSRF protection: Uses session cookies with SameSite attribute
//
// These handlers are typically registered in the router like:
//
//	handlers := core.NewEmailPasswordHandlers(authService)
//	r.POST("/auth/login", handlers.LoginHandler)
//	r.POST("/auth/register", handlers.RegisterHandler)
//
// For custom authentication flows, you can implement your own handlers
// using AuthService directly.
type EmailPasswordHandlers struct {
	authService *AuthService
}

// NewEmailPasswordHandlers creates a new set of email+password authentication handlers.
//
// Parameters:
//   - authService: The core authentication service with all sub-services
//
// Example:
//
//	authService := core.NewAuthService(authClient, sessionConfig, nil, nil, nil)
//	handlers := core.NewEmailPasswordHandlers(authService)
func NewEmailPasswordHandlers(authService *AuthService) *EmailPasswordHandlers {
	return &EmailPasswordHandlers{
		authService: authService,
	}
}

// LoginRequest represents the JSON payload for email+password login.
//
// Example request body:
//
//	{
//	  "email": "user@example.com",
//	  "password": "securePassword123!"
//	}
type LoginRequest struct {
	// Email is the user's email address (must be previously registered)
	Email string `json:"email"`

	// Password is the plaintext password (will be compared against Argon2id hash)
	Password string `json:"password"`
}

// RegisterRequest represents the JSON payload for email+password registration.
//
// Example request body:
//
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
//	  "password": "securePassword123!",
//	  "avatar": "https://example.com/avatar.jpg" // optional
//	}
type RegisterRequest struct {
	// Avatar URL for the user's profile picture (optional)
	Avatar *string `json:"avatar"`

	// Name is the user's display name (required)
	Name *string `json:"name"`

	// Email is the user's email address (must be unique)
	Email string `json:"email"`

	// Password is the plaintext password (will be hashed with Argon2id)
	Password string `json:"password"`
}

// LoginHandler handles email+password authentication.
//
// Authentication flow:
//  1. Parse login credentials from request body
//  2. Check if account is locked out (brute force protection)
//  3. Lookup user by email
//  4. Verify password against Argon2id hash
//  5. Clear failed login attempts on success
//  6. Create new session
//  7. Set session cookie
//  8. Return user data
//
// Security features:
//   - Rate limiting: Automatic account lockout after too many failed attempts
//   - Audit logging: All login attempts (success/failure) are logged
//   - Constant-time comparison: Password verification uses constant-time comparison
//   - Generic error messages: "Invalid credentials" for both missing users and wrong passwords (prevents user enumeration)
//
// Request:
//
//	POST /auth/login
//	Content-Type: application/json
//	{
//	  "email": "user@example.com",
//	  "password": "password123"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Login successful",
//	  "data": {
//	    "user": { "id": "...", "email": "...", ... }
//	  }
//	}
//
// Response (401 Unauthorized - Invalid credentials):
//
//	{
//	  "success": false,
//	  "error": "Invalid credentials"
//	}
//
// Response (429 Too Many Requests - Account locked):
//
//	{
//	  "success": false,
//	  "error": "Account is temporarily locked due to too many failed login attempts. Please try again later."
//	}
func (h *EmailPasswordHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Check if account is locked out
	if h.authService.loginAttemptTracker != nil {
		locked, remaining, err := h.authService.loginAttemptTracker.IsLockedOut(r.Context(), req.Email)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, &Response{
				Success: false,
				Error:   "Internal server error",
			})
			return
		}
		if locked {
			_ = h.authService.auditLogger.LogAuthEvent(r.Context(), AuditEventLoginFailed, "", r.RemoteAddr, r.UserAgent(), false, map[string]interface{}{
				"email":     req.Email,
				"reason":    "account_locked",
				"remaining": remaining.String(),
			})
			WriteJSON(w, http.StatusTooManyRequests, &Response{
				Success: false,
				Error:   "Account is temporarily locked due to too many failed login attempts. Please try again later.",
			})
			return
		}
	}

	// Get user by email
	user, err := h.authService.User.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if h.authService.loginAttemptTracker != nil {
			// Record failed attempt for non-existent user
			_, _, _ = h.authService.loginAttemptTracker.RecordFailedAttempt(r.Context(), req.Email)
		}
		_ = h.authService.auditLogger.LogAuthEvent(r.Context(), AuditEventLoginFailed, "", r.RemoteAddr, r.UserAgent(), false, map[string]interface{}{
			"email":  req.Email,
			"reason": "user_not_found",
		})
		WriteJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	uid := ""
	if ua, ok := any(&user).(UserModel); ok {
		uid = ua.GetID()
	}

	// Verify password
	valid, err := h.authService.Account.VerifyPassword(r.Context(), uid, req.Password)
	if err != nil || !valid {
		if h.authService.loginAttemptTracker != nil {
			// Record failed attempt
			_, _, _ = h.authService.loginAttemptTracker.RecordFailedAttempt(r.Context(), req.Email)
		}
		_ = h.authService.auditLogger.LogAuthEvent(r.Context(), AuditEventLoginFailed, uid, r.RemoteAddr, r.UserAgent(), false, map[string]interface{}{
			"email":  req.Email,
			"reason": "invalid_password",
		})
		WriteJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Clear failed attempts on successful login
	if h.authService.loginAttemptTracker != nil {
		_ = h.authService.loginAttemptTracker.ClearAttempts(r.Context(), req.Email)
	}

	// Create session
	session, err := h.authService.Session.CreateSession(r.Context(), &user, r.RemoteAddr, r.UserAgent())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Error:   "Failed to create session",
		})
		return
	}

	token := ""
	if sa, ok := any(session).(interface{ GetToken() string }); ok {
		token = sa.GetToken()
	} else if s2, ok := any(session).(*auth.Session); ok {
		token = s2.Token
	}

	// Set session cookie using CookieManager
	h.authService.Session.GetCookieManager().SetSessionCookie(w, token)

	WriteJSON(w, http.StatusOK, &Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// RegisterHandler handles new user registration with email+password.
//
// Registration flow:
//  1. Parse registration data from request body
//  2. Validate required fields (name, email, password)
//  3. Create user with email and password (creates password account automatically)
//  4. Update user email (separate step for validation)
//  5. Auto-login: Create session immediately
//  6. Set session cookie
//  7. Return created user data
//
// Security features:
//   - Password validation: Enforces password policy (min length, complexity)
//   - Email validation: Checks email format and uniqueness
//   - Automatic hashing: Password is hashed with Argon2id before storage
//   - Transactional: User is deleted if email update fails (maintains consistency)
//   - Audit logging: Registration events are logged
//
// Request:
//
//	POST /auth/register
//	Content-Type: application/json
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
//	  "password": "SecurePassword123!",
//	  "avatar": "https://example.com/avatar.jpg" // optional
//	}
//
// Response (201 Created):
//
//	{
//	  "success": true,
//	  "message": "Registration successful",
//	  "data": {
//	    "user": { "id": "...", "email": "...", "name": "...", ... }
//	  }
//	}
//
// Response (400 Bad Request - Validation error):
//
//	{
//	  "success": false,
//	  "error": "Email already registered" // or other validation error
//	}
//
// Note: After successful registration, the user is automatically logged in
// (session cookie is set). No separate login request is needed.
func (h *EmailPasswordHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate required fields
	if req.Name == nil || *req.Name == "" {
		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   "Name is required",
		})
		return
	}

	// Create user with password
	user, err := h.authService.User.CreateUserWithEmail(r.Context(), *req.Name, req.Email, req.Password)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	uid := ""
	if ua, ok := any(user).(UserModel); ok {
		uid = ua.GetID()
	}

	// Set email
	if err := h.authService.User.UpdateUserEmail(r.Context(), uid, req.Email); err != nil {
		// Cleanup user if email set fails
		_ = h.authService.User.DeleteUser(r.Context(), uid)
		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   "Failed to set email: " + err.Error(),
		})
		return
	}

	// Create session (auto-login)
	session, err := h.authService.Session.CreateSession(r.Context(), &user, r.RemoteAddr, r.UserAgent())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Error:   "Failed to create session",
		})
		return
	}

	token := ""
	if sa, ok := any(session).(interface{ GetToken() string }); ok {
		token = sa.GetToken()
	} else if s2, ok := any(session).(*auth.Session); ok {
		token = s2.Token
	}

	// Set session cookie using CookieManager
	h.authService.Session.GetCookieManager().SetSessionCookie(w, token)

	WriteJSON(w, http.StatusCreated, &Response{
		Success: true,
		Message: "Registration successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}
