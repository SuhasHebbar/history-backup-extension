package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config holds values that can be supplied via a JSON config file.
type Config struct {
	Addr           string   `json:"addr"`
	WorkDir        string   `json:"working-directory"`
	AllowedOrigins []string `json:"allowed-origins"`
}

// loadConfig reads and parses a JSON config file at path.
// If path is empty it returns an empty Config without error.
func loadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

// resolveSettings merges CLI flag values with config-file values.
// CLI values take precedence; the built-in default for addr is ":8080".
// It returns an error when workDir cannot be determined.
func resolveSettings(cliAddr, cliWorkDir, cliAllowedOrigins string, cfg Config) (addr, workDir string, allowedOrigins []string, err error) {
	addr = cliAddr
	if addr == "" {
		addr = cfg.Addr
	}
	if addr == "" {
		addr = ":8080"
	}

	workDir = cliWorkDir
	if workDir == "" {
		workDir = cfg.WorkDir
	}
	if workDir == "" {
		return "", "", nil, fmt.Errorf("--working-directory is required (via flag or config file)")
	}

	if cliAllowedOrigins != "" {
		allowedOrigins = strings.Split(cliAllowedOrigins, ",")
	} else {
		allowedOrigins = cfg.AllowedOrigins
	}

	return addr, workDir, allowedOrigins, nil
}

func finalConfigJSON(addr, workDir string, allowedOrigins []string) (string, error) {
	if allowedOrigins == nil {
		allowedOrigins = []string{}
	}

	b, err := json.Marshal(Config{
		Addr:           addr,
		WorkDir:        workDir,
		AllowedOrigins: allowedOrigins,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
