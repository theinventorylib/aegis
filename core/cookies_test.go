package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TC-COOK-001: Cookie Manager Creation
func TestNewCookieManager(t *testing.T) {
	// Given
	config := DefaultSessionConfig()

	// When
	cm := NewCookieManager(config)

	// Then
	if cm == nil {
		t.Fatal("NewCookieManager should return a non-nil manager")
	}
}

// TC-COOK-002: Cookie Manager with Nil Config
func TestNewCookieManager_NilConfig(t *testing.T) {
	// When - nil config should use defaults
	cm := NewCookieManager(nil)

	// Then
	if cm == nil {
		t.Fatal("NewCookieManager should return a non-nil manager with nil config")
	}
	if cm.GetConfig() == nil {
		t.Error("Cookie manager should have a default config")
	}
}

// TC-COOK-003: Get Session Cookie Name
func TestCookieManager_GetSessionCookieName(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)

	// When
	name := cm.GetSessionCookieName()

	// Then
	if name == "" {
		t.Error("Session cookie name should not be empty")
	}
	if name != DefaultCookieName {
		t.Errorf("Expected default cookie name %s, got %s", DefaultCookieName, name)
	}
}

// TC-COOK-004: Get Custom Session Cookie Name
func TestCookieManager_GetSessionCookieName_Custom(t *testing.T) {
	// Given
	customName := "my_custom_session"
	config := DefaultSessionConfig()
	config.CookieSettings.Name = customName
	cm := NewCookieManager(config)

	// When
	name := cm.GetSessionCookieName()

	// Then
	if name != customName {
		t.Errorf("Expected custom cookie name %s, got %s", customName, name)
	}
}

// TC-COOK-005: Set Session Cookie
func TestCookieManager_SetSessionCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()
	token := "test_session_token_12345"

	// When
	cm.SetSessionCookie(w, token)

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	cookie := cookies[0]
	if cookie.Name != DefaultCookieName {
		t.Errorf("Expected cookie name %s, got %s", DefaultCookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Errorf("Expected cookie value %s, got %s", token, cookie.Value)
	}
}

// TC-COOK-006: Cookie HttpOnly Flag
func TestCookieManager_SetSessionCookie_HttpOnly(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	config.CookieSettings.HTTPOnly = true
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.SetSessionCookie(w, "test_token")

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	if !cookies[0].HttpOnly {
		t.Error("Cookie should have HttpOnly flag set")
	}
}

// TC-COOK-007: Cookie Secure Flag
func TestCookieManager_SetSessionCookie_Secure(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	config.CookieSettings.Secure = true
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.SetSessionCookie(w, "test_token")

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	if !cookies[0].Secure {
		t.Error("Cookie should have Secure flag set")
	}
}

// TC-COOK-008: Cookie SameSite Configuration
func TestCookieManager_SameSite(t *testing.T) {
	tests := []struct {
		name         string
		sameSite     string
		expectedMode http.SameSite
	}{
		{
			name:         "Lax",
			sameSite:     "Lax",
			expectedMode: http.SameSiteLaxMode,
		},
		{
			name:         "Strict",
			sameSite:     "Strict",
			expectedMode: http.SameSiteStrictMode,
		},
		{
			name:         "None",
			sameSite:     "None",
			expectedMode: http.SameSiteNoneMode,
		},
		{
			name:         "Default (empty)",
			sameSite:     "",
			expectedMode: http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSessionConfig()
			config.CookieSettings.SameSite = tt.sameSite
			cm := NewCookieManager(config)
			w := httptest.NewRecorder()

			cm.SetSessionCookie(w, "test_token")

			cookies := w.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("Expected at least one cookie")
			}

			if cookies[0].SameSite != tt.expectedMode {
				t.Errorf("Expected SameSite %v, got %v", tt.expectedMode, cookies[0].SameSite)
			}
		})
	}
}

// TC-COOK-009: Cookie Path
func TestCookieManager_SetSessionCookie_Path(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.SetSessionCookie(w, "test_token")

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	if cookies[0].Path != DefaultCookiePath {
		t.Errorf("Expected path %s, got %s", DefaultCookiePath, cookies[0].Path)
	}
}

// TC-COOK-010: Clear Session Cookie
func TestCookieManager_ClearSessionCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.ClearSessionCookie(w)

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	cookie := cookies[0]
	if cookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 for deletion, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Errorf("Expected empty value for deletion, got %s", cookie.Value)
	}
}

// TC-COOK-011: Get Session Cookie from Request
func TestCookieManager_GetSessionCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	expectedToken := "test_session_token"

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: expectedToken,
	})

	// When
	token, err := cm.GetSessionCookie(req)

	// Then
	if err != nil {
		t.Fatalf("GetSessionCookie failed: %v", err)
	}
	if token != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token)
	}
}

// TC-COOK-012: Get Missing Session Cookie
func TestCookieManager_GetSessionCookie_Missing(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	req := httptest.NewRequest("GET", "/", nil)

	// When
	token, err := cm.GetSessionCookie(req)

	// Then
	if err == nil {
		t.Error("Expected error for missing cookie")
	}
	if token != "" {
		t.Errorf("Expected empty token for missing cookie, got %s", token)
	}
}

// TC-COOK-013: Get Cookie by Name
func TestCookieManager_GetCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	cookieName := "custom_cookie"
	expectedValue := "custom_value"

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: expectedValue,
	})

	// When
	value, err := cm.GetCookie(req, cookieName)

	// Then
	if err != nil {
		t.Fatalf("GetCookie failed: %v", err)
	}
	if value != expectedValue {
		t.Errorf("Expected value %s, got %s", expectedValue, value)
	}
}

// TC-COOK-014: Get Missing Cookie by Name
func TestCookieManager_GetCookie_Missing(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	req := httptest.NewRequest("GET", "/", nil)

	// When
	value, err := cm.GetCookie(req, "nonexistent_cookie")

	// Then
	if err == nil {
		t.Error("Expected error for missing cookie")
	}
	if value != "" {
		t.Errorf("Expected empty value for missing cookie, got %s", value)
	}
}

// TC-COOK-015: Set Custom Cookie
func TestCookieManager_SetCustomCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	opts := CookieOptions{
		Name:     "custom_cookie",
		Value:    "custom_value",
		Path:     "/api",
		MaxAge:   3600,
		HTTPOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	// When
	cm.SetCustomCookie(w, opts)

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	cookie := cookies[0]
	if cookie.Name != opts.Name {
		t.Errorf("Expected name %s, got %s", opts.Name, cookie.Name)
	}
	if cookie.Value != opts.Value {
		t.Errorf("Expected value %s, got %s", opts.Value, cookie.Value)
	}
	if cookie.Path != opts.Path {
		t.Errorf("Expected path %s, got %s", opts.Path, cookie.Path)
	}
	if cookie.MaxAge != opts.MaxAge {
		t.Errorf("Expected MaxAge %d, got %d", opts.MaxAge, cookie.MaxAge)
	}
	if !cookie.HttpOnly {
		t.Error("Expected HttpOnly to be true")
	}
	if !cookie.Secure {
		t.Error("Expected Secure to be true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("Expected SameSite Strict, got %v", cookie.SameSite)
	}
}

// TC-COOK-016: Cookie Domain Configuration
func TestCookieManager_SetCookie_Domain(t *testing.T) {
	// Given
	domain := "example.com"
	config := DefaultSessionConfig()
	config.CookieSettings.Domain = domain
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.SetSessionCookie(w, "test_token")

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	// Note: Go's http package normalizes domains (strips leading dots)
	if cookies[0].Domain != domain {
		t.Errorf("Expected domain %s, got %s", domain, cookies[0].Domain)
	}
}

// TC-COOK-017: Cookie Expiration
func TestCookieManager_SetCookie_Expiration(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	config.SessionExpiry = 2 * time.Hour
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When
	cm.SetSessionCookie(w, "test_token")

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	// MaxAge should be approximately 2 hours in seconds
	expectedMaxAge := int(config.SessionExpiry.Seconds())
	if cookies[0].MaxAge != expectedMaxAge {
		t.Errorf("Expected MaxAge %d, got %d", expectedMaxAge, cookies[0].MaxAge)
	}
}

// TC-COOK-018: Set Generic Cookie
func TestCookieManager_SetCookie(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	cookieName := "test_cookie"
	cookieValue := "test_value"
	maxAge := 30 * time.Minute

	// When
	cm.SetCookie(w, cookieName, cookieValue, maxAge)

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	cookie := cookies[0]
	if cookie.Name != cookieName {
		t.Errorf("Expected name %s, got %s", cookieName, cookie.Name)
	}
	if cookie.Value != cookieValue {
		t.Errorf("Expected value %s, got %s", cookieValue, cookie.Value)
	}
	if cookie.MaxAge != int(maxAge.Seconds()) {
		t.Errorf("Expected MaxAge %d, got %d", int(maxAge.Seconds()), cookie.MaxAge)
	}
}

// TC-COOK-019: Default Cookie Settings
func TestDefaultCookieSettings(t *testing.T) {
	// When
	config := DefaultSessionConfig()

	// Then
	if config.CookieSettings.HTTPOnly != true {
		t.Errorf("Expected HTTPOnly true, got %v", config.CookieSettings.HTTPOnly)
	}
	if config.CookieSettings.Secure != true {
		t.Errorf("Expected Secure true, got %v", config.CookieSettings.Secure)
	}
	if config.CookieSettings.SameSite != DefaultCookieSameSite {
		t.Errorf("Expected SameSite %s, got %s", DefaultCookieSameSite, config.CookieSettings.SameSite)
	}
}

// TC-COOK-020: Cookie with Empty Value
func TestCookieManager_SetCookie_EmptyValue(t *testing.T) {
	// Given
	config := DefaultSessionConfig()
	cm := NewCookieManager(config)
	w := httptest.NewRecorder()

	// When - Set cookie with empty value (used for deletion)
	cm.SetCookie(w, "test_cookie", "", -1)

	// Then
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected at least one cookie")
	}

	if cookies[0].Value != "" {
		t.Errorf("Expected empty value, got %s", cookies[0].Value)
	}
}
