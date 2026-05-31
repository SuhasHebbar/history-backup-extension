# history-backup-extension

A self-hosted Chrome browser history backup system. A Chrome extension uploads browsing history on a configurable schedule to a small Go HTTP server that stores it in a local SQLite database.

## Components

- [extension/](extension/) — Chrome Manifest V3 extension (plain JS, no build step)
- [server/](server/) — Go HTTP server that receives and stores history uploads

## How it works

1. The extension's background service worker reads history via the `chrome.history` API and POSTs it as JSON to a configured URL.
2. On first run it backfills all available history; after that it uploads only items newer than the last successful upload.
3. The server receives each upload and upserts rows into a SQLite database keyed by `(device_name, url)`, so repeated uploads don't create duplicates.

## Quick start

### 1. Run the server

Requires Go and a C compiler (`gcc`/`clang`) for cgo.

```bash
cd server
go build -o history-server .
./history-server --working-directory /tmp/history-data
```

The server listens on `:8080` by default. The SQLite database is created at `<working-directory>/history.db`.

### 2. Install the extension

1. Open `chrome://extensions` in Chrome.
2. Enable **Developer mode**.
3. Click **Load unpacked** and select the `extension/` directory.
4. Pin the extension and click its icon.
5. Enter the server URL (e.g. `http://localhost:8080/`), optionally set an upload period and device name, and click **Save**.

The extension starts uploading history automatically. The popup shows the last successful upload time and provides manual upload buttons.

## Configuration

### Server

Flags can be supplied via CLI or a JSON config file:

```json
{
  "addr": "0.0.0.0:8080",
  "working-directory": "/var/lib/history-server"
}
```

```bash
./history-server --config config.json
```

CLI flags take precedence over config file values. `--working-directory` is required.

### Extension

All settings are configured in the popup: upload URL, upload period (default: 60 minutes), and device name (auto-generated on first install).

## Packaging (Arch Linux)

The `server/pkg/` directory contains a PKGBUILD and systemd unit files for running the server as a system service. See [server/README.md](server/README.md) for details.

## License

MIT. See [LICENSE](LICENSE).
