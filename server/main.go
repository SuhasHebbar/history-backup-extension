package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file (optional)")
	addr := flag.String("addr", "", "listen address (host:port) — overrides config file")
	workDir := flag.String("working-directory", "", "path to working directory for the SQLite database — overrides config file")
	allowedOrigins := flag.String("allowed-origins", "", "comma-separated list of allowed CORS origins — overrides config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	effectiveAddr, effectiveWorkDir, effectiveAllowedOrigins, err := resolveSettings(*addr, *workDir, *allowedOrigins, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(1)
	}

	finalConfig, err := finalConfigJSON(effectiveAddr, effectiveWorkDir, effectiveAllowedOrigins)
	if err != nil {
		log.Fatalf("marshal final config: %v", err)
	}
	log.Printf("Final config: %s", finalConfig)

	if err := os.MkdirAll(effectiveWorkDir, 0755); err != nil {
		log.Fatalf("create working directory %q: %v", effectiveWorkDir, err)
	}

	dbPath := filepath.Join(effectiveWorkDir, "history.db")
	log.Printf("Opening database at %s", dbPath)

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	log.Printf("Listening on %s", effectiveAddr)
	if err := http.ListenAndServe(effectiveAddr, buildHandler(db, effectiveAllowedOrigins)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
