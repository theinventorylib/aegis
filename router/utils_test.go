package router

import "testing"

// TC-UTIL-001: Single parameter conversion
func TestNormalizePath_SingleParam(t *testing.T) {
	// Given
	input := "/users/:id"

	// When
	result := NormalizePath(input)

	// Then
	expected := "/users/{id}"
	if result != expected {
		t.Errorf("NormalizePath(%q) = %q, want %q", input, result, expected)
	}
}

// TC-UTIL-002: Multiple parameters
func TestNormalizePath_MultipleParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "two params",
			input:    "/organizations/:orgId/members/:userId",
			expected: "/organizations/{orgId}/members/{userId}",
		},
		{
			name:     "three params",
			input:    "/orgs/:orgId/teams/:teamId/members/:userId",
			expected: "/orgs/{orgId}/teams/{teamId}/members/{userId}",
		},
		{
			name:     "param with underscore",
			input:    "/api/:user_id",
			expected: "/api/{user_id}",
		},
		{
			name:     "param with numbers",
			input:    "/api/:page2",
			expected: "/api/{page2}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TC-UTIL-003: No parameters
func TestNormalizePath_NoParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "static path",
			input:    "/users/list",
			expected: "/users/list",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "health endpoint",
			input:    "/health",
			expected: "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TC-UTIL-004: Already OpenAPI syntax (no conversion)
func TestNormalizePath_AlreadyOpenAPI(t *testing.T) {
	input := "/users/{id}"
	result := NormalizePath(input)
	if result != input {
		t.Errorf("Already OpenAPI path should be unchanged: got %q, want %q", result, input)
	}
}

// TC-UTIL-005: Empty string
func TestNormalizePath_EmptyString(t *testing.T) {
	result := NormalizePath("")
	if result != "" {
		t.Errorf("Empty string should remain empty: got %q, want %q", result, "")
	}
}

// TC-UTIL-006: Colon not followed by param chars (should not convert)
func TestNormalizePath_ColonNotParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing colon with no param name",
			input:    "/path/:",
			expected: "/path/{}",
		},
		{
			name:     "colon in middle of non-param (e.g. port)",
			input:    "/host/localhost:8080",
			expected: "/host/localhost{8080}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TC-UTIL-007: Mixed static and param segments
func TestNormalizePath_MixedSegments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "static prefix with param",
			input:    "/auth/organizations/:id",
			expected: "/auth/organizations/{id}",
		},
		{
			name:     "param between static segments",
			input:    "/auth/organizations/:id/teams/:teamId",
			expected: "/auth/organizations/{id}/teams/{teamId}",
		},
		{
			name:     "trailing static after param",
			input:    "/users/:id/settings",
			expected: "/users/{id}/settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
