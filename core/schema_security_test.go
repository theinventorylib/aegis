package core

import (
	"strings"
	"testing"
)

func TestValidateTableExistsRejectsInvalidIdentifier(t *testing.T) {
	assertPanics(t, func() {
		ValidateTableExists("users'; DROP TABLE users; --")
	})
}

func TestValidateColumnExistsForDialectRejectsInvalidIdentifier(t *testing.T) {
	assertPanics(t, func() {
		ValidateColumnExistsForDialect(SchemaDialectSQLite, "users", "name'); DROP TABLE users; --")
	})
}

func TestValidateColumnSpecEscapesLiteralValues(t *testing.T) {
	req := ValidateColumnSpec("users", "email", ColumnSpec{
		DataType: "TEXT' OR '1'='1",
	})

	if strings.Contains(req.Query, "LOWER(data_type) = LOWER('TEXT' OR '1'='1')") {
		t.Fatalf("expected injected data type to be escaped, got query %q", req.Query)
	}
	if want := "LOWER(data_type) = LOWER('TEXT'' OR ''1''=''1')"; !strings.Contains(req.Query, want) {
		t.Fatalf("expected escaped data type predicate %q in query %q", want, req.Query)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
