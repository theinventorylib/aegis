package openapi

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Regular expressions for parsing validation tags
var (
	lengthRegex = regexp.MustCompile(`Length\((\d+),\s*(\d+)\)`)
	matchRegex  = regexp.MustCompile(`Match\(([^)]+)\)`)
	inRegex     = regexp.MustCompile(`In\(([^)]+)\)`)
	minMaxRegex = regexp.MustCompile(`(Min|Max)\((\d+)\)`)
)

// GenerateSchema creates an OpenAPI schema from a Go struct instance.
func GenerateSchema(v interface{}) *Schema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return generateSchemaType(t)
}

func generateSchemaType(t reflect.Type) *Schema {
	switch t.Kind() {
	case reflect.Struct:
		// Handle special types
		if t == reflect.TypeOf(time.Time{}) {
			return DateTimeSchema("")
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
				if !strings.Contains(jsonTag, "omitempty") && field.Type.Kind() != reflect.Ptr {
					required = append(required, name)
				}
			}

			properties[name] = fieldSchema
		}

		return ObjectSchema("", properties, required)

	case reflect.Slice, reflect.Array:
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

	default:
		return StringSchema("") // Fallback
	}
}

// applyValidationConstraints parses validation tags and applies constraints to the schema.
func applyValidationConstraints(schema *Schema, validationTag string, _ reflect.Type) {
	// Length constraints: validation:"Length(3,50)"
	if matches := lengthRegex.FindStringSubmatch(validationTag); matches != nil {
		minLen, _ := strconv.Atoi(matches[1])
		maxLen, _ := strconv.Atoi(matches[2])

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
		value, _ := strconv.Atoi(match[2])

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
		schema.Enum = make([]interface{}, len(values))
		for i, v := range values {
			// Trim quotes and whitespace
			cleaned := strings.Trim(strings.TrimSpace(v), "\"'")
			schema.Enum[i] = cleaned
		}
	}
}
