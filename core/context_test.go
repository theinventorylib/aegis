package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theinventorylib/aegis/auth"
)

// TC-CTX-001: GetPathParam with PathParamFunc in context
func TestGetPathParam_WithPathParamFunc(t *testing.T) {
	// Given
	fn := PathParamFunc(func(r *http.Request, name string) string {
		params := map[string]string{
			"id":     "org_abc123",
			"teamId": "team_xyz789",
			"userId": "usr_def456",
		}
		return params[name]
	})

	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc123", nil)
	ctx := WithPathParamFunc(req.Context(), fn)
	req = req.WithContext(ctx)

	// When
	result := GetPathParam(req, "id")

	// Then
	if result != "org_abc123" {
		t.Errorf("GetPathParam(id) = %q, want %q", result, "org_abc123")
	}
}

// TC-CTX-002: GetPathParam with multiple parameters
func TestGetPathParam_MultipleParams(t *testing.T) {
	// Given
	fn := PathParamFunc(func(r *http.Request, name string) string {
		params := map[string]string{
			"id":     "org_abc123",
			"teamId": "team_xyz789",
		}
		return params[name]
	})

	req := httptest.NewRequest(http.MethodGet, "/organizations/org_abc123/teams/team_xyz789", nil)
	ctx := WithPathParamFunc(req.Context(), fn)
	req = req.WithContext(ctx)

	// When & Then
	if got := GetPathParam(req, "id"); got != "org_abc123" {
		t.Errorf("GetPathParam(id) = %q, want %q", got, "org_abc123")
	}
	if got := GetPathParam(req, "teamId"); got != "team_xyz789" {
		t.Errorf("GetPathParam(teamId) = %q, want %q", got, "team_xyz789")
	}
}

// TC-CTX-003: GetPathParam falls back to r.PathValue when no PathParamFunc
func TestGetPathParam_PathValueFallback(t *testing.T) {
	tests := []struct {
		name      string
		setupReq  func() *http.Request
		paramName string
		expected  string
	}{
		{
			name: "r.PathValue returns value",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.SetPathValue("id", "fallback_value")
				return req
			},
			paramName: "id",
			expected:  "fallback_value",
		},
		{
			name: "no PathParamFunc, no PathValue set",
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			paramName: "id",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result := GetPathParam(req, tt.paramName)
			if result != tt.expected {
				t.Errorf("GetPathParam(%q) = %q, want %q", tt.paramName, result, tt.expected)
			}
		})
	}
}

// TC-CTX-004: PathParamFunc takes precedence over r.PathValue
func TestGetPathParam_PathParamFuncTakesPrecedence(t *testing.T) {
	// Given - both PathParamFunc and PathValue are set
	fn := PathParamFunc(func(r *http.Request, name string) string {
		return "from_param_func"
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("id", "from_path_value")
	ctx := WithPathParamFunc(req.Context(), fn)
	req = req.WithContext(ctx)

	// When
	result := GetPathParam(req, "id")

	// Then - PathParamFunc should take precedence
	if result != "from_param_func" {
		t.Errorf("PathParamFunc should take precedence: got %q, want %q", result, "from_param_func")
	}
}

// TC-CTX-005: PathParamFunc returns empty string falls back to PathValue
func TestGetPathParam_PathParamFuncEmptyFallsBack(t *testing.T) {
	// Given - PathParamFunc returns empty for unknown param, PathValue has it
	fn := PathParamFunc(func(r *http.Request, name string) string {
		return ""
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("id", "from_path_value")
	ctx := WithPathParamFunc(req.Context(), fn)
	req = req.WithContext(ctx)

	// When
	result := GetPathParam(req, "id")

	// Then - empty PathParamFunc result should fall back to PathValue
	if result != "from_path_value" {
		t.Errorf("Empty PathParamFunc should fall back to PathValue: got %q, want %q", result, "from_path_value")
	}
}

// TC-CTX-006: GetSanitizedPathParam sanitizes the extracted value
func TestGetSanitizedPathParam(t *testing.T) {
	tests := []struct {
		name      string
		rawValue  string
		paramName string
		expected  string
	}{
		{
			name:      "clean value passes through",
			rawValue:  "org_abc123",
			paramName: "id",
			expected:  "org_abc123",
		},
		{
			name:      "trims whitespace",
			rawValue:  "  org_abc123  ",
			paramName: "id",
			expected:  "org_abc123",
		},
		{
			name:      "strips HTML tags from param value",
			rawValue:  "org_<b>abc</b>123",
			paramName: "id",
			expected:  "org_abc123",
		},
		{
			name:      "removes null bytes",
			rawValue:  "org\x00_abc123",
			paramName: "id",
			expected:  "org_abc123",
		},
		{
			name:      "empty param name returns empty",
			rawValue:  "",
			paramName: "nonexistent",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			fn := PathParamFunc(func(r *http.Request, name string) string {
				if name == tt.paramName {
					return tt.rawValue
				}
				return ""
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := WithPathParamFunc(req.Context(), fn)
			req = req.WithContext(ctx)

			// When
			result := GetSanitizedPathParam(req, tt.paramName)

			// Then
			if result != tt.expected {
				t.Errorf("GetSanitizedPathParam(%q) = %q, want %q", tt.paramName, result, tt.expected)
			}
		})
	}
}

// TC-CTX-007: WithPathParamFunc preserves other context values
func TestWithPathParamFunc_PreservesContext(t *testing.T) {
	// Given
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := req.Context()

	// Set user in context first
	user := &auth.User{ID: "usr_test", Email: "test@example.com", Name: "Test User"}
	ctx = WithUser(ctx, user)
	ctx = WithPathParamFunc(ctx, PathParamFunc(func(r *http.Request, name string) string {
		return "param_value"
	}))

	req = req.WithContext(ctx)

	// When
	retrievedUser, err := GetUser(req.Context())
	paramValue := GetPathParam(req, "test_param")

	// Then
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if retrievedUser.ID != "usr_test" {
		t.Errorf("User ID should be preserved: got %q, want %q", retrievedUser.ID, "usr_test")
	}
	if paramValue != "param_value" {
		t.Errorf("PathParam should work: got %q, want %q", paramValue, "param_value")
	}
}