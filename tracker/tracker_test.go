package tracker

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/config"
)

func TestMain(m *testing.M) {
	// Make sure test flags are parsed before using testing.Verbose().
	flag.Parse()

	// By default use INFO
	level := slog.LevelInfo

	// testing.Verbose() is safe here — we're already inside TestMain
	if testing.Verbose() {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestStartTracker_RecordsCreateEvent(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	tracker := NewEventTracker()

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "",
			},
		},
	}

	go StartTracker(cfg, tracker)

	// needs a delay to allow for tracker to start (could use a chan to signal ready
	// from tracker but I'm not sure it's necessary to add it there yet.)
	time.Sleep(300 * time.Millisecond)

	// Trigger create event
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	time.Sleep(300 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[testFile]; !ok {
		t.Logf("Snapshot contains %v", snapshot)
		t.Errorf("Expected create event for %s, not found in snapshot", testFile)
	}
}

func TestStartTracker_RecordsWriteEvent(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "write.txt")

	tracker := NewEventTracker()

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "",
			},
		},
	}

	go StartTracker(cfg, tracker)

	// needs a delay to allow for tracker to start (could use a chan to signal ready
	// from tracker but I'm not sure it's necessary to add it there yet.)
	time.Sleep(300 * time.Millisecond)

	err := os.WriteFile(testFile, []byte("write content"), 0644)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[testFile]; !ok {
		t.Errorf("Expected write event for %s, not found in snapshot", testFile)
	}
}

func TestStartTracker_SkipsNonMatchingFilter(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "not_a_match.csv")

	tracker := NewEventTracker()

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          `^match_.*\.txt$`,
			},
		},
	}

	go StartTracker(cfg, tracker)
	// needs a delay to allow for tracker to start (could use a chan to signal ready
	// from tracker but I'm not sure it's necessary to add it there yet.)
	time.Sleep(300 * time.Millisecond)

	err := os.WriteFile(testFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatal("Failed to write file:", err)
	}

	time.Sleep(300 * time.Millisecond)

	if _, ok := tracker.GetSnapshot()[testFile]; ok {
		t.Errorf("Expected file NOT to be tracked: %s", testFile)
	}
}

func TestFilterMatcher_Validations(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "example.txt")
	_ = os.WriteFile(testFile, []byte("hello"), 0644)

	tests := []struct {
		name        string
		entry       config.ConfigEntry
		expectMatch bool
		expectErr   bool
	}{
		{
			name: "no filter applied",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          "",
			},
			expectMatch: true,
			expectErr:   false,
		},
		{
			name: "valid regex match",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `.*\.txt$`,
			},
			expectMatch: true,
			expectErr:   false,
		},
		{
			name: "valid regex no match",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `.*\.log$`,
			},
			expectMatch: false,
			expectErr:   false,
		},
		{
			name: "invalid regex",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `[abc`,
			},
			expectMatch: false,
			expectErr:   true,
		},
		{
			name: "outside directory",
			entry: config.ConfigEntry{
				SourceDirectory: "/not/here",
				Filter:          "",
			},
			expectMatch: false,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := FilterMatcher(testFile, tt.entry)

			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}
			if match != tt.expectMatch {
				t.Errorf("Expected match=%v, got %v", tt.expectMatch, match)
			}
		})
	}
}
