# Agent Instructions

## Project Overview
- This repository is a small Chrome Manifest V3 extension.
- The extension uses the `chrome.history` API to upload browser history on a configurable schedule.
- The upload target is configurable, and the default upload period is 60 minutes.
- Core files:
  - `manifest.json` defines extension metadata, permissions, background service worker, and popup.
  - `popup.html` contains the upload settings UI and inline CSS.
  - `popup.js` manages upload settings, optional host permission requests, and manual uploads.
  - `service-worker.js` schedules and performs incremental or all-history uploads.
  - `shared-config.js` contains default upload settings, storage key constants, and device name helpers.
  - `upload-payload-spec.md` documents the JSON upload payload.
  - `chrome-history-api-docs.md` is local reference material for the Chrome History API.
  - `chrome-storage-api-docs.md` is local reference material for the Chrome Storage API.

## Development Notes
- There is no package manager, build step, or test runner configured.
- Keep changes lightweight and dependency-free unless the user explicitly asks for tooling.
- Prefer plain JavaScript, HTML, and CSS that can run directly as an unpacked Chrome extension.
- Preserve Manifest V3 compatibility.
- Use Chrome extension APIs only in extension contexts where they are available.
- Be careful with the `history` permission. Avoid adding broader permissions unless required for the requested behavior.
- Be careful with host permissions. The user must set an upload URL to use the extension, and saving that URL requests the required `http://*/*` or `https://*/*` host access.
- Keep upload configuration in `chrome.storage.local` under the shared `historyUpload` key.
- When changing upload cadence, update `shared-config.js` and any user-facing docs or placeholders that mention the default period.

## Style Guidelines
- Follow the existing simple, browser-native style.
- Use clear DOM APIs rather than framework code.
- Keep comments concise and useful.
- Default to ASCII when editing files.
- Avoid unrelated refactors while changing behavior.

## Verification
- For static checks, inspect the changed files directly since no automated checks are configured.
- To manually test:
  1. Open `chrome://extensions`.
  2. Enable Developer mode.
  3. Load this repository as an unpacked extension.
  4. Pin the extension and open the popup.
  5. Verify that upload URL, upload period, and device name settings load and save.
  6. Verify that the default upload period is 60 minutes when no custom period is saved.
  7. Run a server at the configured upload URL and verify scheduled or manual upload requests.
  8. Verify that manual uploads can send all history or only history since the last successful upload.

## Documentation
- Update `README.md` when user-facing behavior or setup steps change.
- Update `upload-payload-spec.md` when the upload request body changes.
- Use `chrome-history-api-docs.md` and `chrome-storage-api-docs.md` for local API context before looking elsewhere.
