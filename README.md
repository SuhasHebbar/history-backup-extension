# chrome.history

This sample uses the [`chrome.history`](https://developer.chrome.com/docs/extensions/reference/history/) API to display in a popup the user's most visited pages.

## Overview

This extension calls `chrome.history.search()` to scrape the browser's history and count occurrences of each visited URL.

The background service worker also uploads history incrementally once per minute to `http://placeholder:9001/`.
It stores upload configuration and state in `chrome.storage.local`, including the timestamp of the last
successful upload. On first run, it backfills all available history. After that, it only uploads history
items newer than the last successful upload.

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

## Running this extension

1. Clone this repository.
2. Load this directory in Chrome as an [unpacked extension](https://developer.chrome.com/docs/extensions/mv3/getstarted/development-basics/#load-unpacked).
3. Pin the extension to the browser's taskbar.
4. Click on the extension's action button to view your most visited pages.
5. Run a server at `http://placeholder:9001/` and verify that the background service worker sends history
   uploads about once per minute.
