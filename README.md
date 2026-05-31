# history-server

`history-server` is a small Go HTTP service that receives browser history uploads
from a Chrome extension and stores them in a local SQLite database.

The server exposes a single `POST /` endpoint. Uploads are stored in
`history.db` under the configured working directory, and repeated uploads upsert
rows by `(device_name, url)` instead of creating duplicates.

## Requirements

- Go 1.26.3 or newer
- A C compiler such as `gcc` or `clang`

This project uses `github.com/mattn/go-sqlite3`, which depends on cgo.

## Build

```bash
go build ./...
```

This creates a `history-server` binary in the project directory when building
the main package directly:

```bash
go build -o history-server .
```

## Run

`--working-directory` is required unless it is supplied by a config file. The
SQLite database is created at `<working-directory>/history.db`.

```bash
./history-server --working-directory /tmp/hs-data
```

By default, the server listens on `:8080`. Override it with `--addr`:

```bash
./history-server --addr :9090 --working-directory /tmp/hs-data
```

## Configuration

Configuration can be supplied with a JSON file:

```json
{
  "addr": "0.0.0.0:9001",
  "working-directory": "/var/lib/history-server"
}
```

Run with:

```bash
./history-server --config config.json
```

Configuration precedence is:

1. CLI flags
2. Config file values
3. Built-in defaults

The only built-in default is `addr: ":8080"`. There is no default working
directory.

## API

### `POST /`

Stores one upload payload. A successful request returns `204 No Content`.

Required top-level fields:

- `deviceName`
- `uploadedAt`
- `items`, which must be non-empty

`rangeStartTime` and `rangeEndTime` may be included by the client, but they are
not currently persisted.

Example:

```bash
curl -i http://localhost:8080/ \
  -H 'Content-Type: application/json' \
  -d '{
    "deviceName": "laptop",
    "uploadedAt": 1716300000000,
    "rangeStartTime": 1716213600000,
    "rangeEndTime": 1716300000000,
    "items": [
      {
        "id": "42",
        "url": "https://example.com",
        "title": "Example",
        "lastVisitTime": 1716299000000,
        "visitCount": 7,
        "typedCount": 2
      }
    ]
  }'
```

Items without a `url` are skipped. Optional item fields are stored as `NULL`
when absent or empty.

## Database

The server creates a SQLite table named `history_items`:

```sql
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
```

On startup, the server creates the table if needed. If the table already exists,
it validates that the expected columns and declared SQLite types are present.

SQLite is opened with WAL mode and other pragmas intended for a local
single-writer service.

## Development

Run all tests:

```bash
go test ./...
```

Run a specific test:

```bash
go test -run TestPersistItems_Upsert ./...
```

## Packaging

The `pkg/` directory contains Arch Linux packaging files:

- `pkg/PKGBUILD`
- `pkg/history-server.service`
- `pkg/history-server.sysusers`
- `pkg/history-server.tmpfiles`

The packaged systemd service runs:

```bash
/usr/bin/history-server --config /etc/history-server/config.json
```

with the default config pointing at `/var/lib/history-server`.

## License

MIT. See [LICENSE](LICENSE).
