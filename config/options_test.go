package config

import (
	"net/http"
	"testing"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/server"
)

func TestValidateNilDB(t *testing.T) {
	cfg := Default()
	cfg.DB = nil
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for nil database, got nil")
	}
	if err.Error() != "database provider is required: use WithDB, WithPostgres, or WithMySQL" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidateNilRouter(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = nil
	cfg.APIMode = true

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for nil router, got nil")
	}
	if err.Error() != "router is required: use WithRouter" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidateCSRFRequired(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = false // Web mode
	cfg.CSRFSecret = nil

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for missing CSRF secret in web mode, got nil")
	}
}

func TestValidateCSRFNotRequiredInAPIMode(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.CSRFSecret = nil

	err := cfg.Validate()
	if err != nil {
		t.Errorf("API mode should not require CSRF secret, got error: %v", err)
	}
}

func TestValidateSessionExpiryPositive(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.SessionExpiry = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero session expiry, got nil")
	}
	if err.Error() != "session expiry must be positive" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidateSessionExpiryNegative(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.SessionExpiry = -1 * time.Hour

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for negative session expiry, got nil")
	}
}

func TestValidateRefreshExpiryPositive(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.RefreshExpiry = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero refresh expiry, got nil")
	}
}

func TestValidateRefreshExpiryGreaterThanSession(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.SessionExpiry = 7 * 24 * time.Hour
	cfg.RefreshExpiry = 1 * time.Hour // Less than session expiry

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when refresh expiry < session expiry, got nil")
	}
	if err.Error() != "refresh expiry should be greater than session expiry" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidateArgon2TimeZero(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.Argon2Time = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero Argon2 time, got nil")
	}
	if err.Error() != "Argon2 time parameter must be positive" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidateArgon2MemoryZero(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.Argon2Memory = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero Argon2 memory, got nil")
	}
}

func TestValidateArgon2ThreadsZero(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.Argon2Threads = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero Argon2 threads, got nil")
	}
}

func TestValidateArgon2KeyLengthZero(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true
	cfg.Argon2KeyLength = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero Argon2 key length, got nil")
	}
}

func TestValidateValidConfiguration(t *testing.T) {
	cfg := Default()
	cfg.DB = core.NewMockDB()
	cfg.Router = server.NewDefaultRouter(http.NewServeMux())
	cfg.APIMode = true

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Valid configuration should pass validation, got error: %v", err)
	}
}

// mockLogger implements Logger interface for testing
type mockLogger struct{}

func (mockLogger) Info(_ string, keysAndValues ...interface{})  {}
func (mockLogger) Error(_ string, keysAndValues ...interface{}) {}
func (mockLogger) Debug(_ string, keysAndValues ...interface{}) {}

func TestWithLogger(t *testing.T) {
	var _ Logger = mockLogger{} // Verify it implements Logger interface

	logger := mockLogger{}
	cfg := Default()
	WithLogger(logger)(cfg)

	if cfg.Logger == nil {
		t.Error("Logger should be set")
	}
}

func TestDefaultConfiguration(t *testing.T) {
	cfg := Default()

	// Verify defaults
	if cfg.SessionExpiry != 24*time.Hour {
		t.Errorf("Expected session expiry 24h, got %v", cfg.SessionExpiry)
	}
	if cfg.RefreshExpiry != 7*24*time.Hour {
		t.Errorf("Expected refresh expiry 7d, got %v", cfg.RefreshExpiry)
	}
	if !cfg.CookieHTTPOnly {
		t.Error("Expected CookieHTTPOnly=true by default")
	}
	if !cfg.CookieSecure {
		t.Error("Expected CookieSecure=true by default")
	}
	if cfg.CookieSameSite != "Lax" {
		t.Errorf("Expected CookieSameSite='Lax', got %s", cfg.CookieSameSite)
	}
	if cfg.Argon2Time != 1 {
		t.Errorf("Expected Argon2Time=1, got %d", cfg.Argon2Time)
	}
	if cfg.Argon2Memory != 64*1024 {
		t.Errorf("Expected Argon2Memory=64KB, got %d", cfg.Argon2Memory)
	}
	if cfg.Argon2Threads != 4 {
		t.Errorf("Expected Argon2Threads=4, got %d", cfg.Argon2Threads)
	}
	if cfg.Argon2KeyLength != 32 {
		t.Errorf("Expected Argon2KeyLength=32, got %d", cfg.Argon2KeyLength)
	}
}

func TestOptionsChaining(t *testing.T) {
	db := core.NewMockDB()
	router := server.NewDefaultRouter(http.NewServeMux())

	cfg := Default()
	WithDB(db, "postgres")(cfg)
	WithRouter(router)(cfg)
	WithSessionExpiry(1 * time.Hour)(cfg)
	WithRefreshExpiry(7 * time.Hour)(cfg)
	WithAPIOnlyMode(true)(cfg)

	if cfg.DB == nil {
		t.Error("DB should be set")
	}
	if cfg.Router == nil {
		t.Error("Router should be set")
	}
	if cfg.SessionExpiry != 1*time.Hour {
		t.Error("SessionExpiry should be updated")
	}
	if cfg.RefreshExpiry != 7*time.Hour {
		t.Error("RefreshExpiry should be updated")
	}
	if !cfg.APIMode {
		t.Error("APIMode should be enabled")
	}
}
