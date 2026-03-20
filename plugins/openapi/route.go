package openapi

import (
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// Route describes a single API endpoint for OpenAPI documentation.
//
// This is the only struct users and plugins need to construct.
// Pass it to Doc() to register documentation for a route.
//
// Example:
//
//	openapi.Doc(openapi.Route{
//	    Method:  "GET",
//	    Path:    "/api/users/{id}",
//	    Summary: "Get user by ID",
//	    Tags:    []string{"Users"},
//	    Auth:    true,
//	    Params: []openapi.Param{
//	        {Name: "id", In: "path", Type: "string", Required: true},
//	    },
//	    Responses: openapi.Responses{
//	        200: openapi.ResponseOf[UserResponse]("User retrieved"),
//	        404: openapi.Text("User not found"),
//	    },
//	})
type Route struct {
	// Required
	Method string // "GET", "POST", "PUT", "DELETE", "PATCH"
	Path   string // "/api/users/{id}" — use {param} style for path params

	// Optional metadata
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Auth        bool // Adds BearerAuth security requirement when true
	Deprecated  bool

	// Parameters, body, and responses
	Params    []Param
	Body      *BodySchema
	Responses Responses
}

// Param describes a single operation parameter.
//
// When In == "path", Required is forcibly set to true regardless of
// what the caller passes. This is a hard requirement of the OpenAPI 3.0
// spec.
type Param struct {
	Name        string
	In          string // "path", "query", "header", "cookie"
	Description string
	Required    bool     // Always forced to true when In == "path"
	Type        string   // "string", "integer", "boolean", "array"
	Format      string   // "uuid", "date-time", "email", etc.
	Enum        []string
}

// BodySchema holds a generated request body schema.
// Use BodyOf[T]() or RefBody() to construct one.
type BodySchema struct {
	schema *Schema
	name   string // type name for $ref registration
}

// ResponseDef describes a single response entry.
type ResponseDef struct {
	Description string
	schema      *Schema
	name        string // type name for $ref registration
}

// Responses maps HTTP status codes to response definitions.
type Responses map[int]ResponseDef

// ─────────────────────────────────────────────
// Request body constructors
// ─────────────────────────────────────────────

// BodyOf generates a request body schema from a Go struct type via reflection.
// The schema is registered in components/schemas under the struct's type name,
// enabling $ref reuse across endpoints.
//
// Example:
//
//	Body: openapi.BodyOf[CreateUserRequest]()
func BodyOf[T any]() *BodySchema {
	schema, name := resolveSchemaAndName[T]()
	return &BodySchema{schema: schema, name: name}
}

// RefBody creates a request body that references an existing named schema.
// The caller is responsible for registering the referenced schema elsewhere
// (e.g. via addCommonSchemas).
//
// Example:
//
//	Body: openapi.RefBody("LoginRequest")
func RefBody(schemaName string) *BodySchema {
	return &BodySchema{schema: RefSchema(schemaName)}
}

// ─────────────────────────────────────────────
// Response constructors
// ─────────────────────────────────────────────

// ResponseOf generates a response schema from a Go struct type via reflection.
// The schema is registered in components/schemas under the struct's type name,
// enabling $ref reuse across endpoints.
//
// Example:
//
//	200: openapi.ResponseOf[UserResponse]("User retrieved successfully")
func ResponseOf[T any](description string) ResponseDef {
	schema, name := resolveSchemaAndName[T]()
	return ResponseDef{Description: description, schema: schema, name: name}
}

// RefResponse creates a response that references an existing named schema.
//
// Example:
//
//	200: openapi.RefResponse("Success response", "Success")
//	401: openapi.RefResponse("Not authenticated", "Error")
func RefResponse(description string, schemaName string) ResponseDef {
	return ResponseDef{Description: description, schema: RefSchema(schemaName)}
}

// TextResponse is a shorthand for responses with a plain description and no body schema.
//
// Example:
//
//	404: openapi.TextResponse("User not found")
//	204: openapi.TextResponse("Item deleted")
func TextResponse(description string) ResponseDef {
	return ResponseDef{Description: description}
}

// ─────────────────────────────────────────────
// Data response constructors
//
// These wrap the response in the core.Response envelope:
//   { success: bool, message: string, data: T }
// ─────────────────────────────────────────────

// DataResponseOf generates a response that wraps a Go struct in the
// core.Response envelope: { success: bool, message: string, data: T }.
//
// Example:
//
//	200: openapi.DataResponseOf[SessionWithUser]("Login successful")
func DataResponseOf[T any](description string) ResponseDef {
	return buildEnvelopedResponse[T](description, asDirectSchema, "Response")
}

// PaginatedResponseOf generates a plain response schema from a paginated Go
// struct type. Unlike DataResponseOf, this does NOT wrap the response in the
// core.Response envelope, because the paginated type already acts as its own
// envelope via fields like items, limit, offset, page, and totalCount.
//
// Example:
//
//	200: openapi.PaginatedResponseOf[PaginatedResult[User]]("List of users")
func PaginatedResponseOf[T any](description string) ResponseDef {
	schema, name := resolveSchemaAndName[T]()
	return ResponseDef{Description: description, schema: schema, name: name}
}

// DataRefResponse creates a response that wraps an existing named schema
// in the core.Response envelope: { success: bool, message: string, data: $ref }.
//
// Example:
//
//	200: openapi.DataRefResponse("Login successful", "SessionWithUser")
func DataRefResponse(description string, schemaName string) ResponseDef {
	return buildEnvelopedResponseFromSchema(description, RefSchema(schemaName), schemaName+"Response")
}

// ─────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────

// resolveSchemaAndName reflects on T and returns its generated Schema and
// a sanitized component name derived from the type.
func resolveSchemaAndName[T any]() (*Schema, string) {
	var zero T
	t := reflect.TypeOf(zero)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return GenerateSchema(zero), sanitizeComponentName(t)
}

// schemaTransform is a function that transforms a Schema (e.g. wrap in array).
type schemaTransform func(inner *Schema) *Schema

// asDirectSchema returns the schema as-is with no transformation.
func asDirectSchema(s *Schema) *Schema { return s }

// asArraySchema wraps the schema in an array schema.
func asArraySchema(s *Schema) *Schema { return ArraySchema("", s) }

// buildEnvelopedResponse is the shared base for DataResponseOf and
// PaginatedResponseOf. It resolves the schema for T, applies the given
// transform, then delegates to buildEnvelopedResponseFromSchema.
func buildEnvelopedResponse[T any](description string, transform schemaTransform, nameSuffix string) ResponseDef {
	schema, innerName := resolveSchemaAndName[T]()
	return buildEnvelopedResponseFromSchema(description, transform(schema), innerName+nameSuffix)
}

// buildEnvelopedResponseFromSchema wraps a schema in the core.Response envelope
// and returns a ResponseDef registered under the given component name.
func buildEnvelopedResponseFromSchema(description string, dataSchema *Schema, componentName string) ResponseDef {
	envelope := ObjectSchema("", map[string]*Schema{
		"success": BooleanSchema("Request success status"),
		"message": {Type: "string", Description: "Optional message"},
		"data":    dataSchema,
	}, []string{"success"})

	return ResponseDef{
		Description: description,
		schema:      envelope,
		name:        componentName,
	}
}

// sanitizeComponentName converts a Go type name into an OpenAPI-component-safe
// schema name.
//
// Instantiated generics can produce names like:
//
//	"PaginatedResponse[github.com/.../admin.User]"
//
// which contain characters invalid for component schema keys. This function
// flattens them into a clean alphanumeric name like "PaginatedResponseUser".
func sanitizeComponentName(t reflect.Type) string {
	n := t.Name()
	if n == "" {
		n = t.String()
	}

	// Handle single-parameter generic instantiations: Base[Param] -> BaseParam
	if i := strings.Index(n, "["); i != -1 && strings.HasSuffix(n, "]") {
		base := n[:i]
		inside := strings.TrimSuffix(n[i+1:], "]")

		// Split by comma to support future multi-parameter generics.
		paramParts := strings.Split(inside, ",")
		var paramNames []string
		for _, p := range paramParts {
			p = strings.TrimSpace(p)
			// Flatten package paths: a/b/c.Type -> Type
			p = strings.ReplaceAll(p, "/", ".")
			segs := strings.Split(p, ".")
			if len(segs) > 0 && segs[len(segs)-1] != "" {
				paramNames = append(paramNames, segs[len(segs)-1])
			}
		}
		if len(paramNames) > 0 {
			return base + strings.Join(paramNames, "")
		}
	}

	// Fallback: keep alphanumeric, replace everything else with underscore.
	var b strings.Builder
	for _, r := range n {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// ─────────────────────────────────────────────
// Operation ID derivation
// ─────────────────────────────────────────────

// deriveOperationID generates an operationId from the HTTP method and path.
//
// Strategy: strip the mount prefix, title-case each path segment,
// expand {param} segments to By{Param}, prepend the lowercased HTTP method verb.
//
// Examples:
//
//	POST /auth/email/signup  → postEmailSignup
//	GET  /api/users/{id}     → getUserById
//	DELETE /api/users/{id}   → deleteUserById
//	GET  /api/dashboard      → getDashboard
func deriveOperationID(method, path string) string {
	verb := strings.ToLower(method)

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	segments := strings.Split(path, "/")
	var parts []string

	pathParamRegex := regexp.MustCompile(`^\{(.+)\}$`)
	colonParamRegex := regexp.MustCompile(`^:(.+)$`)

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if matches := pathParamRegex.FindStringSubmatch(seg); len(matches) > 1 {
			parts = append(parts, "By"+titleCase(matches[1]))
			continue
		}
		if matches := colonParamRegex.FindStringSubmatch(seg); len(matches) > 1 {
			parts = append(parts, "By"+titleCase(matches[1]))
			continue
		}
		parts = append(parts, titleCase(seg))
	}

	if len(parts) == 0 {
		return verb
	}
	return verb + strings.Join(parts, "")
}

// titleCase capitalizes the first rune of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// normalizePathParams converts :param style path parameters to {param} style
// for OpenAPI compatibility.
func normalizePathParams(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}
