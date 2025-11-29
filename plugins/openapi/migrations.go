package openapi

// Migrations returns the plugin migrations.
// OpenAPI plugin is stateless and requires no database schema.
func Migrations() []string {
	return []string{}
}
