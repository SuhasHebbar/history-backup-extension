package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const initPragmas = `
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA cache_size = -20000;
PRAGMA foreign_keys = ON;
PRAGMA auto_vacuum = INCREMENTAL;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 2147483648;
PRAGMA page_size = 8192;
`

const createTableSQL = `
CREATE TABLE history_items (
    device_name     TEXT    NOT NULL,
    url             TEXT    NOT NULL,
    title           TEXT,
    last_visit_time INTEGER,
    visit_count     INTEGER,
    typed_count     INTEGER,
    uploaded_at     INTEGER NOT NULL,
    PRIMARY KEY (device_name, url)
);
`

const createUploadEventsSQL = `
CREATE TABLE upload_events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp        INTEGER NOT NULL,
    device_name      TEXT    NOT NULL,
    num_items        INTEGER NOT NULL,
    range_start_time INTEGER NOT NULL,
    range_end_time   INTEGER NOT NULL
);
`

const upsertSQL = `
INSERT INTO history_items
    (device_name, url, title, last_visit_time, visit_count, typed_count, uploaded_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(device_name, url) DO UPDATE SET
    title           = excluded.title,
    last_visit_time = excluded.last_visit_time,
    visit_count     = excluded.visit_count,
    typed_count     = excluded.typed_count,
    uploaded_at     = excluded.uploaded_at;
`

const insertUploadEventSQL = `
INSERT INTO upload_events (timestamp, device_name, num_items, range_start_time, range_end_time)
VALUES (?, ?, ?, ?, ?);
`

// expectedColumns defines the required schema for the history_items table.
// Keys are column names; values are declared SQLite types (uppercased for comparison).
var expectedColumns = map[string]string{
	"device_name":     "TEXT",
	"url":             "TEXT",
	"title":           "TEXT",
	"last_visit_time": "INTEGER",
	"visit_count":     "INTEGER",
	"typed_count":     "INTEGER",
	"uploaded_at":     "INTEGER",
}

var expectedUploadEventColumns = map[string]string{
	"id":               "INTEGER",
	"timestamp":        "INTEGER",
	"device_name":      "TEXT",
	"num_items":        "INTEGER",
	"range_start_time": "INTEGER",
	"range_end_time":   "INTEGER",
}

// openDB opens (or creates) the SQLite database at path, applies initialization
// pragmas, and ensures the schema exists and is valid.
func openDB(path string) (*sql.DB, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}

	// txlock sets the transaction type on the connection to BEGIN IMMEDIATE.
	db, err := sql.Open("sqlite3", path+sep+"_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec(initPragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply pragmas: %w", err)
	}

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// ensureSchema creates required tables if they don't exist, or validates them
// if they do.
func ensureSchema(db *sql.DB) error {
	if err := ensureTable(db, "history_items", createTableSQL, expectedColumns); err != nil {
		return err
	}
	return ensureTable(db, "upload_events", createUploadEventsSQL, expectedUploadEventColumns)
}

// ensureTable creates the named table if absent, or validates its columns if present.
func ensureTable(db *sql.DB, table, createSQL string, expected map[string]string) error {
	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&name)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := db.Exec(createSQL); err != nil {
			return fmt.Errorf("create %s table: %w", table, err)
		}
		log.Printf("Created %s table", table)
		return nil
	case err != nil:
		return fmt.Errorf("check %s table existence: %w", table, err)
	default:
		return validateTableSchema(db, table, expected)
	}
}

// validateTableSchema checks that the named table has exactly the expected
// columns with the expected declared types.
func validateTableSchema(db *sql.DB, table string, expected map[string]string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer rows.Close()

	type colInfo struct {
		cid          int
		name         string
		declType     string
		notNull      int
		defaultValue sql.NullString
		pk           int
	}

	found := make(map[string]string)
	for rows.Next() {
		var c colInfo
		if err := rows.Scan(&c.cid, &c.name, &c.declType, &c.notNull, &c.defaultValue, &c.pk); err != nil {
			return fmt.Errorf("scan table_info row: %w", err)
		}
		found[c.name] = c.declType
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info: %w", err)
	}

	for col, wantType := range expected {
		gotType, ok := found[col]
		if !ok {
			return fmt.Errorf("schema mismatch: column %q not found in %s", col, table)
		}
		if gotType != wantType {
			return fmt.Errorf("schema mismatch: column %q has type %q, want %q", col, gotType, wantType)
		}
	}

	for col := range found {
		if _, ok := expected[col]; !ok {
			return fmt.Errorf("schema mismatch: unexpected column %q found in %s", col, table)
		}
	}

	log.Printf("Schema validation passed for %s", table)
	return nil
}
