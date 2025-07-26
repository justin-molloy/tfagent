package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"log/slog"
)

func TestParseFlags(t *testing.T) {
	flags := ParseFlags()

	// Optionally check expected fields (if you use defaults)
	if flags.ConfigFile == "" {
		t.Error("Expected ConfigPath to be set (or defaulted), got empty string")
	}

	// You can also log the result for visibility
	t.Logf("Flags parsed: %+v", flags)
}

func TestLoadConfig_Success(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_config.yaml")

	content := `
logdest: "/tmp/logs"
loglevel: "info"
delay: 2
transfers:
  - name: "test-transfer"
    source_directory: "/tmp/source"
    destination: "/tmp/dest"
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	// Call the function
	cfg, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig returned an error: %v", err)
	}

	// Basic assertions
	if cfg.LogFile != "/tmp/logs" {
		t.Errorf("Expected LogFile to be /tmp/logs, got %s", cfg.LogFile)
	}

	if len(cfg.Transfers) != 1 {
		t.Fatalf("Expected 1 transfer entry, got %d", len(cfg.Transfers))
	}

	transfer := cfg.Transfers[0]
	if transfer.Name != "test-transfer" {
		t.Errorf("Expected Name to be 'test-transfer', got %s", transfer.Name)
	}
}

func TestLoadConfigWarnStreamingNotSet(t *testing.T) {
	// Set up a buffer and a custom slog handler to capture log warnings.
	var logBuf bytes.Buffer
	handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		// Set the level to Warning to capture warning messages.
		Level: slog.LevelWarn,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Create a temporary YAML config file with a transfer that leaves 'streaming' unset.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_config.yaml")
	yamlContent := `
logdest: "/tmp/logs"
loglevel: "info"
transfers:
  - name: "test-transfer"
    source_directory: "/tmp/source"
    destination: "/tmp/dest"
    delay: 5
`
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temporary config file: %v", err)
	}

	// Call LoadConfig
	cfg, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Retrieve the logged output.
	logOutput := logBuf.String()
	t.Logf("Captured log output: \n%s", logOutput)

	// Check that the log output contains the expected warning.
	expectedSubstring := fmt.Sprintf("Streaming value for %s", cfg.Transfers[0].Name)
	if !strings.Contains(logOutput, expectedSubstring) {
		t.Errorf("Expected log warning to contain %q, got: %s", expectedSubstring, logOutput)
	}

	// Optionally, verify that defaults were set.
	if cfg.Transfers[0].Streaming == nil {
		t.Errorf("Expected Streaming default to be set, but found nil")
	} else if !*cfg.Transfers[0].Streaming { // expecting default false
		t.Logf("Streaming default is set to false as expected")
	} else {
		t.Errorf("Unexpected Streaming default value: %v", *cfg.Transfers[0].Streaming)
	}
}
