package server

import "testing"

func TestNormalizePathToOpenAPI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single parameter",
			input:    "/users/:id",
			expected: "/users/{id}",
		},
		{
			name:     "multiple parameters",
			input:    "/organizations/:orgId/teams/:teamId",
			expected: "/organizations/{orgId}/teams/{teamId}",
		},
		{
			name:     "parameter with underscore",
			input:    "/items/:item_id",
			expected: "/items/{item_id}",
		},
		{
			name:     "no parameters",
			input:    "/users",
			expected: "/users",
		},
		{
			name:     "parameter at end",
			input:    "/sessions/:id",
			expected: "/sessions/{id}",
		},
		{
			name:     "parameter at start",
			input:    "/:id/details",
			expected: "/{id}/details",
		},
		{
			name:     "mixed with query-like syntax",
			input:    "/users/:userId/posts/:postId",
			expected: "/users/{userId}/posts/{postId}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePathToOpenAPI(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePathToOpenAPI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
