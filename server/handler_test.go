package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ensure sql import is used (newTestDB returns *sql.DB)
var _ *sql.DB

func TestStatusHandler_OK(t *testing.T) {
	h := statusHandler()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}
	if got, want := rr.Body.String(), "ok"; got != want {
		t.Errorf("body: got %q, want %q", got, want)
	}
	if got, want := rr.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
		t.Errorf("content type: got %q, want %q", got, want)
	}
}

func TestStatusHandler_MethodNotAllowed(t *testing.T) {
	h := statusHandler()
	req := httptest.NewRequest(http.MethodPost, "/status", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not valid json"))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_MissingDeviceName(t *testing.T) {
	payload := UploadPayload{
		UploadedAt: 1234567890,
		DeviceName: "",
		Items:      []HistoryItem{{ID: "1", URL: "https://example.com"}},
	}
	body, _ := json.Marshal(payload)

	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_MissingUploadedAt(t *testing.T) {
	payload := UploadPayload{
		UploadedAt: 0,
		DeviceName: "test-device",
		Items:      []HistoryItem{{ID: "1", URL: "https://example.com"}},
	}
	body, _ := json.Marshal(payload)

	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_EmptyItems(t *testing.T) {
	payload := UploadPayload{
		UploadedAt: 1234567890,
		DeviceName: "test-device",
		Items:      []HistoryItem{},
	}
	body, _ := json.Marshal(payload)

	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_ValidPayload(t *testing.T) {
	payload := UploadPayload{
		UploadedAt: 1234567890,
		DeviceName: "test-device",
		Items:      []HistoryItem{{ID: "1", URL: "https://example.com"}},
	}
	body, _ := json.Marshal(payload)

	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusNoContent; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_SkipsEmptyURLItems(t *testing.T) {
	// Items array is non-empty, but all items have empty URLs — persist succeeds, items get skipped.
	payload := UploadPayload{
		UploadedAt: 1234567890,
		DeviceName: "test-device",
		Items:      []HistoryItem{{ID: "1", URL: ""}, {ID: "2", URL: ""}},
	}
	body, _ := json.Marshal(payload)

	h := handler(newTestDB(t))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusNoContent; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHandler_ItemsPersistedToDB(t *testing.T) {
	db := newTestDB(t)

	payload := UploadPayload{
		UploadedAt: 1234567890,
		DeviceName: "test-device",
		Items: []HistoryItem{
			{ID: "1", URL: "https://example.com"},
			{ID: "2", URL: "https://golang.org"},
		},
	}
	body, _ := json.Marshal(payload)

	h := handler(db)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h(rr, req)

	if got, want := rr.Code, http.StatusNoContent; got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM history_items").Scan(&count); err != nil {
		t.Fatalf("query row count: %v", err)
	}
	if count != 2 {
		t.Errorf("row count: got %d, want 2", count)
	}
}
