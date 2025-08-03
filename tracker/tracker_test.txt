package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"
)

func TestStartWatcher_RecordsCreateEvent(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	tracker := NewEventTracker()

	// Config entry matches the watched dir and filename
	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "", // No filter
			},
		},
	}

	StartWatcher(cfg, watcher, tracker)

	err = watcher.Add(dir)
	if err != nil {
		t.Fatalf("Failed to add directory: %v", err)
	}

	// Trigger create event
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	time.Sleep(200 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[testFile]; !ok {
		t.Errorf("Expected create event for %s, not found in snapshot", testFile)
	}
}

func TestStartWatcher_WriteEvent(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "write.txt")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Watcher creation failed: %v", err)
	}
	defer watcher.Close()

	tracker := NewEventTracker()

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "", // No filter
			},
		},
	}

	StartWatcher(cfg, watcher, tracker)

	err = watcher.Add(dir)
	if err != nil {
		t.Fatalf("Directory add failed: %v", err)
	}

	// Write to trigger event
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[testFile]; !ok {
		t.Errorf("Expected write event for %s, not found in snapshot", testFile)
	}
}

func TestStartWatcher_HandlesError(t *testing.T) {
	dir := t.TempDir()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "", // No filter
			},
		},
	}

	tracker := NewEventTracker()
	StartWatcher(cfg, watcher, tracker)

	// Inject a synthetic error (safe in controlled test)
	go func() {
		watcher.Errors <- fsnotify.ErrEventOverflow
	}()

	// Wait briefly to allow the watcher goroutine to handle the error
	time.Sleep(100 * time.Millisecond)

	t.Log("Injected error was handled without panic")
}

func TestStartWatcher_GracefulExitOnClosedChannels(t *testing.T) {
	dir := t.TempDir()

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "test",
				SourceDirectory: dir,
				Filter:          "", // No regex filter
			},
		},
	}

	tracker := NewEventTracker()
	StartWatcher(cfg, fsWatcher, tracker)

	// Close the watcher (which should close the underlying channels)
	err = fsWatcher.Close()
	if err != nil {
		t.Fatalf("Failed to close watcher: %v", err)
	}

	// Wait to ensure the goroutine exits without panic
	time.Sleep(200 * time.Millisecond)

	t.Log("StartWatcher exited gracefully after fsnotify watcher was closed")
}

func TestMatchesTransfer(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	_ = os.WriteFile(testFile, []byte("hello"), 0644)

	tests := []struct {
		name        string
		entry       config.ConfigEntry
		expectMatch bool
		expectErr   bool
	}{
		{
			name: "no filter match",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          "",
			},
			expectMatch: true,
			expectErr:   false,
		},
		{
			name: "regex match",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `.*\.txt$`,
			},
			expectMatch: true,
			expectErr:   false,
		},
		{
			name: "regex non-match",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `.*\.log$`,
			},
			expectMatch: false,
			expectErr:   false,
		},
		{
			name: "outside source dir",
			entry: config.ConfigEntry{
				SourceDirectory: "C:\\temp\\otherdir",
				Filter:          "",
			},
			expectMatch: false,
			expectErr:   false,
		},
		{
			name: "invalid regex",
			entry: config.ConfigEntry{
				SourceDirectory: tempDir,
				Filter:          `[abc`, // invalid regex
			},
			expectMatch: false,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := FilterMatcher(testFile, tt.entry)

			if tt.expectErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("did not expect error but got: %v", err)
			}
			if match != tt.expectMatch {
				t.Errorf("expected match=%v, got %v", tt.expectMatch, match)
			}
		})
	}
}

func TestStartWatcher_RecordsMatchingEvents(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "match.txt")

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				SourceDirectory: tempDir,
				Filter:          `.*\.txt$`,
			},
		},
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(tempDir)
	if err != nil {
		t.Fatalf("failed to add temp dir to watcher: %v", err)
	}

	tracker := NewEventTracker()
	StartWatcher(cfg, watcher, tracker)

	// Write to file to trigger event
	err = os.WriteFile(testFile, []byte("trigger event"), 0644)
	if err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}

	// Wait for the event to propagate
	time.Sleep(500 * time.Millisecond)

	snapshot := tracker.GetSnapshot()

	found := false
	for file := range snapshot {
		if strings.HasSuffix(file, "match.txt") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected event to be recorded for match.txt, but it wasn't")
	}
}

func TestStartWatcher_CreateEvent_WithMatchingFilter(t *testing.T) {
	dir := t.TempDir()
	matchingFile := filepath.Join(dir, "match_me.txt")

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "with-filter",
				SourceDirectory: dir,
				Filter:          `^match_.*\.txt$`, // Only match files starting with 'match_'
			},
		},
	}

	tracker := NewEventTracker()
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal("Failed to create watcher:", err)
	}
	defer fsWatcher.Close()

	err = fsWatcher.Add(dir)
	if err != nil {
		t.Fatal("Failed to add dir to watcher:", err)
	}

	StartWatcher(cfg, fsWatcher, tracker)

	// Trigger Create event
	f, err := os.Create(matchingFile)
	if err != nil {
		t.Fatal("Failed to create test file:", err)
	}
	f.Close()

	time.Sleep(200 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, found := snapshot[matchingFile]; !found {
		t.Errorf("Expected matching file to be tracked: %s", matchingFile)
	}
}

func TestStartWatcher_WriteEvent_WithNonMatchingFilter(t *testing.T) {
	dir := t.TempDir()
	nonMatchingFile := filepath.Join(dir, "not_this_one.txt")

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "with-filter",
				SourceDirectory: dir,
				Filter:          `^match_.*\.txt$`, // Doesn't match 'not_this_one.txt'
			},
		},
	}

	tracker := NewEventTracker()
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal("Failed to create watcher:", err)
	}
	defer fsWatcher.Close()

	err = fsWatcher.Add(dir)
	if err != nil {
		t.Fatal("Failed to add dir to watcher:", err)
	}

	StartWatcher(cfg, fsWatcher, tracker)

	// Trigger Write event
	err = os.WriteFile(nonMatchingFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatal("Failed to write test file:", err)
	}

	time.Sleep(200 * time.Millisecond)

	snapshot := tracker.GetSnapshot()
	if _, found := snapshot[nonMatchingFile]; found {
		t.Errorf("Expected non-matching file NOT to be tracked: %s", nonMatchingFile)
	}
}
