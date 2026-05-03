package core

import (
	"fmt"
	"regexp"
)

// sqlIdentifierPattern matches a syntactically valid SQL identifier:
// must start with a letter or underscore and contain only letters,
// digits, or underscores. This intentionally rejects quoted identifiers,
// dots (schema.table), and any character that could be used to break
// out of a quoted literal in a Sprintf'd query.
var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SanitizeSQLIdentifier validates that name is a safe SQL identifier
// (table or column name) and returns it unchanged. It panics if name
// contains any character outside of [A-Za-z0-9_] or does not start with
// a letter / underscore.
//
// This helper exists because the schema-validation requirement helpers
// (ValidateTableExists, ValidateColumnExists, …) build their probe
// queries via fmt.Sprintf — there is no portable way to parameterise an
// identifier across information_schema, sqlite_master, and pragma_*.
// Identifiers come from compile-time plugin code, never from user
// input, so a panic on an invalid identifier is the correct
// fail-fast behavior: it surfaces the bug at startup instead of
// silently producing a malformed (or worse, injectable) query.
func SanitizeSQLIdentifier(name string) string {
	if !sqlIdentifierPattern.MatchString(name) {
		panic(fmt.Sprintf("aegis: invalid SQL identifier %q: must match [A-Za-z_][A-Za-z0-9_]*", name))
	}
	return name
}

// sanitizeSQLLiteral escapes a string value for safe interpolation
// inside a single-quoted SQL literal. It doubles any embedded single
// quotes per the SQL standard. This is used for the small set of
// non-identifier values that the schema validators interpolate
// (data type names like "TEXT", nullability flags, etc.) — values
// which are also plugin-controlled but not regex-restricted.
func sanitizeSQLLiteral(value string) string {
	out := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '\'' {
			out = append(out, '\'', '\'')
			continue
		}
		// Reject NULs outright — they are never legitimate inside an
		// SQL literal and several drivers truncate at the first NUL.
		if c == 0 {
			panic("aegis: NUL byte in SQL literal")
		}
		out = append(out, c)
	}
	return string(out)
}
