package main

// HistoryItem mirrors the Chrome HistoryItem fields from the extension payload.
// All fields except ID are optional per the Chrome API.
type HistoryItem struct {
	ID            string   `json:"id"`
	URL           string   `json:"url,omitempty"`
	Title         string   `json:"title,omitempty"`
	LastVisitTime *float64 `json:"lastVisitTime,omitempty"`
	VisitCount    *int     `json:"visitCount,omitempty"`
	TypedCount    *int     `json:"typedCount,omitempty"`
}

// UploadPayload is the top-level JSON body sent by the extension.
type UploadPayload struct {
	UploadedAt     int64         `json:"uploadedAt"`
	DeviceName     string        `json:"deviceName"`
	RangeStartTime int64         `json:"rangeStartTime"`
	RangeEndTime   int64         `json:"rangeEndTime"`
	Items          []HistoryItem `json:"items"`
}
