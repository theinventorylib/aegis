package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// MigrationRegistry tracks which plugin owns which (table, column) and
// which plugin created which table, so two plugins extending the same
// shared core table or creating a table with the same name fail at
// registration time instead of at first query.
//
// This is a structural conflict detector — it does not run plugin SQL.
// It scans embedded migration text with conservative regexes, looking
// only for unambiguous CREATE TABLE / ALTER TABLE ... ADD COLUMN
// statements. Anything we cannot parse is silently skipped: false
// negatives are acceptable, false positives are not.
//
// Aegis instantiates a single MigrationRegistry per process and calls
// Register for each plugin during Use; tests and tooling can construct
// their own via NewMigrationRegistry.
type MigrationRegistry struct {
	mu sync.Mutex
	// columns maps "table.column" -> owning plugin name
	columns map[string]string
	// tables maps table name -> owning plugin name
	tables map[string]string
}

// NewMigrationRegistry returns an empty registry.
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		columns: make(map[string]string),
		tables:  make(map[string]string),
	}
}

// Pre-compiled at package load. Both patterns are case-insensitive and
// only match the leading keywords of a statement; quoted identifiers
// (`"foo"` or "`foo`") are tolerated.
var (
	reCreateTable = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["` + "`" + `]?([a-zA-Z_][a-zA-Z0-9_]*)["` + "`" + `]?`)
	reAddColumn   = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+["` + "`" + `]?([a-zA-Z_][a-zA-Z0-9_]*)["` + "`" + `]?[\s\S]*?\bADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?["` + "`" + `]?([a-zA-Z_][a-zA-Z0-9_]*)["` + "`" + `]?`)
)

// scanMigration extracts (table, column) ownership claims from a single
// migration's Up SQL. It splits on semicolons so an unparseable
// statement does not poison the rest of the migration.
func scanMigration(up string) (tables []string, columns []string) {
	stmts := strings.Split(up, ";")
	for _, stmt := range stmts {
		if m := reCreateTable.FindStringSubmatch(stmt); m != nil {
			tables = append(tables, strings.ToLower(m[1]))
		}
		// ADD COLUMN: an ALTER TABLE statement may add multiple columns
		// in one statement; FindAllStringSubmatch picks them up.
		for _, m := range reAddColumn.FindAllStringSubmatch(stmt, -1) {
			columns = append(columns, strings.ToLower(m[1])+"."+strings.ToLower(m[2]))
		}
	}
	return tables, columns
}

// Register records ownership claims for a plugin's migrations and
// returns an error if any claim collides with one already made by a
// different plugin. On collision the registry is not modified (the
// claims for this plugin are committed all-or-nothing), so the caller
// can abort plugin registration cleanly.
func (mr *MigrationRegistry) Register(pluginName string, migrations []Migration) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	type claim struct {
		kind  string // "table" or "column"
		key   string
		owner string
	}
	var pending []claim

	for _, m := range migrations {
		tables, columns := scanMigration(m.Up)
		for _, t := range tables {
			if owner, ok := mr.tables[t]; ok && owner != pluginName {
				return fmt.Errorf("migration conflict: plugin %q tries to CREATE TABLE %q which is already owned by plugin %q", pluginName, t, owner)
			}
			pending = append(pending, claim{kind: "table", key: t, owner: pluginName})
		}
		for _, c := range columns {
			if owner, ok := mr.columns[c]; ok && owner != pluginName {
				return fmt.Errorf("migration conflict: plugin %q tries to ADD COLUMN %s which was already added by plugin %q", pluginName, c, owner)
			}
			pending = append(pending, claim{kind: "column", key: c, owner: pluginName})
		}
	}

	for _, p := range pending {
		switch p.kind {
		case "table":
			mr.tables[p.key] = p.owner
		case "column":
			mr.columns[p.key] = p.owner
		}
	}
	return nil
}
