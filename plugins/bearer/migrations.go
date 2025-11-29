package bearer

// Migrations returns the plugin migrations.
// Bearer plugin is stateless and requires no database schema.
func Migrations() []string {
	return []string{}
}
