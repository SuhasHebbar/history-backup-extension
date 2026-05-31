package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// Local pointer helpers
// ---------------------------------------------------------------------------

func ptrFloat64(f float64) *float64 { return &f }
func ptrInt(i int) *int             { return &i }

// ---------------------------------------------------------------------------
// nullableString tests
// ---------------------------------------------------------------------------

func TestNullableString_Empty(t *testing.T) {
	if got := nullableString(""); got != nil {
		t.Errorf("nullableString(%q): want nil, got %v", "", got)
	}
}

func TestNullableString_NonEmpty(t *testing.T) {
	want := "hello"
	got := nullableString(want)
	if got != want {
		t.Errorf("nullableString(%q): want %q, got %v", want, want, got)
	}
}

// ---------------------------------------------------------------------------
// nullableFloat64 tests
// ---------------------------------------------------------------------------

func TestNullableFloat64_Nil(t *testing.T) {
	if got := nullableFloat64(nil); got != nil {
		t.Errorf("nullableFloat64(nil): want nil, got %v", got)
	}
}

func TestNullableFloat64_NonNil(t *testing.T) {
	f := 1.5
	got := nullableFloat64(&f)
	if got != f {
		t.Errorf("nullableFloat64(&1.5): want 1.5, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// nullableInt tests
// ---------------------------------------------------------------------------

func TestNullableInt_Nil(t *testing.T) {
	if got := nullableInt(nil); got != nil {
		t.Errorf("nullableInt(nil): want nil, got %v", got)
	}
}

func TestNullableInt_NonNil(t *testing.T) {
	i := 42
	got := nullableInt(&i)
	if got != i {
		t.Errorf("nullableInt(&42): want 42, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// persistItems tests
// ---------------------------------------------------------------------------

func TestPersistItems_Basic(t *testing.T) {
	db := newTestDB(t)

	visitCount := 3
	typedCount := 1
	payload := &UploadPayload{
		UploadedAt: 1000,
		DeviceName: "my-device",
		Items: []HistoryItem{
			{
				ID:         "1",
				URL:        "https://example.com",
				Title:      "Example",
				VisitCount: &visitCount,
				TypedCount: &typedCount,
			},
		},
	}

	if err := persistItems(db, payload); err != nil {
		t.Fatalf("persistItems: %v", err)
	}

	var deviceName, url, title string
	var visitCountDB, typedCountDB, uploadedAt int64
	err := db.QueryRow(
		`SELECT device_name, url, title, visit_count, typed_count, uploaded_at FROM history_items`,
	).Scan(&deviceName, &url, &title, &visitCountDB, &typedCountDB, &uploadedAt)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}

	if deviceName != "my-device" {
		t.Errorf("device_name: want %q, got %q", "my-device", deviceName)
	}
	if url != "https://example.com" {
		t.Errorf("url: want %q, got %q", "https://example.com", url)
	}
	if title != "Example" {
		t.Errorf("title: want %q, got %q", "Example", title)
	}
	if visitCountDB != int64(visitCount) {
		t.Errorf("visit_count: want %d, got %d", visitCount, visitCountDB)
	}
	if typedCountDB != int64(typedCount) {
		t.Errorf("typed_count: want %d, got %d", typedCount, typedCountDB)
	}
	if uploadedAt != 1000 {
		t.Errorf("uploaded_at: want 1000, got %d", uploadedAt)
	}
}

func TestPersistItems_Upsert(t *testing.T) {
	db := newTestDB(t)

	base := &UploadPayload{
		UploadedAt: 1000,
		DeviceName: "dev",
		Items: []HistoryItem{
			{ID: "1", URL: "https://example.com", Title: "First"},
		},
	}
	if err := persistItems(db, base); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	second := &UploadPayload{
		UploadedAt: 2000,
		DeviceName: "dev",
		Items: []HistoryItem{
			{ID: "1", URL: "https://example.com", Title: "Second"},
		},
	}
	if err := persistItems(db, second); err != nil {
		t.Fatalf("second insert (upsert): %v", err)
	}

	var title string
	if err := db.QueryRow(`SELECT title FROM history_items`).Scan(&title); err != nil {
		t.Fatalf("query title: %v", err)
	}
	if title != "Second" {
		t.Errorf("title after upsert: want %q, got %q", "Second", title)
	}
}

func TestPersistItems_SkipsEmptyURL(t *testing.T) {
	db := newTestDB(t)

	payload := &UploadPayload{
		UploadedAt: 1000,
		DeviceName: "dev",
		Items: []HistoryItem{
			{ID: "1", URL: ""},                    // should be skipped
			{ID: "2", URL: "https://example.com"}, // should be inserted
		},
	}
	if err := persistItems(db, payload); err != nil {
		t.Fatalf("persistItems: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM history_items`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("row count: want 1, got %d", count)
	}
}

func TestPersistItems_NullableFields(t *testing.T) {
	db := newTestDB(t)

	payload := &UploadPayload{
		UploadedAt: 1000,
		DeviceName: "dev",
		Items: []HistoryItem{
			{ID: "1", URL: "https://example.com", Title: "", VisitCount: nil, TypedCount: nil},
		},
	}
	if err := persistItems(db, payload); err != nil {
		t.Fatalf("persistItems: %v", err)
	}

	var title sql.NullString
	var visitCount sql.NullInt64
	var typedCount sql.NullInt64
	err := db.QueryRow(`SELECT title, visit_count, typed_count FROM history_items`).
		Scan(&title, &visitCount, &typedCount)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if title.Valid {
		t.Errorf("title: want NULL, got %q", title.String)
	}
	if visitCount.Valid {
		t.Errorf("visit_count: want NULL, got %d", visitCount.Int64)
	}
	if typedCount.Valid {
		t.Errorf("typed_count: want NULL, got %d", typedCount.Int64)
	}
}

func TestPersistItems_AllFieldsPopulated(t *testing.T) {
	db := newTestDB(t)

	lvt := 1716300000.0
	vc := 7
	tc := 2
	payload := &UploadPayload{
		UploadedAt: 9999,
		DeviceName: "full-device",
		Items: []HistoryItem{
			{
				ID:            "42",
				URL:           "https://full.example.com",
				Title:         "Full Item",
				LastVisitTime: &lvt,
				VisitCount:    &vc,
				TypedCount:    &tc,
			},
		},
	}
	if err := persistItems(db, payload); err != nil {
		t.Fatalf("persistItems: %v", err)
	}

	var title string
	var lastVisitTime sql.NullInt64
	var visitCount, typedCount int
	var uploadedAt int64
	err := db.QueryRow(
		`SELECT title, last_visit_time, visit_count, typed_count, uploaded_at FROM history_items`,
	).Scan(&title, &lastVisitTime, &visitCount, &typedCount, &uploadedAt)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if title != "Full Item" {
		t.Errorf("title: want %q, got %q", "Full Item", title)
	}
	if !lastVisitTime.Valid {
		t.Error("last_visit_time: want non-NULL")
	} else if lastVisitTime.Int64 != int64(lvt) {
		t.Errorf("last_visit_time: want %d, got %d", int64(lvt), lastVisitTime.Int64)
	}
	if visitCount != vc {
		t.Errorf("visit_count: want %d, got %d", vc, visitCount)
	}
	if typedCount != tc {
		t.Errorf("typed_count: want %d, got %d", tc, typedCount)
	}
	if uploadedAt != 9999 {
		t.Errorf("uploaded_at: want 9999, got %d", uploadedAt)
	}
}

func TestPersistItems_MultipleItems(t *testing.T) {
	db := newTestDB(t)

	items := make([]HistoryItem, 5)
	for i := range items {
		items[i] = HistoryItem{
			ID:  string(rune('a' + i)),
			URL: "https://example.com/" + string(rune('a'+i)),
		}
	}

	payload := &UploadPayload{
		UploadedAt: 5000,
		DeviceName: "multi-dev",
		Items:      items,
	}
	if err := persistItems(db, payload); err != nil {
		t.Fatalf("persistItems: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM history_items`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 5 {
		t.Errorf("row count: want 5, got %d", count)
	}
}
