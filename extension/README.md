# chrome.history

This extension uses the [`chrome.history`](https://developer.chrome.com/docs/extensions/reference/history/) API to upload browser history on a configurable schedule.

## Overview

The background service worker uploads history incrementally to the configured upload URL.
The popup configures the required upload URL, upload period, and device name, and displays the date and
time of the last successful upload. If the upload period is left blank, the extension uses the default
value. The device name is initialized automatically on first install with a generated
`<random adjective>-<random name>` value and can be edited in the popup. If the device name field is
cleared and saved, the extension generates a new one. It stores upload configuration and state in
`chrome.storage.local`, including the timestamp of the last successful upload. On first run, it backfills
all available history. After that, it only uploads history items newer than the last successful upload.

The popup also includes manual upload buttons. "Upload all history" uploads all history items Chrome
returns regardless of the saved timestamp. "Upload since last successful upload" uploads only items newer
than the saved timestamp. A successful manual upload updates the last successful upload timestamp.

Upload requests are sent as `POST` requests to the configured upload URL with a JSON body containing:

- `uploadedAt`
- `deviceName`
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
- `http://*/*` and `https://*/*` host access, requested when saving the required upload URL.

## Running this extension

1. Clone this repository.
2. Load this directory in Chrome as an [unpacked extension](https://developer.chrome.com/docs/extensions/mv3/getstarted/development-basics/#load-unpacked).
3. Pin the extension to the browser's taskbar.
4. Click on the extension's action button to configure the upload settings.
5. Enter an upload URL in the popup, optionally enter an upload period or device name, and click Save.
   Chrome asks for host permission when saving the upload URL.
6. Run a server at the configured upload URL and verify that the background service worker sends history
   uploads on the configured schedule. The popup shows the last successful upload date and time, or
   `Never` before the first successful upload.
7. Use the popup's manual upload buttons to upload all available history or only history since the last
   successful upload.
