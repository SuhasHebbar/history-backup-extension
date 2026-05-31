# Upload Payload JSON Spec

This document describes the JSON payload sent by the extension to the configured
`uploadUrl`.

The payload is sent with:

```http
POST <uploadUrl>
Content-Type: application/json
```

The extension only sends a request when there is at least one history item to
upload. If the queried history range returns zero items, no HTTP request is made.

## Top-Level Object

```json
{
  "uploadedAt": 1716230400123,
  "deviceName": "curious-alex",
  "rangeStartTime": 1716226800000,
  "rangeEndTime": 1716230400000,
  "items": [
    {
      "id": "12345",
      "url": "https://example.com/",
      "title": "Example Domain",
      "lastVisitTime": 1716230300000,
      "visitCount": 4,
      "typedCount": 1
    }
  ]
}
```

## JSON Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "History Upload Payload",
  "type": "object",
  "additionalProperties": false,
  "required": [
    "uploadedAt",
    "deviceName",
    "rangeStartTime",
    "rangeEndTime",
    "items"
  ],
  "properties": {
    "uploadedAt": {
      "type": "number",
      "description": "Unix epoch time in milliseconds when the payload body was created."
    },
    "deviceName": {
      "type": "string",
      "description": "User-configurable device name stored by the extension. Defaults to a generated '<adjective>-<name>' value."
    },
    "rangeStartTime": {
      "type": "number",
      "description": "Inclusive lower bound passed to chrome.history.search, as Unix epoch time in milliseconds. For an all-history upload this is 0."
    },
    "rangeEndTime": {
      "type": "number",
      "description": "Upper bound passed to chrome.history.search, as Unix epoch time in milliseconds. Captured before the history query starts."
    },
    "items": {
      "type": "array",
      "minItems": 1,
      "description": "History items returned by chrome.history.search for the requested time range.",
      "items": {
        "$ref": "#/$defs/historyItem"
      }
    }
  },
  "$defs": {
    "historyItem": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id"],
      "properties": {
        "id": {
          "type": "string",
          "description": "Chrome HistoryItem id. Unique identifier for the history entry."
        },
        "url": {
          "type": "string",
          "description": "URL navigated to by the user. This field is optional in the Chrome API and may be absent if Chrome returns undefined."
        },
        "title": {
          "type": "string",
          "description": "Page title when the URL was last loaded. This field is optional in the Chrome API and may be absent if Chrome returns undefined."
        },
        "lastVisitTime": {
          "type": "number",
          "description": "Unix epoch time in milliseconds when this page was last loaded. This field is optional in the Chrome API and may be absent if Chrome returns undefined."
        },
        "visitCount": {
          "type": "number",
          "description": "Number of times the user has navigated to this page. This field is optional in the Chrome API and may be absent if Chrome returns undefined."
        },
        "typedCount": {
          "type": "number",
          "description": "Number of times the user navigated to this page by typing in the address bar. This field is optional in the Chrome API and may be absent if Chrome returns undefined."
        }
      }
    }
  }
}
```

## Field Notes For LLM Consumers

- All time values are JavaScript timestamps: Unix epoch time in milliseconds.
- `uploadedAt` can be slightly later than `rangeEndTime` because it is captured
  after the `chrome.history.search` call returns.
- `rangeStartTime` is based on the upload mode:
  - Incremental upload: previous `lastSuccessfulUploadTime`, or `0` if none is
    stored.
  - All-history upload: always `0`.
- `rangeEndTime` is saved as `lastSuccessfulUploadTime` after a successful
  upload.
- `items` contains serialized Chrome `HistoryItem` objects, not visit-level
  records. It does not include individual visit IDs, referring visit IDs,
  transition types, or per-visit timestamps.
- The service worker constructs each item with the keys shown in the schema, but
  `JSON.stringify` omits any key whose value is `undefined`. Because the Chrome
  History API marks every item field except `id` as optional, consumers should
  tolerate missing `url`, `title`, `lastVisitTime`, `visitCount`, and
  `typedCount`.
- The payload has no explicit upload mode field. Infer mode from
  `rangeStartTime` only when appropriate: `0` can mean an all-history upload or
  the first incremental upload.
