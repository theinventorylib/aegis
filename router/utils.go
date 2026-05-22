package router

import "strings"

// NormalizePath converts router parameter syntax to OpenAPI path syntax.
//
// Different HTTP routers use different syntax for path parameters:
//   - Gin, Echo: /users/:id/posts/:postId
//   - Chi v5, Gorilla Mux, OpenAPI spec: /users/{id}/posts/{postId}
//
// This function normalizes paths from :param syntax (gin/echo) to {param}
// syntax (chi v5/OpenAPI spec) for automatic API documentation generation
// and chi v5 route registration compatibility.
//
// Conversion rules:
//   - :id → {id}
//   - :userId → {userId}
//   - :post_id → {post_id}
//   - Parameter names can contain: a-z, A-Z, 0-9, _
//
// Examples:
//
//	NormalizePath("/users/:id")
//	// Returns: "/users/{id}"
//
//	NormalizePath("/organizations/:orgId/members/:userId")
//	// Returns: "/organizations/{orgId}/members/{userId}"
//
//	NormalizePath("/static/file.css")
//	// Returns: "/static/file.css" (unchanged, no parameters)
func NormalizePath(path string) string {
	result := path
	for {
		start := strings.Index(result, ":")
		if start == -1 {
			break
		}

		// Find end of parameter name
		end := start + 1
		for end < len(result) && isParamChar(result[end]) {
			end++
		}

		paramName := result[start+1 : end]
		result = result[:start] + "{" + paramName + "}" + result[end:]
	}
	return result
}

// isParamChar returns true if the character is valid in a router parameter name.
//
// Valid characters:
//   - Lowercase letters: a-z
//   - Uppercase letters: A-Z
//   - Digits: 0-9
//   - Underscore: _
//
// This matches the typical parameter naming rules for HTTP routers.
func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}
