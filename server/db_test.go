package main

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// openDB tests
// ---------------------------------------------------------------------------

func TestOpenDB_CreatesDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := openDB(path)
	if err != nil {
		t.Fatalf("openDB: unexpected error: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"history_items", "upload_events"} {
		var name string
		err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("%s table not found: %v", table, err)
		}
		if name != table {
			t.Errorf("expected table name %q, got %q", table, name)
		}
	}
}

func TestOpenDB_InvalidPath(t *testing.T) {
	_, err := openDB("/nonexistent/dir/db.sqlite")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

// ---------------------------------------------------------------------------
// ensureSchema tests
// ---------------------------------------------------------------------------

func TestEnsureSchema_CreatesTable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := ensureSchema(db); err != nil {
		t.Fatalf("ensureSchema: unexpected error: %v", err)
	}

	for _, table := range []string{"history_items", "upload_events"} {
		var name string
		err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("%s table not found after ensureSchema: %v", table, err)
		}
	}
}

func TestEnsureSchema_ValidatesExistingSchema(t *testing.T) {
	db := newTestDB(t) // table already created by openDB

	// Call ensureSchema again — should be idempotent.
	if err := ensureSchema(db); err != nil {
		t.Fatalf("ensureSchema (second call): unexpected error: %v", err)
	}
}

func TestEnsureSchema_MissingColumn(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create history_items without typed_count.
	_, err = db.Exec(`
		CREATE TABLE history_items (
			device_name     TEXT    NOT NULL,
			url             TEXT    NOT NULL,
			title           TEXT,
			last_visit_time REAL,
			visit_count     INTEGER,
			uploaded_at     INTEGER NOT NULL,
			PRIMARY KEY (device_name, url)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := ensureSchema(db); err == nil {
		t.Fatal("expected error for missing column typed_count, got nil")
	}
}

func TestEnsureSchema_WrongColumnType(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create history_items with url as INTEGER instead of TEXT.
	_, err = db.Exec(`
		CREATE TABLE history_items (
			device_name     TEXT    NOT NULL,
			url             INTEGER NOT NULL,
			title           TEXT,
			last_visit_time REAL,
			visit_count     INTEGER,
			typed_count     INTEGER,
			uploaded_at     INTEGER NOT NULL,
			PRIMARY KEY (device_name, url)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := ensureSchema(db); err == nil {
		t.Fatal("expected error for wrong column type (url INTEGER vs TEXT), got nil")
	}
}

// ---------------------------------------------------------------------------
// validateTableSchema tests
// ---------------------------------------------------------------------------

func TestValidateTableSchema_Pass(t *testing.T) {
	db := newTestDB(t)

	if err := validateTableSchema(db, "history_items", expectedColumns); err != nil {
		t.Fatalf("validateTableSchema: unexpected error: %v", err)
	}
}

func TestValidateTableSchema_MissingColumn(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create history_items without visit_count.
	_, err = db.Exec(`
		CREATE TABLE history_items (
			device_name     TEXT    NOT NULL,
			url             TEXT    NOT NULL,
			title           TEXT,
			last_visit_time REAL,
			typed_count     INTEGER,
			uploaded_at     INTEGER NOT NULL,
			PRIMARY KEY (device_name, url)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = validateTableSchema(db, "history_items", expectedColumns)
	if err == nil {
		t.Fatal("expected error for missing column visit_count, got nil")
	}
	if !strings.Contains(err.Error(), "visit_count") {
		t.Errorf("error should mention missing column, got: %v", err)
	}
}

func TestValidateTableSchema_WrongType(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create history_items with title as INTEGER instead of TEXT.
	_, err = db.Exec(`
		CREATE TABLE history_items (
			device_name     TEXT    NOT NULL,
			url             TEXT    NOT NULL,
			title           INTEGER,
			last_visit_time REAL,
			visit_count     INTEGER,
			typed_count     INTEGER,
			uploaded_at     INTEGER NOT NULL,
			PRIMARY KEY (device_name, url)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = validateTableSchema(db, "history_items", expectedColumns)
	if err == nil {
		t.Fatal("expected error for wrong column type (title INTEGER vs TEXT), got nil")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error should mention column title, got: %v", err)
	}
}

func TestValidateTableSchema_ExtraColumn(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE history_items (
			device_name     TEXT    NOT NULL,
			url             TEXT    NOT NULL,
			title           TEXT,
			last_visit_time REAL,
			visit_count     INTEGER,
			typed_count     INTEGER,
			uploaded_at     INTEGER NOT NULL,
			extra_column    TEXT,
			PRIMARY KEY (device_name, url)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = validateTableSchema(db, "history_items", expectedColumns)
	if err == nil {
		t.Fatal("expected error for extra column, got nil")
	}
	if !strings.Contains(err.Error(), "extra_column") {
		t.Errorf("error should mention extra column, got: %v", err)
	}
}
