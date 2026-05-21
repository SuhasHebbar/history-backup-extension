# chrome.history

This extension uses the [`chrome.history`](https://developer.chrome.com/docs/extensions/reference/history/) API to upload browser history on a configurable schedule.

## Overview

The background service worker uploads history incrementally once per minute to `http://placeholder:9001/`.
The popup can configure a custom upload URL and upload period, and displays the date and time of the
last successful upload. If either settings field is left blank, the extension uses the default value.
It stores upload configuration and state in `chrome.storage.local`, including the timestamp of the last
successful upload. On first run, it backfills all available history. After that, it only uploads history
items newer than the last successful upload.

The popup also includes manual upload buttons. "Upload all history" uploads all history items Chrome
returns regardless of the saved timestamp. "Upload since last successful upload" uploads only items newer
than the saved timestamp. A successful manual upload updates the last successful upload timestamp.

Upload requests are sent as `POST http://placeholder:9001/` with a JSON body containing:

- `uploadedAt`
- `rangeStartTime`
- `rangeEndTime`
- `items`

Each item contains the `HistoryItem` fields returned by Chrome: `id`, `url`, `title`, `lastVisitTime`,
`visitCount`, and `typedCount`.

## Permissions

This extension uses:

- `history` to read browser history.
- `storage` to persist upload configuration and state.
- `alarms` to run the upload every minute.
- `http://placeholder:9001/*` host access to upload history.
- Optional `http://*/*` and `https://*/*` host access, requested when saving a custom upload URL.

## Running this extension

1. Clone this repository.
2. Load this directory in Chrome as an [unpacked extension](https://developer.chrome.com/docs/extensions/mv3/getstarted/development-basics/#load-unpacked).
3. Pin the extension to the browser's taskbar.
4. Click on the extension's action button to configure the upload settings.
5. Optionally enter a custom upload URL or upload period in the popup and click Save. Chrome asks for
   host permission when saving a custom upload URL.
6. Run a server at the configured upload URL, or at `http://placeholder:9001/` when using the default, and
   verify that the background service worker sends history uploads on the configured schedule. The popup
   shows the last successful upload date and time, or `Never` before the first successful upload.
7. Use the popup's manual upload buttons to upload all available history or only history since the last
   successful upload.
