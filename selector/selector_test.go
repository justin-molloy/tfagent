package selector

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/utils"
	"github.com/justin-molloy/tfagent/watcher"
)

func TestStartSelectorQueuesFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile.txt")

	// Create test file
	err := os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Ensure file is ready
	if !utils.CheckReadyForProcessing(filePath) {
		t.Fatal("Expected test file to be ready for processing")
	}

	// Set up
	tracker := watcher.NewEventTracker()
	tracker.RecordEvent(filePath)

	fileQueue := make(chan string, 1)
	var processingSet sync.Map
	delay := 1 // seconds

	// Start selector
	go StartSelector(delay, tracker, fileQueue, &processingSet)

	select {
	case queuedFile := <-fileQueue:
		if queuedFile != filePath {
			t.Errorf("Expected file %s to be queued, got %s", filePath, queuedFile)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for file to be queued")
	}
}
