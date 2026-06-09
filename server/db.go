package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS history_items (
    device_name     TEXT    NOT NULL,
    url             TEXT    NOT NULL,
    title           TEXT,
    last_visit_time REAL,
    visit_count     INTEGER,
    typed_count     INTEGER,
    uploaded_at     INTEGER NOT NULL,
    PRIMARY KEY (url, device_name)
);
`

const createUploadEventsSQL = `
CREATE TABLE IF NOT EXISTS upload_events (
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

const createLastVisitTimeIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_history_items_last_visit_time
    ON history_items (last_visit_time);
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
	"last_visit_time": "REAL",
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
	db, err := sql.Open("sqlite3_pragma", path+sep+"_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// ensureSchema creates required tables if they don't exist, or validates them
// if they do, then ensures indexes are present and correct.
func ensureSchema(db *sql.DB) error {
	if err := ensureTable(db, "history_items", createTableSQL, expectedColumns); err != nil {
		return err
	}
	if err := ensureTable(db, "upload_events", createUploadEventsSQL, expectedUploadEventColumns); err != nil {
		return err
	}
	return ensureIndex(db,
		"idx_history_items_last_visit_time",
		createLastVisitTimeIndexSQL,
		[]string{"last_visit_time"},
	)
}

// ensureIndex creates the named index if absent, then verifies it covers exactly
// the expected columns in the given order.
func ensureIndex(db *sql.DB, indexName, createSQL string, wantCols []string) error {
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("create index %s: %w", indexName, err)
	}

	rows, err := db.Query(`PRAGMA index_info(` + indexName + `)`)
	if err != nil {
		return fmt.Errorf("pragma index_info(%s): %w", indexName, err)
	}
	defer rows.Close()

	var gotCols []string
	for rows.Next() {
		var seqno, cid int
		var name string
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return fmt.Errorf("scan index_info row: %w", err)
		}
		gotCols = append(gotCols, name)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate index_info: %w", err)
	}

	if len(gotCols) != len(wantCols) {
		return fmt.Errorf("index %s: got columns %v, want %v", indexName, gotCols, wantCols)
	}
	for i, col := range wantCols {
		if gotCols[i] != col {
			return fmt.Errorf("index %s: column[%d] is %q, want %q", indexName, i, gotCols[i], col)
		}
	}

	log.Printf("Ensured index %s", indexName)
	return nil
}

func ensureTable(db *sql.DB, table, createSQL string, expected map[string]string) error {
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("create %s table: %w", table, err)
	}
	return validateTableSchema(db, table, expected)
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
