package main

import (
	"database/sql"
	"testing"
)

// newTestDB opens an in-memory SQLite database, applies pragmas, and
// ensures the schema exists. The database is automatically closed when
// the test ends.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
