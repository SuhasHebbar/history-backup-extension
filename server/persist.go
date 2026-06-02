package main

import (
	"database/sql"
	"fmt"
	"log"
)

// persistItems upserts all history items from the payload in a single
// BEGIN IMMEDIATE transaction.
func persistItems(db *sql.DB, payload *UploadPayload) error {
	tx, err := db.Begin() // issues BEGIN IMMEDIATE via _txlock=immediate DSN param
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(upsertSQL)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	skipped := 0
	itemCount := 0
	for _, item := range payload.Items {
		if item.URL == "" {
			skipped++
			continue
		}
		if _, err := stmt.Exec(
			payload.DeviceName,
			item.URL,
			nullableString(item.Title),
			nullableFloat64(item.LastVisitTime),
			nullableInt(item.VisitCount),
			nullableInt(item.TypedCount),
			payload.UploadedAt,
		); err != nil {
			return fmt.Errorf("upsert item url=%q: %w", item.URL, err)
		}
		itemCount++
	}

	if skipped > 0 {
		log.Printf("Skipped %d item(s) with missing URL", skipped)
	}

	if _, err := tx.Exec(insertUploadEventSQL,
		payload.UploadedAt,
		payload.DeviceName,
		itemCount,
		payload.RangeStartTime,
		payload.RangeEndTime,
	); err != nil {
		return fmt.Errorf("insert upload event: %w", err)
	}

	return tx.Commit()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

func nullableInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
