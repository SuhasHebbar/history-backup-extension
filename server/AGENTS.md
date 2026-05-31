# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test -run TestPersistItems_Upsert ./...

# Run the server
./history-server --working-directory /tmp/hs-data
./history-server --config config.json
./history-server --addr :9090 --working-directory /tmp/hs-data
```

**Note:** `go-sqlite3` uses cgo — a C compiler (`gcc`/`clang`) must be available.

## Architecture

This is a single-file Go HTTP server (`main.go`) in `package main`. It receives browser history uploads from a Chrome extension and persists them to a local SQLite database.

**Request flow:** `POST /` → `handler()` validates the JSON payload → `persistItems()` upserts rows in a `BEGIN IMMEDIATE` transaction.

**Key types:**
- `UploadPayload` — top-level JSON body: `deviceName`, `uploadedAt`, `rangeStartTime`, `rangeEndTime`, `items[]`
- `HistoryItem` — mirrors Chrome's `HistoryItem` API; all fields except `id` are optional (pointer types for nullable columns)
- `Config` — JSON config file structure (`addr`, `working-directory`)

**Database:** SQLite at `<working-directory>/history.db`. Primary key is `(device_name, url)` — repeated uploads upsert rather than duplicate. `openDB()` applies WAL-mode pragmas and calls `ensureSchema()`, which creates the table on first run or validates column names/types against `expectedColumns` on subsequent runs.

**Configuration precedence:** CLI flags > config file > built-in defaults. `--working-directory` is required (no built-in default). Default listen address is `:8080`.

**Test helpers:** `newTestDB(t)` in `testhelpers_test.go` opens an in-memory SQLite DB with the full schema applied — use this in any new tests rather than setting up DB state manually.
