# Privacy Policy for History Backupper

Effective date: May 31, 2026

History Backupper is a Chrome extension that lets users back up their browser history to an upload endpoint they configure.

## Data Collection by the Extension Author

The extension author does not collect, receive, store, sell, share, or otherwise process users' personal data.

The extension does not upload browsing history, configuration values, device names, or any other personal data to servers controlled by the extension author.

## Data Processed by the Extension

To provide its backup functionality, the extension uses Chrome extension APIs on the user's device:

- The `history` permission allows the extension to read browser history.
- The `storage` permission allows the extension to save local configuration and upload state.
- The `alarms` permission allows the extension to run uploads on a configurable schedule.
- Optional host permissions are requested only for the upload URL chosen by the user.

Browsing history may include sensitive personal information, including visited URLs, page titles, visit times, visit counts, and typed counts.

## User-Controlled Uploads

The user chooses whether to configure an upload URL. If the user configures an upload URL, the extension may upload browser history to that user-selected endpoint on the configured schedule or when the user manually starts an upload.

The upload endpoint may be a self-hosted service operated by the user or a third-party service chosen by the user. The privacy, security, retention, and processing practices of that endpoint are controlled by the user or by the third party operating it, not by the extension author.

Users should only configure upload endpoints they trust.

## Local Storage

The extension stores configuration and upload state locally in the user's browser using `chrome.storage.local`. This may include the configured upload URL, upload period, device name, and the timestamp of the last successful upload.

## Third Parties

The extension does not include analytics, advertising, tracking SDKs, or third-party data collection code.

If a user chooses to upload browsing history to a third-party endpoint, that transfer is initiated by the user's own configuration.

## Data Retention and Deletion

Because the extension author does not collect or store user data, the extension author has no user data to retain or delete.

Users can delete locally stored extension data by removing the extension from Chrome or by clearing the extension's stored data through Chrome's extension settings. Users who upload data to a self-hosted or third-party endpoint are responsible for managing deletion from that endpoint.

## Security

Users are responsible for selecting and securing their upload endpoint. When possible, users should use HTTPS endpoints to protect uploaded browsing history in transit.

## Changes to This Policy

This policy may be updated if the extension's functionality or data practices change. Any future changes should continue to describe what data is processed, who receives it, and how users control it.

## Contact

For privacy questions about History Backupper, open an issue on the extension github repository.
