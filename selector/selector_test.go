// selector/selector_test.go

package selector

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/tracker"
)

func TestStartSelector_RespectsDelay_NoEnqueueTooSoon(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "new.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	et := tracker.NewEventTracker()
	et.RecordEvent(file) // timestamp = now

	q := make(chan string, 1)
	ps := NewFileSelector()

	go StartSelector(et, q, ps)

	// Wait > ticker (0.5s) but < hard-coded delay (1s): nothing should arrive.
	// Use 800ms to be safely below 1s on all OSes.
	select {
	case got := <-q:
		t.Fatalf("did not expect enqueue yet, got %s", got)
	case <-time.After(800 * time.Millisecond):
		// OK
	}
}

func TestStartSelector_EnqueuesAfterDelay(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "ready.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Small settle so utils.CheckReadyForProcessing is less likely to reject the file.
	time.Sleep(100 * time.Millisecond)

	et := tracker.NewEventTracker()
	et.RecordEvent(file) // now

	q := make(chan string, 1)
	ps := NewFileSelector()

	go StartSelector(et, q, ps)

	// Wait for: delay (1s) + one tick (0.5s) + cushion
	timeout := 2 * time.Second
	if runtime.GOOS == "windows" {
		timeout = 2500 * time.Millisecond
	}
	select {
	case got := <-q:
		if got != file {
			t.Fatalf("expected %s, got %s", file, got)
		}
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for enqueue")
	}

	if !ps.AlreadyExists(file) {
		t.Fatalf("expected file to be marked as processing")
	}

	// Ensure it doesn't immediately enqueue again (tracker entry is deleted)
	time.Sleep(700 * time.Millisecond)
	select {
	case again := <-q:
		t.Fatalf("did not expect duplicate enqueue, got %s", again)
	default:
	}
}

func TestStartSelector_SkipsIfAlreadyProcessing(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "busy.txt")
	if err := os.WriteFile(file, []byte("y"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Settle slightly for readiness checks
	time.Sleep(100 * time.Millisecond)

	et := tracker.NewEventTracker()
	et.RecordEvent(file) // eligible after 1s

	q := make(chan string, 1)
	ps := NewFileSelector()
	ps.AddFile(file) // mark as already processing

	go StartSelector(et, q, ps)

	// Give it enough time to consider (≥ delay + ≥ one tick)
	timeout := 2 * time.Second
	if runtime.GOOS == "windows" {
		timeout = 2500 * time.Millisecond
	}
	select {
	case got := <-q:
		t.Fatalf("expected skip due to AlreadyExists; got %s", got)
	case <-time.After(timeout):
		// OK: not enqueued
	}

	// NEW: assert it was cleared from tracker
	snap := et.GetSnapshot()
	if _, ok := snap[file]; ok {
		t.Fatalf("expected file to be deleted from tracker when already processing")
	}
}
