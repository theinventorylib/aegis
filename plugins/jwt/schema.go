package jwt

import (
	"github.com/theinventorylib/aegis/plugins"
)

// GetSchemaRequirements returns schema validation requirements for the JWT plugin.
//
// This function defines what database objects must exist for the plugin to function.
// The framework uses these requirements to validate the schema before plugin initialization.
//
// Requirements:
//   - Table "jwks" must exist: Stores JSON Web Keys for JWT signing/verification
//
// Validation Timing:
//   - Called during plugin.Init() after migrations are applied
//   - Plugin initialization fails if requirements are not met
//   - Prevents runtime errors from missing database objects
//
// Parameters:
//   - dialect: SQL dialect (DialectPostgres, DialectMySQL, etc.)
//
// Returns:
//   - []plugins.SchemaRequirement: List of validation checks to perform
//   - Empty slice if dialect is not supported
//
// Example:
//
//	reqs := GetSchemaRequirements(plugins.DialectPostgres)
//	for _, req := range reqs {
//	    if err := req.Validate(db); err != nil {
//	        log.Fatal("Schema validation failed:", err)
//	    }
//	}
func GetSchemaRequirements(dialect plugins.Dialect) []plugins.SchemaRequirement {
	switch dialect {
	case plugins.DialectPostgres, plugins.DialectMySQL:
		return []plugins.SchemaRequirement{
			// Table existence check for JWKS table
			plugins.ValidateTableExists("jwks"),
		}
	default:
		return []plugins.SchemaRequirement{}
	}
}
