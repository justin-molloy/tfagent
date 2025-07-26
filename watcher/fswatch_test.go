package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestStartWatcher_RecordsCreateEvent(t *testing.T) {
	// Setup: temp dir & file
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(dir)
	if err != nil {
		t.Fatalf("Failed to add directory to watcher: %v", err)
	}

	tracker := NewEventTracker()
	done := make(chan struct{})
	StartWatcher(watcher, tracker)

	// Trigger a Create event
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	// Wait a bit for watcher to process
	time.Sleep(200 * time.Millisecond)
	close(done)

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[testFile]; !ok {
		t.Errorf("Expected event for %s, but not found in snapshot", testFile)
	}
}

func TestStartWatcher_WriteEvent(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "write_test.txt")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(dir)
	if err != nil {
		t.Fatalf("Failed to watch dir: %v", err)
	}

	tracker := NewEventTracker()
	StartWatcher(watcher, tracker)

	// Write a file to trigger fsnotify.Write
	if err := os.WriteFile(file, []byte("initial write"), 0644); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond) // Let event process

	snapshot := tracker.GetSnapshot()
	if _, ok := snapshot[file]; !ok {
		t.Errorf("Expected Write event for %s, but not found in snapshot", file)
	}
}

func TestStartWatcher_HandlesError(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	tracker := NewEventTracker()
	StartWatcher(watcher, tracker)

	// Inject a fake error (safe because we control the channel in this test)
	go func() {
		watcher.Errors <- fsnotify.ErrEventOverflow
	}()

	// Just confirm test doesn't panic or hang
	time.Sleep(100 * time.Millisecond)
}

func TestStartWatcher_GracefulExitOnClosedChannels(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	tracker := NewEventTracker()
	StartWatcher(watcher, tracker)

	// Closing watcher also closes its channels
	watcher.Close()

	time.Sleep(200 * time.Millisecond) // Wait to allow goroutine to finish
}
