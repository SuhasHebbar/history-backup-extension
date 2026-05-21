# Agent Instructions

## Project Overview
- This repository is a small Chrome Manifest V3 extension.
- The extension uses the `chrome.history` API to show recently typed URLs in a browser action popup.
- Core files:
  - `manifest.json` defines extension metadata, permissions, background service worker, and popup.
  - `popup.html` contains the popup shell and inline CSS.
  - `popup.js` queries browser history and renders popup links.
  - `service-worker.js` contains the MV3 background service worker.
  - `chrome-history-api-docs.md` is local reference material for the Chrome History API.

## Development Notes
- There is no package manager, build step, or test runner configured.
- Keep changes lightweight and dependency-free unless the user explicitly asks for tooling.
- Prefer plain JavaScript, HTML, and CSS that can run directly as an unpacked Chrome extension.
- Preserve Manifest V3 compatibility.
- Use Chrome extension APIs only in extension contexts where they are available.
- Be careful with the `history` permission. Avoid adding broader permissions unless required for the requested behavior.

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
  5. Verify that recently typed URLs appear and links open in a new tab.

## Documentation
- Update `README.md` when user-facing behavior or setup steps change.
- Use `chrome-history-api-docs.md` for local API context before looking elsewhere.
