package server

import "strings"

// NormalizePathToOpenAPI converts router path syntax to OpenAPI syntax.
// Router paths use :param syntax (e.g., "/users/:id/posts/:postId")
// OpenAPI paths use {param} syntax (e.g., "/users/{id}/posts/{postId}")
//
// Example:
//
//	NormalizePathToOpenAPI("/organizations/:id/members/:userId")
//	// Returns: "/organizations/{id}/members/{userId}"
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

// isParamChar returns true if the character is valid in a parameter name.
func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}
