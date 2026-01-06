package router

import "strings"

// NormalizePathToOpenAPI converts router parameter syntax to OpenAPI path syntax.
//
// Different HTTP routers use different syntax for path parameters:
//   - Chi, Gin, Echo: /users/:id/posts/:postId
//   - Gorilla Mux: /users/{id}/posts/{postId}
//   - OpenAPI spec: /users/{id}/posts/{postId}
//
// This function normalizes paths from :param syntax (chi/gin/echo) to {param}
// syntax (OpenAPI spec) for automatic API documentation generation.
//
// Conversion rules:
//   - :id → {id}
//   - :userId → {userId}
//   - :post_id → {post_id}
//   - Parameter names can contain: a-z, A-Z, 0-9, _
//
// Examples:
//
//	NormalizePathToOpenAPI("/users/:id")
//	// Returns: "/users/{id}"
//
//	NormalizePathToOpenAPI("/organizations/:orgId/members/:userId")
//	// Returns: "/organizations/{orgId}/members/{userId}"
//
//	NormalizePathToOpenAPI("/static/file.css")
//	// Returns: "/static/file.css" (unchanged, no parameters)
func NormalizePathToOpenAPI(path string) string {
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
