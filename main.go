package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// SQL constants
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Payload types
// ---------------------------------------------------------------------------

// HistoryItem mirrors the Chrome HistoryItem fields from the extension payload.
// All fields except ID are optional per the Chrome API.
type HistoryItem struct {
	ID            string   `json:"id"`
	URL           string   `json:"url,omitempty"`
	Title         string   `json:"title,omitempty"`
	LastVisitTime *float64 `json:"lastVisitTime,omitempty"`
	VisitCount    *int     `json:"visitCount,omitempty"`
	TypedCount    *int     `json:"typedCount,omitempty"`
}

// UploadPayload is the top-level JSON body sent by the extension.
type UploadPayload struct {
	UploadedAt     int64         `json:"uploadedAt"`
	DeviceName     string        `json:"deviceName"`
	RangeStartTime int64         `json:"rangeStartTime"`
	RangeEndTime   int64         `json:"rangeEndTime"`
	Items          []HistoryItem `json:"items"`
}

// ---------------------------------------------------------------------------
// Database helpers
// ---------------------------------------------------------------------------

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

// openDB opens (or creates) the SQLite database at path, applies initialization
// pragmas, and ensures the schema exists and is valid.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Apply pragmas immediately after opening.
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

// ensureSchema creates the history_items table if it doesn't exist, or
// validates it if it does.
func ensureSchema(db *sql.DB) error {
	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='history_items'`,
	).Scan(&name)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Table does not exist — create it.
		if _, err := db.Exec(createTableSQL); err != nil {
			return fmt.Errorf("create history_items table: %w", err)
		}
		log.Println("Created history_items table")
		return nil
	case err != nil:
		return fmt.Errorf("check table existence: %w", err)
	default:
		// Table exists — validate its schema.
		return validateSchema(db)
	}
}

// validateSchema checks that history_items has exactly the expected columns
// with the expected declared types.
func validateSchema(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(history_items)`)
	if err != nil {
		return fmt.Errorf("pragma table_info: %w", err)
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

	found := make(map[string]string) // name → declared type
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

	for col, wantType := range expectedColumns {
		gotType, ok := found[col]
		if !ok {
			return fmt.Errorf("schema mismatch: column %q not found in history_items", col)
		}
		if gotType != wantType {
			return fmt.Errorf("schema mismatch: column %q has type %q, want %q", col, gotType, wantType)
		}
	}

	log.Println("Schema validation passed")
	return nil
}

// ---------------------------------------------------------------------------
// HTTP handler
// ---------------------------------------------------------------------------

// handler returns an http.HandlerFunc that parses upload payloads and persists
// history items to the SQLite database.
func handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR read body: %v", err)
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}

		var payload UploadPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("ERROR parse JSON: %v", err)
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if payload.DeviceName == "" {
			http.Error(w, "missing required field: deviceName", http.StatusBadRequest)
			return
		}
		if payload.UploadedAt == 0 {
			http.Error(w, "missing required field: uploadedAt", http.StatusBadRequest)
			return
		}
		if len(payload.Items) == 0 {
			http.Error(w, "items array must not be empty", http.StatusBadRequest)
			return
		}

		if err := persistItems(db, &payload); err != nil {
			log.Printf("ERROR persist items: %v", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		elapsed := time.Since(start)
		log.Printf("Stored %d item(s) from device=%q in %s", len(payload.Items), payload.DeviceName, elapsed)

		w.WriteHeader(http.StatusNoContent)
	}
}

// persistItems upserts all history items from the payload in a single
// BEGIN IMMEDIATE transaction.
func persistItems(db *sql.DB, payload *UploadPayload) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Upgrade to an IMMEDIATE lock to avoid writer contention.
	if _, err := tx.Exec("SAVEPOINT _; ROLLBACK TO _; RELEASE _;"); err != nil {
		// Fallback: just proceed; the BEGIN IMMEDIATE hint is a best-effort.
	}
	// Actually request IMMEDIATE explicitly by reopening via raw exec.
	// db.Begin() uses BEGIN DEFERRED by default; to use BEGIN IMMEDIATE we
	// must exec it directly via a raw connection. We accomplish this by
	// rolling back the deferred tx and using Exec on the db directly.
	tx.Rollback() //nolint:errcheck

	// Use BEGIN IMMEDIATE directly.
	if _, err := db.Exec("BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin immediate: %w", err)
	}

	// Wrap everything in a cleanup that commits or rolls back.
	committed := false
	defer func() {
		if !committed {
			db.Exec("ROLLBACK") //nolint:errcheck
		}
	}()

	stmt, err := db.Prepare(upsertSQL)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	skipped := 0
	for _, item := range payload.Items {
		if item.URL == "" {
			skipped++
			continue
		}
		_, err := stmt.Exec(
			payload.DeviceName,
			item.URL,
			nullableString(item.Title),
			nullableFloat64(item.LastVisitTime),
			nullableInt(item.VisitCount),
			nullableInt(item.TypedCount),
			payload.UploadedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert item url=%q: %w", item.URL, err)
		}
	}

	if _, err := db.Exec("COMMIT"); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true

	if skipped > 0 {
		log.Printf("Skipped %d item(s) with missing URL", skipped)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Nullable helpers
// ---------------------------------------------------------------------------

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

func nullableInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds values that can be supplied via a JSON config file.
type Config struct {
	Addr    string `json:"addr"`
	WorkDir string `json:"working-directory"`
}

// loadConfig reads and parses a JSON config file at path.
// If path is empty it returns an empty Config without error.
func loadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

// Entry point
// ---------------------------------------------------------------------------

func main() {
	configPath := flag.String("config", "", "path to JSON config file (optional)")
	addr := flag.String("addr", "", "listen address (host:port) — overrides config file")
	workDir := flag.String("working-directory", "", "path to working directory for the SQLite database — overrides config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// CLI flags take precedence; fall back to config file, then built-in default.
	effectiveAddr := *addr
	if effectiveAddr == "" {
		effectiveAddr = cfg.Addr
	}
	if effectiveAddr == "" {
		effectiveAddr = ":8080"
	}

	effectiveWorkDir := *workDir
	if effectiveWorkDir == "" {
		effectiveWorkDir = cfg.WorkDir
	}

	if effectiveWorkDir == "" {
		fmt.Fprintln(os.Stderr, "error: --working-directory is required (via flag or config file)")
		flag.Usage()
		os.Exit(1)
	}

	if err := os.MkdirAll(effectiveWorkDir, 0755); err != nil {
		log.Fatalf("create working directory %q: %v", effectiveWorkDir, err)
	}

	dbPath := filepath.Join(effectiveWorkDir, "history.db")
	log.Printf("Opening database at %s", dbPath)

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/", handler(db))

	log.Printf("Listening on %s", effectiveAddr)
	if err := http.ListenAndServe(effectiveAddr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
