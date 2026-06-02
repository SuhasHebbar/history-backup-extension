package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"
)

// handler returns an http.HandlerFunc that parses upload payloads and persists
// history items to the SQLite database.
func handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR read body: %v", err)
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}

		var payload UploadPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("ERROR parse JSON: %v", err)
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if payload.DeviceName == "" {
			http.Error(w, "missing required field: deviceName", http.StatusBadRequest)
			return
		}
		if payload.UploadedAt == 0 {
			http.Error(w, "missing required field: uploadedAt", http.StatusBadRequest)
			return
		}
		if len(payload.Items) == 0 {
			http.Error(w, "items array must not be empty", http.StatusBadRequest)
			return
		}

		if err := persistItems(db, &payload); err != nil {
			log.Printf("ERROR persist items: %v", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		elapsed := time.Since(start)
		log.Printf("Stored %d item(s) from device=%q in %s", len(payload.Items), payload.DeviceName, elapsed)

		w.WriteHeader(http.StatusNoContent)
	}
}

func statusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func buildHandler(db *sql.DB, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler())
	mux.HandleFunc("/", handler(db))
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
	})
	return c.Handler(mux)
}
