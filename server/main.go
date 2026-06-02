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
	"strings"
	"time"

	"github.com/rs/cors"

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

func statusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

// persistItems upserts all history items from the payload in a single
// BEGIN IMMEDIATE transaction.
func persistItems(db *sql.DB, payload *UploadPayload) error {
	tx, err := db.Begin() // issues BEGIN IMMEDIATE via _txlock=immediate DSN param
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(upsertSQL)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	skipped := 0
	itemCount := 0
	for _, item := range payload.Items {
		if item.URL == "" {
			skipped++
			continue
		}
		if _, err := stmt.Exec(
			payload.DeviceName,
			item.URL,
			nullableString(item.Title),
			nullableFloat64(item.LastVisitTime),
			nullableInt(item.VisitCount),
			nullableInt(item.TypedCount),
			payload.UploadedAt,
		); err != nil {
			return fmt.Errorf("upsert item url=%q: %w", item.URL, err)
		}
		itemCount++
	}

	if skipped > 0 {
		log.Printf("Skipped %d item(s) with missing URL", skipped)
	}

	if _, err := tx.Exec(insertUploadEventSQL,
		payload.UploadedAt,
		payload.DeviceName,
		itemCount,
		payload.RangeStartTime,
		payload.RangeEndTime,
	); err != nil {
		return fmt.Errorf("insert upload event: %w", err)
	}

	return tx.Commit()
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
	Addr           string   `json:"addr"`
	WorkDir        string   `json:"working-directory"`
	AllowedOrigins []string `json:"allowed-origins"`
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

// resolveSettings merges CLI flag values with config-file values.
// CLI values take precedence; the built-in default for addr is ":8080".
// It returns an error when workDir cannot be determined.
func resolveSettings(cliAddr, cliWorkDir, cliAllowedOrigins string, cfg Config) (addr, workDir string, allowedOrigins []string, err error) {
	addr = cliAddr
	if addr == "" {
		addr = cfg.Addr
	}
	if addr == "" {
		addr = ":8080"
	}

	workDir = cliWorkDir
	if workDir == "" {
		workDir = cfg.WorkDir
	}
	if workDir == "" {
		return "", "", nil, fmt.Errorf("--working-directory is required (via flag or config file)")
	}

	if cliAllowedOrigins != "" {
		allowedOrigins = strings.Split(cliAllowedOrigins, ",")
	} else {
		allowedOrigins = cfg.AllowedOrigins
	}

	return addr, workDir, allowedOrigins, nil
}

func finalConfigJSON(addr, workDir string, allowedOrigins []string) (string, error) {
	if allowedOrigins == nil {
		allowedOrigins = []string{}
	}

	b, err := json.Marshal(Config{
		Addr:           addr,
		WorkDir:        workDir,
		AllowedOrigins: allowedOrigins,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func buildHandler(db *sql.DB, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler())
	mux.HandleFunc("/", handler(db))
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
	})
	return c.Handler(mux)
}

// Entry point
// ---------------------------------------------------------------------------

func main() {
	configPath := flag.String("config", "", "path to JSON config file (optional)")
	addr := flag.String("addr", "", "listen address (host:port) — overrides config file")
	workDir := flag.String("working-directory", "", "path to working directory for the SQLite database — overrides config file")
	allowedOrigins := flag.String("allowed-origins", "", "comma-separated list of allowed CORS origins — overrides config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	effectiveAddr, effectiveWorkDir, effectiveAllowedOrigins, err := resolveSettings(*addr, *workDir, *allowedOrigins, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(1)
	}

	finalConfig, err := finalConfigJSON(effectiveAddr, effectiveWorkDir, effectiveAllowedOrigins)
	if err != nil {
		log.Fatalf("marshal final config: %v", err)
	}
	log.Printf("Final config: %s", finalConfig)

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

	log.Printf("Listening on %s", effectiveAddr)
	if err := http.ListenAndServe(effectiveAddr, buildHandler(db, effectiveAllowedOrigins)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
