package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// loadConfig tests
// ---------------------------------------------------------------------------

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("expected no error for empty path, got %v", err)
	}
	if cfg.Addr != "" || cfg.WorkDir != "" {
		t.Errorf("expected zero Config, got %+v", cfg)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	f := writeTempConfig(t, `{"addr":":9090","working-directory":"/tmp/hs"}`)

	cfg, err := loadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("addr: want :9090, got %q", cfg.Addr)
	}
	if cfg.WorkDir != "/tmp/hs" {
		t.Errorf("working-directory: want /tmp/hs, got %q", cfg.WorkDir)
	}
}

func TestLoadConfig_PartialFile(t *testing.T) {
	// Only addr set — WorkDir should be empty string.
	f := writeTempConfig(t, `{"addr":":7777"}`)

	cfg, err := loadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != ":7777" {
		t.Errorf("addr: want :7777, got %q", cfg.Addr)
	}
	if cfg.WorkDir != "" {
		t.Errorf("working-directory: want empty, got %q", cfg.WorkDir)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	f := writeTempConfig(t, `not valid json`)

	_, err := loadConfig(f)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// resolveSettings tests
// ---------------------------------------------------------------------------

func TestResolveSettings_NoArgsNoConfig(t *testing.T) {
	// Mirrors "Test 1: no args" — both CLI and config are empty.
	_, _, _, err := resolveSettings("", "", "", Config{})
	if err == nil {
		t.Fatal("expected error when working-directory is missing, got nil")
	}
}

func TestResolveSettings_ConfigFileOnly(t *testing.T) {
	// Mirrors "Test 2: --config only" — server should use config file values.
	cfg := Config{Addr: ":19191", WorkDir: "/tmp/hs-test-workdir"}

	addr, workDir, _, err := resolveSettings("", "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != ":19191" {
		t.Errorf("addr: want :19191, got %q", addr)
	}
	if workDir != "/tmp/hs-test-workdir" {
		t.Errorf("workDir: want /tmp/hs-test-workdir, got %q", workDir)
	}
}

func TestResolveSettings_CLIAddrOverridesConfig(t *testing.T) {
	// Mirrors "Test 3: --config + --addr override" — CLI addr wins.
	cfg := Config{Addr: ":19191", WorkDir: "/tmp/hs-test-workdir"}

	addr, workDir, _, err := resolveSettings(":19292", "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != ":19292" {
		t.Errorf("addr: want :19292, got %q", addr)
	}
	if workDir != "/tmp/hs-test-workdir" {
		t.Errorf("workDir: want /tmp/hs-test-workdir, got %q", workDir)
	}
}

func TestResolveSettings_CLIWorkDirOverridesConfig(t *testing.T) {
	// Mirrors "Test 4: --config + --working-directory override" — CLI workDir wins.
	cfg := Config{Addr: ":19191", WorkDir: "/tmp/hs-test-workdir"}

	addr, workDir, _, err := resolveSettings("", "/tmp/hs-alt-workdir", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != ":19191" {
		t.Errorf("addr: want :19191, got %q", addr)
	}
	if workDir != "/tmp/hs-alt-workdir" {
		t.Errorf("workDir: want /tmp/hs-alt-workdir, got %q", workDir)
	}
}

func TestResolveSettings_DefaultAddr(t *testing.T) {
	// When neither CLI nor config specifies addr, built-in default ":8080" is used.
	cfg := Config{WorkDir: "/tmp/hs"}

	addr, _, _, err := resolveSettings("", "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != ":8080" {
		t.Errorf("addr: want :8080 (built-in default), got %q", addr)
	}
}

func TestResolveSettings_CLIAddrAndWorkDirBothOverride(t *testing.T) {
	// Both CLI flags set — config file values should be completely ignored.
	cfg := Config{Addr: ":11111", WorkDir: "/tmp/from-config"}

	addr, workDir, _, err := resolveSettings(":22222", "/tmp/from-cli", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != ":22222" {
		t.Errorf("addr: want :22222, got %q", addr)
	}
	if workDir != "/tmp/from-cli" {
		t.Errorf("workDir: want /tmp/from-cli, got %q", workDir)
	}
}

func TestResolveSettings_CLIAllowedOriginsOverridesConfig(t *testing.T) {
	cfg := Config{WorkDir: "/tmp/hs", AllowedOrigins: []string{"https://config-origin.example"}}

	_, _, origins, err := resolveSettings("", "", "https://cli-a.example,https://cli-b.example", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(origins) != 2 || origins[0] != "https://cli-a.example" || origins[1] != "https://cli-b.example" {
		t.Errorf("allowedOrigins: want [https://cli-a.example https://cli-b.example], got %v", origins)
	}
}

func TestResolveSettings_ConfigAllowedOriginsUsedWhenNoCLI(t *testing.T) {
	cfg := Config{WorkDir: "/tmp/hs", AllowedOrigins: []string{"chrome-extension://abc123"}}

	_, _, origins, err := resolveSettings("", "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(origins) != 1 || origins[0] != "chrome-extension://abc123" {
		t.Errorf("allowedOrigins: want [chrome-extension://abc123], got %v", origins)
	}
}

func TestResolveSettings_AllowedOriginsEmptyWhenNeitherSet(t *testing.T) {
	cfg := Config{WorkDir: "/tmp/hs"}

	_, _, origins, err := resolveSettings("", "", "", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(origins) != 0 {
		t.Errorf("allowedOrigins: want empty, got %v", origins)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// writeTempConfig writes content to a temp file and returns its path.
// The file is automatically removed when the test ends.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return p
}
