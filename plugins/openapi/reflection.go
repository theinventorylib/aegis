package openapi

import (
	"encoding/json"
	"maps"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Regular expressions for parsing validation tags.
//
// These patterns extract validation constraints from Go struct tags:
//   - lengthRegex: Length(3,50) -> min=3, max=50
//   - matchRegex: Match(^[a-z]+$) -> pattern=^[a-z]+$
//   - inRegex: In(active,inactive) -> enum=[active,inactive]
//   - minMaxRegex: Min(1), Max(100) -> minimum=1, maximum=100
var (
	lengthRegex = regexp.MustCompile(`Length\((\d+),\s*(\d+)\)`)
	matchRegex  = regexp.MustCompile(`Match\(([^)]+)\)`)
	inRegex     = regexp.MustCompile(`In\(([^)]+)\)`)
	minMaxRegex = regexp.MustCompile(`(Min|Max)\((\d+)\)`)
)

// GenerateSchema creates an OpenAPI schema from a Go struct instance using reflection.
//
// This function analyzes the struct's type information to automatically generate
// a JSON Schema compatible with OpenAPI 3.0.
//
// Schema Generation:
//  1. Inspect struct fields and types
//  2. Parse JSON tags for field names
//  3. Parse validation tags for constraints
//  4. Determine required fields (non-omitempty, validation:"Required")
//  5. Generate nested schemas for complex types
//
// Supported Types:
//   - Primitives: string, int, bool, float
//   - Complex: struct, array, map
//   - Special: time.Time (date-time format)
//
// Validation Tags:
//   - validation:"Required" -> required field
//   - validation:"Length(3,50)" -> minLength/maxLength
//   - validation:"Min(1),Max(100)" -> minimum/maximum
//   - validation:"Match(^[a-z]+$)" -> pattern
//
// Parameters:
//   - v: Go struct instance or pointer to struct
//
// Returns:
//   - *Schema: OpenAPI schema with properties and constraints
//
// Example:
//
//	type User struct {
//	  Name  string `json:"name" validation:"Required,Length(1,100)"`
//	  Email string `json:"email" validation:"Required,Match(^.+@.+$)"`
//	}
//	schema := GenerateSchema(User{})
func GenerateSchema(v any) *Schema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return generateSchemaType(t)
}

func generateSchemaType(t reflect.Type) *Schema {
	// Handle pointers by dereferencing them recursively
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		// Handle special types
		if t == reflect.TypeFor[time.Time]() {
			return DateTimeSchema("")
		}

		// Handle json.RawMessage as an object
		if t == reflect.TypeFor[json.RawMessage]() {
			return &Schema{
				Type:        "object",
				Description: "Arbitrary JSON data",
			}
		}

		properties := make(map[string]*Schema)
		required := []string{}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Skip unexported fields
			if field.PkgPath != "" {
				continue
			}

			// Get JSON tag
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			name := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				name = parts[0]
			}

			// Handle embedded fields (flattened if no name in json tag)
			if field.Anonymous && (jsonTag == "" || name == "") {
				embeddedSchema := generateSchemaType(field.Type)
				if embeddedSchema.Type == "object" {
					maps.Copy(properties, embeddedSchema.Properties)
					required = append(required, embeddedSchema.Required...)
					continue
				}
			}

			// If name is still empty (e.g., json:",omitempty"), use field name
			if name == "" {
				name = field.Name
			}

			// Generate base schema for field
			fieldSchema := generateSchemaType(field.Type)

			// Parse validation tags and apply constraints
			validationTag := field.Tag.Get("validation")
			if validationTag != "" {
				applyValidationConstraints(fieldSchema, validationTag, field.Type)

				// Check if field is required
				if strings.Contains(validationTag, "required") || strings.Contains(validationTag, "Required") {
					required = append(required, name)
				}
			} else {
				// Fallback: check omitempty for required fields
				// Pointers and slices are typically optional unless marked required
				if !strings.Contains(jsonTag, "omitempty") &&
					field.Type.Kind() != reflect.Pointer &&
					field.Type.Kind() != reflect.Slice &&
					field.Type.Kind() != reflect.Map {
					required = append(required, name)
				}
			}

			properties[name] = fieldSchema
		}

		return ObjectSchema("", properties, required)

	case reflect.Slice, reflect.Array:
		// Special case for []byte which might not be json.RawMessage but we often want as string
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: "string", Description: "Base64 encoded bytes"}
		}
		return ArraySchema("", generateSchemaType(t.Elem()))

	case reflect.String:
		return StringSchema("")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return IntegerSchema("")

	case reflect.Bool:
		return BooleanSchema("")

	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}

	case reflect.Map:
		return &Schema{
			Type:                 "object",
			AdditionalProperties: generateSchemaType(t.Elem()),
		}

	case reflect.Interface:
		return &Schema{Type: "object", Description: "Any value"}

	default:
		return &Schema{Type: "object"} // Fallback for unknown types
	}
}

// applyValidationConstraints parses validation tags and applies constraints to the schema.
func applyValidationConstraints(schema *Schema, validationTag string, _ reflect.Type) {
	// Length constraints: validation:"Length(3,50)"
	if matches := lengthRegex.FindStringSubmatch(validationTag); matches != nil {
		minLen, err := strconv.Atoi(matches[1])
		_ = err
		maxLen, err := strconv.Atoi(matches[2])
		_ = err

		switch schema.Type {
		case "string":
			schema.MinLength = &minLen
			schema.MaxLength = &maxLen
		case "array":
			schema.MinItems = &minLen
			schema.MaxItems = &maxLen
		}
	}

	// Min/Max constraints: validation:"Min(1)" or validation:"Max(100)"
	for _, match := range minMaxRegex.FindAllStringSubmatch(validationTag, -1) {
		constraint := match[1] // "Min" or "Max"
		value, err := strconv.Atoi(match[2])
		_ = err

		if schema.Type == "integer" || schema.Type == "number" {
			floatVal := float64(value)
			switch constraint {
			case "Min":
				schema.Minimum = &floatVal
			case "Max":
				schema.Maximum = &floatVal
			}
		}
	}

	// Pattern constraint: validation:"Match(^[a-z0-9-]+$)"
	if matches := matchRegex.FindStringSubmatch(validationTag); matches != nil {
		if schema.Type == "string" {
			// Remove quotes if present
			pattern := strings.Trim(matches[1], "\"'")
			schema.Pattern = pattern
		}
	}

	// Enum constraint: validation:"In(admin,member)"
	if matches := inRegex.FindStringSubmatch(validationTag); matches != nil {
		values := strings.Split(matches[1], ",")
		schema.Enum = make([]any, len(values))
		for i, v := range values {
			// Trim quotes and whitespace
			cleaned := strings.Trim(strings.TrimSpace(v), "\"'")
			schema.Enum[i] = cleaned
		}
	}
}
